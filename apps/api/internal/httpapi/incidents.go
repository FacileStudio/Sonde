package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type IncidentResponse struct {
	ID         int64      `json:"id"`
	MonitorID  int64      `json:"monitor_id"`
	OpenedAt   time.Time  `json:"opened_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	Cause      string     `json:"cause"`
}

func newIncidentResponse(i *schemas.Incident) IncidentResponse {
	return IncidentResponse{
		ID:         i.ID,
		MonitorID:  i.MonitorID,
		OpenedAt:   i.OpenedAt,
		ResolvedAt: i.ResolvedAt,
		Cause:      i.Cause,
	}
}

type UptimeResponse struct {
	Window string   `json:"window"`
	Uptime *float64 `json:"uptime"`
	Total  int64    `json:"total"`
	Failed int64    `json:"failed"`
}

var uptimeWindows = map[string]bool{"24h": true, "7d": true, "30d": true, "90d": true}

func (s *Service) monitorUptime(w http.ResponseWriter, r *http.Request) {
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "24h"
	}
	if !uptimeWindows[window] {
		httpjson.WriteError(w, errors.Invalid("window must be one of 24h, 7d, 30d, 90d"))
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if _, err := s.monitors.Get(r.Context(), ownerID, id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	stats, err := s.uptime.Summary(r.Context(), id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	for _, stat := range stats {
		if stat.Window == window {
			httpjson.WriteJSON(w, http.StatusOK, UptimeResponse{
				Window: stat.Window,
				Uptime: stat.Uptime,
				Total:  stat.Total,
				Failed: stat.Failed,
			})
			return
		}
	}
	httpjson.WriteError(w, errors.Internal("uptime window missing from summary", nil))
}

func (s *Service) listIncidents(w http.ResponseWriter, r *http.Request) {
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	query := s.db.WithContext(r.Context()).
		Joins("JOIN monitors ON monitors.id = incidents.monitor_id").
		Where("monitors.owner_id = ?", ownerID).
		Order("incidents.opened_at DESC").
		Limit(200)
	if raw := r.URL.Query().Get("monitor_id"); raw != "" {
		id, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || id < 1 {
			httpjson.WriteError(w, errors.Invalid("monitor_id must be a positive integer"))
			return
		}
		query = query.Where("incidents.monitor_id = ?", id)
	}
	var found []schemas.Incident
	if err := query.Find(&found).Error; err != nil {
		httpjson.WriteError(w, errors.Internal("could not list incidents", err))
		return
	}
	out := make([]IncidentResponse, 0, len(found))
	for i := range found {
		out = append(out, newIncidentResponse(&found[i]))
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) monitorIncidents(w http.ResponseWriter, r *http.Request) {
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
	if _, err := s.monitors.Get(r.Context(), ownerID, id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	var found []schemas.Incident
	if dbErr := s.db.WithContext(r.Context()).
		Where("monitor_id = ?", id).
		Order("opened_at DESC").
		Limit(200).
		Find(&found).Error; dbErr != nil {
		httpjson.WriteError(w, errors.Internal("could not list incidents", dbErr))
		return
	}
	out := make([]IncidentResponse, 0, len(found))
	for i := range found {
		out = append(out, newIncidentResponse(&found[i]))
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}
