package httpapi

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/Sonde/apps/api/internal/monitors"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type MonitorInput struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Target          string `json:"target"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	ExpectedStatus  int    `json:"expected_status"`
	ExpectedKeyword string `json:"expected_keyword"`
	Enabled         bool   `json:"enabled"`
}

type MonitorResponse struct {
	ID              int64     `json:"id"`
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Target          string    `json:"target"`
	IntervalSeconds int       `json:"interval_seconds"`
	TimeoutSeconds  int       `json:"timeout_seconds"`
	ExpectedStatus  int       `json:"expected_status"`
	ExpectedKeyword string    `json:"expected_keyword"`
	Enabled         bool      `json:"enabled"`
	PushToken       *string   `json:"push_token,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func newMonitorResponse(m *schemas.Monitor) MonitorResponse {
	return MonitorResponse{
		ID:              m.ID,
		Slug:            m.Slug,
		Name:            m.Name,
		Type:            m.Type,
		Target:          m.Target,
		IntervalSeconds: m.IntervalSeconds,
		TimeoutSeconds:  m.TimeoutSeconds,
		ExpectedStatus:  m.ExpectedStatus,
		ExpectedKeyword: m.ExpectedKeyword,
		Enabled:         m.Enabled,
		PushToken:       m.PushToken,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func (s *Service) ownerID(r *http.Request) (int64, error) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		return 0, errors.Unauthorized("missing auth")
	}
	return identity.UserID, nil
}

func (s *Service) createMonitor(w http.ResponseWriter, r *http.Request) {
	var input MonitorInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	monitor, err := s.monitors.Create(r.Context(), ownerID, monitors.Input{
		Slug:            input.Slug,
		Name:            input.Name,
		Type:            input.Type,
		Target:          input.Target,
		IntervalSeconds: input.IntervalSeconds,
		TimeoutSeconds:  input.TimeoutSeconds,
		ExpectedStatus:  input.ExpectedStatus,
		ExpectedKeyword: input.ExpectedKeyword,
		Enabled:         true,
	})
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, newMonitorResponse(monitor))
}

func (s *Service) listMonitors(w http.ResponseWriter, r *http.Request) {
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	found, err := s.monitors.List(r.Context(), ownerID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out := make([]MonitorResponse, 0, len(found))
	for i := range found {
		out = append(out, newMonitorResponse(&found[i]))
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) getMonitor(w http.ResponseWriter, r *http.Request) {
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	monitor, err := s.monitors.Get(r.Context(), ownerID, id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, newMonitorResponse(monitor))
}

func (s *Service) updateMonitor(w http.ResponseWriter, r *http.Request) {
	var input MonitorInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	monitor, err := s.monitors.Update(r.Context(), ownerID, id, monitors.Input{
		Slug:            input.Slug,
		Name:            input.Name,
		Type:            input.Type,
		Target:          input.Target,
		IntervalSeconds: input.IntervalSeconds,
		TimeoutSeconds:  input.TimeoutSeconds,
		ExpectedStatus:  input.ExpectedStatus,
		ExpectedKeyword: input.ExpectedKeyword,
		Enabled:         input.Enabled,
	})
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, newMonitorResponse(monitor))
}

func (s *Service) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := s.monitors.Delete(r.Context(), ownerID, id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
