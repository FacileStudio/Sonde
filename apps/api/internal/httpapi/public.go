package httpapi

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type PublicIncident struct {
	ID       int64     `json:"id"`
	OpenedAt time.Time `json:"opened_at"`
	Cause    string    `json:"cause"`
}

type PublicMonitor struct {
	ID            int64            `json:"id"`
	Slug          string           `json:"slug"`
	Name          string           `json:"name"`
	CurrentStatus string           `json:"current_status"`
	Uptime24h     *float64         `json:"uptime_24h"`
	Uptime7d      *float64         `json:"uptime_7d"`
	OpenIncidents []PublicIncident `json:"open_incidents"`
}

type PublicStatusPage struct {
	Slug     string          `json:"slug"`
	Title    string          `json:"title"`
	Monitors []PublicMonitor `json:"monitors"`
}

func (s *Service) publicStatusPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var page schemas.StatusPage
	if err := s.db.WithContext(r.Context()).Where("slug = ?", slug).First(&page).Error; err != nil {
		httpjson.WriteError(w, errors.NotFound("status page not found"))
		return
	}

	var members []schemas.StatusPageMonitor
	if err := s.db.WithContext(r.Context()).
		Where("page_id = ?", page.ID).
		Order("position").
		Find(&members).Error; err != nil {
		httpjson.WriteError(w, errors.Internal("could not load page membership", err))
		return
	}

	out := PublicStatusPage{Slug: page.Slug, Title: page.Title, Monitors: make([]PublicMonitor, 0, len(members))}
	for _, member := range members {
		line, err := s.publicMonitorLine(r, member.MonitorID)
		if err != nil {
			httpjson.WriteError(w, err)
			return
		}
		out.Monitors = append(out.Monitors, *line)
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

// current_status is up or down from the newest raw check; a monitor that has
// never been checked is unknown, never a fabricated green.
func (s *Service) publicMonitorLine(r *http.Request, monitorID int64) (*PublicMonitor, error) {
	var monitor schemas.Monitor
	if err := s.db.WithContext(r.Context()).First(&monitor, monitorID).Error; err != nil {
		return nil, errors.Internal("could not load page monitor", err)
	}

	status := "unknown"
	var latest schemas.CheckRaw
	if err := s.db.WithContext(r.Context()).
		Where("monitor_id = ?", monitorID).
		Order("created_at DESC").
		First(&latest).Error; err == nil {
		if latest.OK {
			status = "up"
		} else {
			status = "down"
		}
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.Internal("could not load the latest check", err)
	}

	stats, err := s.uptime.Summary(r.Context(), monitorID)
	if err != nil {
		return nil, err
	}
	window := map[string]*float64{}
	for _, stat := range stats {
		value := stat.Uptime
		window[stat.Window] = value
	}

	var incidents []schemas.Incident
	if dbErr := s.db.WithContext(r.Context()).
		Where("monitor_id = ? AND resolved_at IS NULL", monitorID).
		Order("opened_at DESC").
		Find(&incidents).Error; dbErr != nil {
		return nil, errors.Internal("could not load open incidents", dbErr)
	}
	open := make([]PublicIncident, 0, len(incidents))
	for _, incident := range incidents {
		open = append(open, PublicIncident{ID: incident.ID, OpenedAt: incident.OpenedAt, Cause: incident.Cause})
	}

	return &PublicMonitor{
		ID:            monitor.ID,
		Slug:          monitor.Slug,
		Name:          monitor.Name,
		CurrentStatus: status,
		Uptime24h:     window["24h"],
		Uptime7d:      window["7d"],
		OpenIncidents: open,
	}, nil
}
