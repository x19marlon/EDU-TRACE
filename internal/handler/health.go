package handler

import (
	"net/http"
	"syscall"
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

func freePercent(dir string) (float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	if st.Blocks == 0 {
		return 0, nil
	}
	return 100 * float64(st.Bavail) / float64(st.Blocks), nil
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
