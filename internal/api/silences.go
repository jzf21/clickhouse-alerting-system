package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

func (s *Server) listSilences(w http.ResponseWriter, r *http.Request) {
	connID := r.URL.Query().Get("connection_id")
	var silences []model.Silence
	var err error
	if connID != "" {
		silences, err = s.store.ListSilencesByConnection(r.Context(), connID)
	} else {
		silences, err = s.store.ListSilences(r.Context())
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if silences == nil {
		silences = []model.Silence{}
	}
	writeJSON(w, http.StatusOK, silences)
}

func (s *Server) createSilence(w http.ResponseWriter, r *http.Request) {
	var silence model.Silence
	if err := decodeJSON(r, &silence); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if silence.EndsAt.IsZero() {
		writeError(w, http.StatusBadRequest, "ends_at is required")
		return
	}

	silence.ID = uuid.New().String()
	silence.CreatedAt = time.Now().UTC()

	if silence.StartsAt.IsZero() {
		silence.StartsAt = time.Now().UTC()
	}
	if silence.Matchers == nil {
		silence.Matchers = json.RawMessage(`[]`)
	}

	if err := s.store.CreateSilence(r.Context(), silence); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, silence)
}

func (s *Server) deleteSilence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteSilence(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
