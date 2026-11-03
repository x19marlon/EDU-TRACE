package handler

import (
	"net/http"
	"time"

	"github.com/marlon/edu-trace/internal/auth"
)

const queueStallAfter = 30 * time.Minute

type HealthHandler struct {
	store *auth.Store
	drafts *DraftQueue
	dataDir string
	minFree int
}

func NewHealthHandler(store *auth.Store, drafts *DraftQueue, dataDir string, minFree int) *HealthHandler {
	return &HealthHandler{store: store, drafts: drafts, dataDir: dataDir, minFree: minFree}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	checks := map[string]bool{"base": h.store.Ping() == nil, "ia": h.drafts.available(r.Context())}
	free, err := freePercent(h.dataDir)
	checks["disco"] = err == nil && free >= float64(h.minFree)
	stalled, err := h.store.QueueStalled(time.Now(), queueStallAfter)
	checks["cola"] = err == nil && !stalled
	status, code := "ok", http.StatusOK
	for _, ok := range checks {
		if !ok {
			status = "degradado"
		}
	}
	if !checks["base"] {
		status, code = "caido", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"status": status, "checks": checks})
}
