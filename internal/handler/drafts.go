package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
)

const (
	draftRetryAfter = 2 * time.Minute
	draftIdleCheck = 30 * time.Second
	pingCacheFor = 30 * time.Second
)

type DraftQueue struct {
	store *auth.Store
	reviews *ReviewsHandler
	timeout time.Duration
	wake chan struct{}
	mu sync.Mutex
	lastErr string
	lastErrAt time.Time
	lastOKAt time.Time
	pingAt time.Time
	pingOK bool
}

func NewDraftQueue(store *auth.Store, ollama *ai.OllamaClient, timeout time.Duration) *DraftQueue {
	return &DraftQueue{store: store, reviews: NewReviewsHandler(store, ollama), timeout: timeout, wake: make(chan struct{}, 1)}
}

func (q *DraftQueue) Wake() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *DraftQueue) Run(ctx context.Context) {
	if q.reviews.ollama == nil {
		slog.Warn("cola de propuestas detenida: Ollama no está disponible")
		return
	}
	if err := q.store.ResetRunningJobs(); err != nil {
		slog.Error("reiniciar la cola de propuestas", "error", err)
	}
	ticker := time.NewTicker(draftIdleCheck)
	defer ticker.Stop()
	for {
		for ctx.Err() == nil && q.process(ctx) {
		}
		select {
		case <-ctx.Done():
			return
		case <-q.wake:
		case <-ticker.C:
		}
	}
}

func (q *DraftQueue) process(ctx context.Context) bool {
	job, err := q.store.ClaimJob(draftRetryAfter)
	if err != nil {
		slog.Error("tomar trabajo de la cola", "error", err)
		return false
	}
	if job == nil {
		return false
	}
	needed, err := q.store.NeedsAIDraft(job.SubmissionID)
	if err != nil {
		q.store.FinishJob(job.SubmissionID, auth.JobFailed, auth.JobError(err), false)
		return true
	}
	if !needed {
		q.store.FinishJob(job.SubmissionID, auth.JobSkipped, "", true)
		return true
	}
	jobCtx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()
	start := time.Now()
	_, err = q.reviews.generate(jobCtx, job.TeacherID, job.SubmissionID, true)
	q.store.RecordAIRequest(auth.AIRequest{Kind: auth.AIKindAutoDraft, GroupID: q.store.SubmissionGroup(job.SubmissionID),
		Role: auth.SystemActorRole, TotalMs: time.Since(start).Milliseconds(), OK: err == nil, Error: errText(err),
		Model: q.reviews.ollama.Model(), PromptVersion: q.reviews.ollama.PromptVersion()})
	if ctx.Err() != nil {
		q.store.FinishJob(job.SubmissionID, auth.JobPending, "", false)
		return false
	}
	if errors.Is(err, auth.ErrAlreadyReturned) {
		q.store.FinishJob(job.SubmissionID, auth.JobSkipped, "", true)
		return true
	}
	if err != nil {
		final := errors.Is(err, ai.ErrPromptTooLong) || errors.Is(err, auth.ErrNotFound) || errors.Is(err, errNotOfficial)
		slog.Error("propuesta automática fallida", "submission_id", job.SubmissionID, "intento", job.Attempts, "error", err)
		q.store.FinishJob(job.SubmissionID, auth.JobFailed, auth.JobError(err), final)
		q.mu.Lock()
		q.lastErr, q.lastErrAt = auth.JobError(err), time.Now()
		q.mu.Unlock()
		return true
	}
	q.store.FinishJob(job.SubmissionID, auth.JobDone, "", false)
	q.mu.Lock()
	q.lastOKAt = time.Now()
	q.mu.Unlock()
	return true
}

type aiStatus struct {
	Available bool `json:"available"`
	LastError string `json:"last_error,omitempty"`
	LastErrorAt *time.Time `json:"last_error_at,omitempty"`
	LastOKAt *time.Time `json:"last_ok_at,omitempty"`
	*auth.QueueOverview
}

func (q *DraftQueue) available(ctx context.Context) bool {
	if q.reviews.ollama == nil {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if time.Since(q.pingAt) < pingCacheFor {
		return q.pingOK
	}
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	q.pingOK, q.pingAt = q.reviews.ollama.Ping(pctx) == nil, time.Now()
	return q.pingOK
}

func (q *DraftQueue) Status(w http.ResponseWriter, r *http.Request) {
	overview, err := q.store.TeacherQueue(auth.UserFrom(r.Context()).ID)
	if err != nil {
		writeStoreErr(w, err, "consultar la cola de la IA")
		return
	}
	st := aiStatus{Available: q.available(r.Context()), QueueOverview: overview}
	q.mu.Lock()
	if q.lastErr != "" {
		at := q.lastErrAt
		st.LastError, st.LastErrorAt = q.lastErr, &at
	}
	if !q.lastOKAt.IsZero() {
		at := q.lastOKAt
		st.LastOKAt = &at
	}
	q.mu.Unlock()
	writeJSON(w, http.StatusOK, st)
}
