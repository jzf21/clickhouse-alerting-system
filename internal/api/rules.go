package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListRules(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rules == nil {
		rules = []model.AlertRule{}
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) getRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rule, err := s.store.GetRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	// Decode into raw map to detect if "enabled" was explicitly set
	var raw map[string]json.RawMessage
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	var rule model.AlertRule
	if err := json.Unmarshal(body, &rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if rule.Name == "" || rule.Query == "" || rule.Column == "" || rule.Operator == "" {
		writeError(w, http.StatusBadRequest, "name, query, column, and operator are required")
		return
	}

	rule.ID = uuid.New().String()
	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	// Default enabled to true if not explicitly set
	if _, ok := raw["enabled"]; !ok {
		rule.Enabled = true
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	if rule.EvalInterval <= 0 {
		rule.EvalInterval = 60
	}
	if rule.Labels == nil {
		rule.Labels = json.RawMessage(`{}`)
	}
	if rule.Annotations == nil {
		rule.Annotations = json.RawMessage(`{}`)
	}
	if rule.ChannelIDs == nil {
		rule.ChannelIDs = json.RawMessage(`[]`)
	}

	if err := s.store.CreateRule(r.Context(), rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.store.GetRule(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var update model.AlertRule
	if err := decodeJSON(r, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Apply updates, keeping existing values for unset fields
	update.ID = existing.ID
	update.CreatedAt = existing.CreatedAt
	update.UpdatedAt = time.Now().UTC()

	if update.Name == "" {
		update.Name = existing.Name
	}
	if update.Query == "" {
		update.Query = existing.Query
	}
	if update.Column == "" {
		update.Column = existing.Column
	}
	if update.Operator == "" {
		update.Operator = existing.Operator
	}
	if update.Severity == "" {
		update.Severity = existing.Severity
	}
	if update.Labels == nil {
		update.Labels = existing.Labels
	}
	if update.Annotations == nil {
		update.Annotations = existing.Annotations
	}
	if update.ChannelIDs == nil {
		update.ChannelIDs = existing.ChannelIDs
	}

	if err := s.store.UpdateRule(r.Context(), update); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, update)
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteRule(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
