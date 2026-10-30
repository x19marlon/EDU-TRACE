package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"
)

type AIRequest struct {
	Kind string
	GroupID string
	Role Role
	FirstMs int64
	TotalMs int64
	OK bool
	SectionsOK *bool
	QualityOK *bool
	Error string
}

const (
	AIKindFormal = "retro_formal"
	AIKindInformal = "retro_informal"
	AIKindChat = "chat"
	AIKindDraft = "propuesta"
	AIKindAutoDraft = "propuesta_auto"
	AIKindAnalysis = "analisis"
)

func (s *Store) RecordAIRequest(r AIRequest) {
	var first, sections, quality any
	if r.FirstMs > 0 {
		first = r.FirstMs
	}
	if r.SectionsOK != nil {
		sections = *r.SectionsOK
	}
	if r.QualityOK != nil {
		quality = *r.QualityOK
	}
	if _, err := s.db.Exec(`INSERT INTO ai_requests (at, kind, group_id, role, first_token_ms, total_ms, ok, sections_ok, quality_ok, error)
		VALUES (?, ?, (SELECT id FROM class_groups WHERE id = ?), ?, ?, ?, ?, ?, ?, ?)`,
		fmtTime(time.Now()), r.Kind, r.GroupID, string(r.Role), first, r.TotalMs, r.OK, sections, quality, r.Error); err != nil {
		slog.Error("registrar petición a la IA", "error", err)
	}
}

type Thresholds struct {
	ReviewMinutes *float64 `json:"review_minutes"`
	BatchHours *float64 `json:"batch_hours"`
	FeedbackHours *float64 `json:"feedback_hours"`
}

var ErrInvalidThreshold = errors.New("cada umbral debe ser un número mayor que 0 y menor que 10000")

func (t Thresholds) validate() error {
	for _, v := range []*float64{t.ReviewMinutes, t.BatchHours, t.FeedbackHours} {
		if v != nil && (math.IsNaN(*v) || *v <= 0 || *v >= 10000) {
			return ErrInvalidThreshold
		}
	}
	return nil
}

func (s *Store) SetCourseThresholds(teacherID, courseID string, t Thresholds) error {
	if err := t.validate(); err != nil {
		return err
	}
	return affected(s.db.Exec(`UPDATE courses SET review_minutes_max = ?, batch_hours_max = ?, feedback_hours_max = ?
		WHERE id = ? AND teacher_id = ?`, floatArg(t.ReviewMinutes), floatArg(t.BatchHours), floatArg(t.FeedbackHours), courseID, teacherID))
}

type Stat struct {
	Count int `json:"count"`
	AvgMs int64 `json:"avg_ms"`
	P50Ms int64 `json:"p50_ms"`
	P90Ms int64 `json:"p90_ms"`
	MaxMs int64 `json:"max_ms"`
	Over int `json:"over"`
}

func newStat(values []int64, limitMs int64) Stat {
	st := Stat{Count: len(values)}
	if len(values) == 0 {
		return st
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	var sum int64
	for _, v := range values {
		sum += v
		if limitMs > 0 && v > limitMs {
			st.Over++
		}
	}
	rank := func(p float64) int64 { return values[int(math.Ceil(p*float64(len(values))))-1] }
	st.AvgMs, st.P50Ms, st.P90Ms, st.MaxMs = sum/int64(len(values)), rank(0.5), rank(0.9), values[len(values)-1]
	return st
}

func limitMs(v *float64, unit time.Duration) int64 {
	if v == nil {
		return 0
	}
	return int64(*v * float64(unit/time.Millisecond))
}

type BatchMetric struct {
	AssignmentID string `json:"assignment_id"`
	Title string `json:"title"`
	Submissions int `json:"submissions"`
	Done int `json:"done"`
	Failed int `json:"failed"`
	Waiting int `json:"waiting"`
	WorstMs int64 `json:"worst_ms"`
	Finished bool `json:"finished"`
	Over bool `json:"over"`
}

type QueueMetric struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
	Done int `json:"done"`
	Failed int `json:"failed"`
	Skipped int `json:"skipped"`
	Wait Stat `json:"wait"`
	Run Stat `json:"run"`
}

type InstantMetric struct {
	Requests int `json:"requests"`
	Failures int `json:"failures"`
	FirstToken Stat `json:"first_token"`
	Total Stat `json:"total"`
	SectionsChecked int `json:"sections_checked"`
	SectionsOK int `json:"sections_ok"`
	QualityChecked int `json:"quality_checked"`
	QualityOK int `json:"quality_ok"`
	Chats int `json:"chats"`
}

type Decisions struct {
	AI int `json:"ia"`
	Edited int `json:"ia_editada"`
	Teacher int `json:"docente"`
	Discarded int `json:"descartadas"`
}

type GroupMetrics struct {
	CourseID string `json:"course_id"`
	CourseName string `json:"course_name"`
	AIMode string `json:"ai_mode"`
	Thresholds Thresholds `json:"thresholds"`
	ReviewTime Stat `json:"review_time"`
	Turnaround Stat `json:"turnaround"`
	PendingReviews int `json:"pending_reviews"`
	PendingOver int `json:"pending_over"`
	OldestPendingMs int64 `json:"oldest_pending_ms"`
	Queue QueueMetric `json:"queue"`
	Batches []BatchMetric `json:"batches"`
	Instant InstantMetric `json:"instant"`
	Decisions Decisions `json:"decisions"`
	Comprehension Comprehension `json:"comprehension"`
}

func (s *Store) GroupMetrics(teacherID, groupID string) (*GroupMetrics, error) {
	m := &GroupMetrics{Batches: []BatchMetric{}}
	var review, batch, feedback sql.NullFloat64
	err := s.db.QueryRow(`SELECT c.id, c.name, g.ai_mode, c.review_minutes_max, c.batch_hours_max, c.feedback_hours_max
		FROM class_groups g JOIN courses c ON c.id = g.course_id
		WHERE g.id = ? AND c.teacher_id = ?`, groupID, teacherID).Scan(&m.CourseID, &m.CourseName, &m.AIMode, &review, &batch, &feedback)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	m.Thresholds = Thresholds{ReviewMinutes: floatPtr(review), BatchHours: floatPtr(batch), FeedbackHours: floatPtr(feedback)}
	now := time.Now().UTC()
	if err := s.reviewMetrics(m, groupID, now); err != nil {
		return nil, fmt.Errorf("métricas de revisión: %w", err)
	}
	if err := s.queueMetrics(m, groupID, now); err != nil {
		return nil, fmt.Errorf("métricas de la cola: %w", err)
	}
	if err := s.instantMetrics(m, groupID); err != nil {
		return nil, fmt.Errorf("métricas de la IA inmediata: %w", err)
	}
	if m.Comprehension, err = s.comprehension(groupID); err != nil {
		return nil, fmt.Errorf("comprensión: %w", err)
	}
	return m, nil
}

func (s *Store) reviewMetrics(m *GroupMetrics, groupID string, now time.Time) error {
	rows, err := s.db.Query(`SELECT s.created_at, r.status, r.returned_at, r.review_ms, r.feedback_origin, r.ai_discarded_at
		FROM submissions s LEFT JOIN reviews r ON r.submission_id = s.id
		WHERE s.group_id = ? AND s.kind = 'official'`, groupID)
	if err != nil {
		return err
	}
	defer rows.Close()
	feedbackLimit := limitMs(m.Thresholds.FeedbackHours, time.Hour)
	var reviewTimes, turnarounds []int64
	for rows.Next() {
		var created string
		var status, returnedAt, origin, discarded sql.NullString
		var reviewMs sql.NullInt64
		if err := rows.Scan(&created, &status, &returnedAt, &reviewMs, &origin, &discarded); err != nil {
			return err
		}
		createdAt := parseTime(created)
		if discarded.Valid {
			m.Decisions.Discarded++
		}
		if status.String != ReviewReturned || !returnedAt.Valid {
			m.PendingReviews++
			age := now.Sub(createdAt).Milliseconds()
			if age > m.OldestPendingMs {
				m.OldestPendingMs = age
			}
			if feedbackLimit > 0 && age > feedbackLimit {
				m.PendingOver++
			}
			continue
		}
		turnarounds = append(turnarounds, parseTime(returnedAt.String).Sub(createdAt).Milliseconds())
		if reviewMs.Int64 > 0 {
			reviewTimes = append(reviewTimes, reviewMs.Int64)
		}
		switch origin.String {
		case OriginAI:
			m.Decisions.AI++
		case OriginAIEdited:
			m.Decisions.Edited++
		default:
			m.Decisions.Teacher++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	m.ReviewTime = newStat(reviewTimes, limitMs(m.Thresholds.ReviewMinutes, time.Minute))
	m.Turnaround = newStat(turnarounds, feedbackLimit)
	return nil
}

func (s *Store) queueMetrics(m *GroupMetrics, groupID string, now time.Time) error {
	rows, err := s.db.Query(`SELECT j.status, j.enqueued_at, j.started_at, j.finished_at,
			COALESCE(s.assignment_id, ''), COALESCE(a.title, '')
		FROM ai_jobs j
		JOIN submissions s ON s.id = j.submission_id
		LEFT JOIN assignments a ON a.id = s.assignment_id
		WHERE s.group_id = ? ORDER BY j.enqueued_at`, groupID)
	if err != nil {
		return err
	}
	defer rows.Close()
	batchLimit := limitMs(m.Thresholds.BatchHours, time.Hour)
	var waits, runs []int64
	index := map[string]int{}
	for rows.Next() {
		var status, enq, assignmentID, title string
		var started, finished sql.NullString
		if err := rows.Scan(&status, &enq, &started, &finished, &assignmentID, &title); err != nil {
			return err
		}
		enqueued := parseTime(enq)
		i, ok := index[assignmentID]
		if !ok {
			if title == "" {
				title = "Práctica libre"
			}
			i = len(m.Batches)
			index[assignmentID] = i
			m.Batches = append(m.Batches, BatchMetric{AssignmentID: assignmentID, Title: title, Finished: true})
		}
		b := &m.Batches[i]
		b.Submissions++
		delay := now.Sub(enqueued).Milliseconds()
		switch status {
		case JobPending, JobRunning:
			b.Waiting++
			b.Finished = false
			if status == JobPending {
				m.Queue.Pending++
			} else {
				m.Queue.Running++
			}
		case JobDone:
			b.Done++
			m.Queue.Done++
		case JobFailed:
			b.Failed++
			m.Queue.Failed++
		default:
			m.Queue.Skipped++
		}
		if finished.Valid && status != JobPending && status != JobRunning {
			delay = parseTime(finished.String).Sub(enqueued).Milliseconds()
		}
		if status != JobSkipped && delay > b.WorstMs {
			b.WorstMs = delay
		}
		if status == JobDone && started.Valid && finished.Valid {
			startedAt := parseTime(started.String)
			waits = append(waits, startedAt.Sub(enqueued).Milliseconds())
			runs = append(runs, parseTime(finished.String).Sub(startedAt).Milliseconds())
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range m.Batches {
		m.Batches[i].Over = batchLimit > 0 && m.Batches[i].WorstMs > batchLimit
	}
	m.Queue.Wait, m.Queue.Run = newStat(waits, 0), newStat(runs, 0)
	return nil
}

func (s *Store) instantMetrics(m *GroupMetrics, groupID string) error {
	rows, err := s.db.Query(`SELECT kind, first_token_ms, total_ms, ok, sections_ok, quality_ok FROM ai_requests
		WHERE group_id = ? AND kind IN (?, ?, ?)`, groupID, AIKindFormal, AIKindInformal, AIKindChat)
	if err != nil {
		return err
	}
	defer rows.Close()
	var firsts, totals []int64
	for rows.Next() {
		var kind string
		var first sql.NullInt64
		var total int64
		var ok bool
		var sections, quality sql.NullBool
		if err := rows.Scan(&kind, &first, &total, &ok, &sections, &quality); err != nil {
			return err
		}
		if kind == AIKindChat {
			m.Instant.Chats++
			continue
		}
		m.Instant.Requests++
		if !ok {
			m.Instant.Failures++
			continue
		}
		totals = append(totals, total)
		if first.Valid {
			firsts = append(firsts, first.Int64)
		}
		if sections.Valid {
			m.Instant.SectionsChecked++
			if sections.Bool {
				m.Instant.SectionsOK++
			}
		}
		if quality.Valid {
			m.Instant.QualityChecked++
			if quality.Bool {
				m.Instant.QualityOK++
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	m.Instant.FirstToken, m.Instant.Total = newStat(firsts, 0), newStat(totals, 0)
	return nil
}
