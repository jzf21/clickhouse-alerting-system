package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jozef/clickhouse-alerting-system/internal/notifier"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

type Server struct {
	store      store.Store
	dispatcher *notifier.Dispatcher
	mux        *http.ServeMux
}

func NewServer(st store.Store, dispatcher *notifier.Dispatcher) *Server {
	s := &Server{
		store:      st,
		dispatcher: dispatcher,
		mux:        http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return withMiddleware(s.mux)
}

func (s *Server) routes() {
	// Rules
	s.mux.HandleFunc("GET /api/rules", s.listRules)
	s.mux.HandleFunc("POST /api/rules", s.createRule)
	s.mux.HandleFunc("GET /api/rules/{id}", s.getRule)
	s.mux.HandleFunc("PUT /api/rules/{id}", s.updateRule)
	s.mux.HandleFunc("DELETE /api/rules/{id}", s.deleteRule)

	// Channels
	s.mux.HandleFunc("GET /api/channels", s.listChannels)
	s.mux.HandleFunc("POST /api/channels", s.createChannel)
	s.mux.HandleFunc("GET /api/channels/{id}", s.getChannel)
	s.mux.HandleFunc("PUT /api/channels/{id}", s.updateChannel)
	s.mux.HandleFunc("DELETE /api/channels/{id}", s.deleteChannel)
	s.mux.HandleFunc("POST /api/channels/{id}/test", s.testChannel)

	// Silences
	s.mux.HandleFunc("GET /api/silences", s.listSilences)
	s.mux.HandleFunc("POST /api/silences", s.createSilence)
	s.mux.HandleFunc("DELETE /api/silences/{id}", s.deleteSilence)

	// Alerts
	s.mux.HandleFunc("GET /api/alerts", s.listAlerts)
	s.mux.HandleFunc("GET /api/alerts/history", s.listAlertHistory)

	// Dashboard
	s.mux.HandleFunc("GET /", serveDashboard)
	s.mux.HandleFunc("GET /app.js", serveAppJS)
	s.mux.HandleFunc("GET /style.css", serveStyleCSS)
}

func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Recovery
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		slog.Debug("request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
