package api

import (
	"net/http"
	"strconv"

	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.store.ListAlertStates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if alerts == nil {
		alerts = []model.AlertWithRule{}
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (s *Server) listAlertHistory(w http.ResponseWriter, r *http.Request) {
	ruleID := r.URL.Query().Get("rule_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 50
	}

	events, err := s.store.ListEvents(r.Context(), ruleID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []model.AlertEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}
