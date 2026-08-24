package httpapi

import (
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// The export is import-shaped: slugs instead of ids, no generated values, no
// credentials of any kind. A push token or a password hash in here would turn
// a config snapshot into a secret leak.
type ExportMonitor struct {
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

type ExportStatusPage struct {
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Monitors []string `json:"monitors"`
}

type ExportResponse struct {
	Monitors    []ExportMonitor    `json:"monitors"`
	StatusPages []ExportStatusPage `json:"status_pages"`
}

func (s *Service) exportConfig(w http.ResponseWriter, r *http.Request) {
	var found []schemas.Monitor
	if err := s.db.WithContext(r.Context()).Order("id").Find(&found).Error; err != nil {
		httpjson.WriteError(w, errors.Internal("could not load monitors", err))
		return
	}
	monitors := make([]ExportMonitor, 0, len(found))
	for _, monitor := range found {
		monitors = append(monitors, ExportMonitor{
			Slug:            monitor.Slug,
			Name:            monitor.Name,
			Type:            monitor.Type,
			Target:          monitor.Target,
			IntervalSeconds: monitor.IntervalSeconds,
			TimeoutSeconds:  monitor.TimeoutSeconds,
			ExpectedStatus:  monitor.ExpectedStatus,
			ExpectedKeyword: monitor.ExpectedKeyword,
			Enabled:         monitor.Enabled,
		})
	}

	var pages []schemas.StatusPage
	if err := s.db.WithContext(r.Context()).Order("id").Find(&pages).Error; err != nil {
		httpjson.WriteError(w, errors.Internal("could not load status pages", err))
		return
	}
	statusPages := make([]ExportStatusPage, 0, len(pages))
	for _, page := range pages {
		var members []schemas.StatusPageMonitor
		if err := s.db.WithContext(r.Context()).
			Where("page_id = ?", page.ID).
			Order("position").
			Find(&members).Error; err != nil {
			httpjson.WriteError(w, errors.Internal("could not load page membership", err))
			return
		}
		slugs := make([]string, 0, len(members))
		for _, member := range members {
			var monitor schemas.Monitor
			if err := s.db.First(&monitor, member.MonitorID).Error; err != nil {
				httpjson.WriteError(w, errors.Internal("could not load page monitor", err))
				return
			}
			slugs = append(slugs, monitor.Slug)
		}
		statusPages = append(statusPages, ExportStatusPage{
			Slug:     page.Slug,
			Title:    page.Title,
			Monitors: slugs,
		})
	}

	httpjson.WriteJSON(w, http.StatusOK, ExportResponse{Monitors: monitors, StatusPages: statusPages})
}
