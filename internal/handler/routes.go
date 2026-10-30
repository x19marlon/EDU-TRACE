package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/marlon/edu-trace/internal/ai"
	"github.com/marlon/edu-trace/internal/auth"
	"github.com/marlon/edu-trace/internal/compiler"
	"github.com/marlon/edu-trace/internal/config"
)

func NewRouter(cfg *config.Config, comp *compiler.Compiler, ollama *ai.OllamaClient, store *auth.Store, drafts *DraftQueue) http.Handler {
	mux := http.NewServeMux()
	authH := NewAuthHandler(store, cfg)
	compileFlight := auth.NewSingleFlight()
	feedbackFlight := auth.NewSingleFlight()

	// Health check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth
	mux.HandleFunc("POST /api/auth/register", authH.Register)
	mux.HandleFunc("POST /api/auth/login", authH.Login)
	mux.HandleFunc("POST /api/auth/logout", authH.Logout)
	mux.Handle("GET /api/auth/me", auth.RequireAuth(store, http.HandlerFunc(authH.Me)))
	mux.HandleFunc("POST /api/auth/recover", authH.Recover)
	mux.Handle("POST /api/auth/password", auth.RequireAuth(store, http.HandlerFunc(authH.ChangePassword)))
	mux.Handle("POST /api/auth/recovery-code", auth.RequireAuth(store, http.HandlerFunc(authH.RecoveryCode)))
	mux.Handle("POST /api/auth/consent", auth.RequireAuth(store, http.HandlerFunc(authH.Consent)))
	mux.Handle("PATCH /api/auth/profile", auth.RequireAuth(store, http.HandlerFunc(authH.UpdateProfile)))
	mux.Handle("GET /api/auth/export", auth.RequireAuth(store, http.HandlerFunc(authH.Export)))
	mux.Handle("POST /api/auth/delete", auth.RequireAuth(store, http.HandlerFunc(authH.DeleteAccount)))

	// docente y sus clases
	classes := NewClassesHandler(store, comp, drafts)
	teacher := func(f http.HandlerFunc) http.Handler { return auth.RequireRole(store, auth.RoleTeacher, f) }
	student := func(h http.Handler) http.Handler { return auth.RequireRole(store, auth.RoleStudent, h) }
	mux.Handle("GET /api/teacher/courses", teacher(classes.Courses))
	mux.Handle("POST /api/teacher/courses", teacher(classes.CreateCourse))
	mux.Handle("POST /api/teacher/courses/{id}/groups", teacher(classes.CreateGroup))
	mux.Handle("GET /api/teacher/groups/{id}", teacher(classes.Group))
	mux.Handle("GET /api/teacher/groups/{id}/submissions", teacher(classes.GroupSubmissions))
	mux.Handle("GET /api/teacher/submissions/{id}", teacher(classes.Submission))
	mux.Handle("POST /api/teacher/groups/{id}/assignments", teacher(classes.CreateAssignment))
	mux.Handle("GET /api/teacher/groups/{id}/assignments", teacher(classes.GroupAssignments))
	mux.Handle("PATCH /api/teacher/courses/{id}", teacher(classes.RenameCourse))
	mux.Handle("DELETE /api/teacher/courses/{id}", teacher(classes.DeleteCourse))
	mux.Handle("PATCH /api/teacher/groups/{id}", teacher(classes.RenameGroup))
	mux.Handle("DELETE /api/teacher/groups/{id}", teacher(classes.DeleteGroup))
	mux.Handle("POST /api/teacher/groups/{id}/join-code", teacher(classes.RegenerateJoinCode))
	mux.Handle("DELETE /api/teacher/groups/{id}/students/{student}", teacher(classes.RemoveStudent))
	mux.Handle("POST /api/teacher/groups/{id}/monitors", teacher(classes.AddMonitor))
	mux.Handle("DELETE /api/teacher/groups/{id}/monitors/{monitor}", teacher(classes.RemoveMonitor))
	mux.Handle("PUT /api/teacher/assignments/{id}", teacher(classes.UpdateAssignment))
	mux.Handle("DELETE /api/teacher/assignments/{id}", teacher(classes.DeleteAssignment))
	mux.Handle("GET /api/teacher/courses/{id}/raes", teacher(classes.CourseRAEs))
	mux.Handle("POST /api/teacher/courses/{id}/raes", teacher(classes.CreateRAE))
	mux.Handle("PATCH /api/teacher/raes/{id}", teacher(classes.UpdateRAE))
	mux.Handle("DELETE /api/teacher/raes/{id}", teacher(classes.DeleteRAE))
	mux.Handle("GET /api/teacher/groups/{id}/rae-matrix", teacher(classes.RAEMatrix))
	mux.Handle("PATCH /api/teacher/groups/{id}/ai-mode", teacher(classes.SetAIMode))
	mux.Handle("GET /api/teacher/groups/{id}/metrics", teacher(classes.Metrics))
	mux.Handle("GET /api/teacher/groups/{id}/audit", teacher(classes.GroupAudit))
	mux.Handle("POST /api/teacher/groups/{id}/ai-queue/retry", teacher(classes.RetryDrafts))
	mux.Handle("PUT /api/teacher/courses/{id}/thresholds", teacher(classes.SetThresholds))
	mux.Handle("GET /api/teacher/ai-status", teacher(drafts.Status))
	mux.Handle("GET /api/teacher/groups/{id}/errors", teacher(classes.FrequentErrors))
	mux.Handle("GET /api/teacher/groups/{id}/export/notas.csv", teacher(classes.ExportGrades))
	mux.Handle("GET /api/teacher/groups/{id}/export/revisiones.csv", teacher(classes.ExportReviews))
	mux.Handle("GET /api/teacher/questions", teacher(classes.OpenQuestions))
	mux.Handle("POST /api/teacher/questions/{id}/answer", teacher(classes.AnswerQuestion))
	mux.Handle("POST /api/teacher/submissions/{id}/tests", auth.RequireRole(store, auth.RoleTeacher, compileFlight.Wrap(http.HandlerFunc(classes.RerunTests))))
	reviews := NewReviewsHandler(store, ollama)
	mux.Handle("GET /api/teacher/submissions/{id}/review", teacher(reviews.Get))
	mux.Handle("PUT /api/teacher/submissions/{id}/review", teacher(reviews.Save))
	mux.Handle("POST /api/teacher/submissions/{id}/review/discard", teacher(reviews.Discard))
	mux.Handle("GET /api/teacher/submissions/{id}/audit", teacher(reviews.Audit))

	annotations := NewAnnotationsHandler(store)
	mux.Handle("POST /api/teacher/submissions/{id}/annotations", teacher(annotations.Create))
	mux.Handle("DELETE /api/teacher/annotations/{id}", teacher(annotations.Delete))

	monitors := NewMonitorHandler(store)
	monitor := func(f http.HandlerFunc) http.Handler { return auth.RequireRole(store, auth.RoleMonitor, f) }
	mux.Handle("DELETE /api/teacher/monitor-feedback/{id}", teacher(monitors.DeleteFeedback))
	mux.Handle("GET /api/monitor/groups", monitor(monitors.Groups))
	mux.Handle("GET /api/monitor/groups/{id}", monitor(monitors.Group))
	mux.Handle("GET /api/monitor/groups/{id}/submissions", monitor(monitors.Submissions))
	mux.Handle("GET /api/monitor/groups/{id}/assignments", monitor(monitors.Assignments))
	mux.Handle("GET /api/monitor/submissions/{id}", monitor(monitors.Submission))
	mux.Handle("POST /api/monitor/submissions/{id}/feedback", monitor(monitors.AddFeedback))
	mux.Handle("DELETE /api/monitor/feedback/{id}", monitor(monitors.DeleteFeedback))
	mux.Handle("POST /api/monitor/submissions/{id}/annotations", monitor(annotations.Create))
	mux.Handle("DELETE /api/monitor/annotations/{id}", monitor(annotations.Delete))
	mux.Handle("POST /api/monitor/submissions/{id}/annotations/send", monitor(annotations.Send))
	mux.Handle("POST /api/monitor/questions/{id}/answer", monitor(classes.AnswerQuestion))

	//  adjuntos de talleres
	mux.Handle("GET /api/assignments/{id}/files/{file}", auth.RequireAuth(store, http.HandlerFunc(classes.AssignmentFile)))

	// Clases y envios del estudiante
	mux.Handle("GET /api/student/groups", student(http.HandlerFunc(classes.MyGroups)))
	mux.Handle("POST /api/student/groups/join", student(http.HandlerFunc(classes.Join)))
	mux.Handle("GET /api/student/submissions", student(http.HandlerFunc(classes.MySubmissions)))
	mux.Handle("GET /api/student/submissions/{id}", student(http.HandlerFunc(classes.MySubmission)))
	mux.Handle("GET /api/student/history", student(http.HandlerFunc(classes.MyHistory)))
	mux.Handle("GET /api/student/assignments", student(http.HandlerFunc(classes.MyAssignments)))
	mux.Handle("POST /api/student/submissions", student(compileFlight.Wrap(http.HandlerFunc(classes.Submit))))
	mux.Handle("POST /api/student/submissions/{id}/questions", student(http.HandlerFunc(classes.AskQuestion)))
	mux.Handle("POST /api/student/ratings", student(http.HandlerFunc(classes.RateFeedback)))

	// Endpoint de compilacion
	mux.Handle("POST /api/compile", auth.RequireAuth(store, compileFlight.Wrap(NewCompileHandler(comp, store))))

	// Endpoint de feedback y chat pedagógico
	if ollama != nil {
		mux.Handle("POST /api/feedback", auth.RequireAuth(store, feedbackFlight.Wrap(NewFeedbackHandler(ollama, cfg.OllamaModel, store))))
		mux.Handle("POST /api/chat", auth.RequireAuth(store, feedbackFlight.Wrap(NewChatHandler(ollama, cfg.OllamaModel, store))))
		analysisHandler := auth.RequireRole(store, auth.RoleTeacher, feedbackFlight.Wrap(NewAnalysisHandler(ollama, cfg.OllamaModel, store)))
		mux.Handle("POST /api/teacher/groups/{id}/students/{student}/analysis", analysisHandler)
		mux.Handle("POST /api/teacher/submissions/{id}/review/draft",
			auth.RequireRole(store, auth.RoleTeacher, feedbackFlight.Wrap(http.HandlerFunc(reviews.Draft))))
		slog.Info("feedback endpoint enabled", "model", cfg.OllamaModel)
	} else {
		unavailable := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Ollama no está configurado. La retroalimentación con IA no está disponible.",
			})
		})
		mux.Handle("POST /api/teacher/submissions/{id}/review/draft", auth.RequireRole(store, auth.RoleTeacher, unavailable))
		mux.Handle("POST /api/feedback", auth.RequireAuth(store, unavailable))
		mux.Handle("POST /api/chat", auth.RequireAuth(store, unavailable))
		mux.Handle("POST /api/teacher/groups/{id}/students/{student}/analysis", auth.RequireRole(store, auth.RoleTeacher, unavailable))
		slog.Warn("feedback endpoint disabled — Ollama client not available")
	}

	// Middleware
	var handler http.Handler = mux
	handler = compressMiddleware(handler)
	handler = corsMiddleware(handler, cfg.AllowedOrigins)
	handler = loggingMiddleware(handler)

	return handler
}

func corsMiddleware(next http.Handler, allowedOrigins string) http.Handler {
	allowed := map[string]bool{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		allowed[strings.TrimSpace(o)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && !allowed[origin] && r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "origen no permitido"})
			return
		}

		if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs each request with slog.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}
