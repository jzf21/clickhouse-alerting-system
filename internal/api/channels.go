package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	connID := r.URL.Query().Get("connection_id")
	var channels []model.NotificationChannel
	var err error
	if connID != "" {
		channels, err = s.store.ListChannelsByConnection(r.Context(), connID)
	} else {
		channels, err = s.store.ListChannels(r.Context())
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if channels == nil {
		channels = []model.NotificationChannel{}
	}
	writeJSON(w, http.StatusOK, channels)
}

func (s *Server) getChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, err := s.store.GetChannel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (s *Server) createChannel(w http.ResponseWriter, r *http.Request) {
	var raw map[string]json.RawMessage
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	var ch model.NotificationChannel
	if err := json.Unmarshal(body, &ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if ch.Name == "" || ch.Type == "" {
		writeError(w, http.StatusBadRequest, "name and type are required")
		return
	}
	if ch.Type != "slack" && ch.Type != "webhook" {
		writeError(w, http.StatusBadRequest, "type must be 'slack' or 'webhook'")
		return
	}

	ch.ID = uuid.New().String()
	now := time.Now().UTC()
	ch.CreatedAt = now
	ch.UpdatedAt = now

	if _, ok := raw["enabled"]; !ok {
		ch.Enabled = true
	}
	if ch.Config == nil {
		ch.Config = json.RawMessage(`{}`)
	}

	if err := s.store.CreateChannel(r.Context(), ch); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

func (s *Server) updateChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.store.GetChannel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var update model.NotificationChannel
	if err := decodeJSON(r, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	update.ID = existing.ID
	update.CreatedAt = existing.CreatedAt
	update.UpdatedAt = time.Now().UTC()

	if update.Name == "" {
		update.Name = existing.Name
	}
	if update.Type == "" {
		update.Type = existing.Type
	}
	if update.Config == nil {
		update.Config = existing.Config
	}
	if update.ConnectionID == nil {
		update.ConnectionID = existing.ConnectionID
	}

	if err := s.store.UpdateChannel(r.Context(), update); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, update)
}

func (s *Server) deleteChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteChannel(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testChannel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch, err := s.store.GetChannel(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err := s.dispatcher.SendTest(r.Context(), ch); err != nil {
		writeError(w, http.StatusInternalServerError, "test failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
