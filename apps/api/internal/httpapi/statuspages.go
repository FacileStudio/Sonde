package httpapi

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type StatusPageInput struct {
	Slug       string  `json:"slug"`
	Title      string  `json:"title"`
	MonitorIDs []int64 `json:"monitor_ids"`
}

type StatusPageMonitorLine struct {
	ID       int64  `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type StatusPageResponse struct {
	ID        int64                   `json:"id"`
	Slug      string                  `json:"slug"`
	Title     string                  `json:"title"`
	Monitors  []StatusPageMonitorLine `json:"monitors"`
	CreatedAt time.Time               `json:"created_at"`
}

func (s *Service) validateStatusPageInput(in *StatusPageInput) error {
	if len(in.Slug) == 0 || len(in.Slug) > 100 {
		return errors.Invalid("slug must be between 1 and 100 characters")
	}
	if in.Title == "" || len(in.Title) > 200 {
		return errors.Invalid("title must be between 1 and 200 characters")
	}
	return nil
}

func (s *Service) setMembership(r *http.Request, pageID int64, monitorIDs []int64) error {
	if err := s.db.WithContext(r.Context()).
		Where("page_id = ?", pageID).
		Delete(&schemas.StatusPageMonitor{}).Error; err != nil {
		return errors.Internal("could not update page membership", err)
	}
	for position, monitorID := range monitorIDs {
		var count int64
		if err := s.db.WithContext(r.Context()).Model(&schemas.Monitor{}).
			Where("id = ?", monitorID).Count(&count).Error; err != nil || count == 0 {
			return errors.Invalid("unknown monitor in monitor_ids")
		}
		member := schemas.StatusPageMonitor{
			PageID:    pageID,
			MonitorID: monitorID,
			Position:  position,
		}
		if err := s.db.WithContext(r.Context()).Create(&member).Error; err != nil {
			return errors.Invalid("a monitor may appear on a page only once")
		}
	}
	return nil
}

func (s *Service) statusPageResponse(r *http.Request, page *schemas.StatusPage) (*StatusPageResponse, error) {
	var members []schemas.StatusPageMonitor
	if err := s.db.WithContext(r.Context()).
		Where("page_id = ?", page.ID).
		Order("position").
		Find(&members).Error; err != nil {
		return nil, errors.Internal("could not load page membership", err)
	}
	lines := make([]StatusPageMonitorLine, 0, len(members))
	for _, member := range members {
		var monitor schemas.Monitor
		if err := s.db.WithContext(r.Context()).First(&monitor, member.MonitorID).Error; err != nil {
			return nil, errors.Internal("could not load page monitor", err)
		}
		lines = append(lines, StatusPageMonitorLine{
			ID:       monitor.ID,
			Slug:     monitor.Slug,
			Name:     monitor.Name,
			Position: member.Position,
		})
	}
	return &StatusPageResponse{
		ID:        page.ID,
		Slug:      page.Slug,
		Title:     page.Title,
		Monitors:  lines,
		CreatedAt: page.CreatedAt,
	}, nil
}

func (s *Service) createStatusPage(w http.ResponseWriter, r *http.Request) {
	var input StatusPageInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := s.validateStatusPageInput(&input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	page := schemas.StatusPage{Slug: input.Slug, Title: input.Title, CreatedAt: time.Now()}
	if err := s.db.WithContext(r.Context()).Create(&page).Error; err != nil {
		httpjson.WriteError(w, errors.Conflict("slug already exists"))
		return
	}
	if err := s.setMembership(r, page.ID, input.MonitorIDs); err != nil {
		s.db.Delete(&page)
		httpjson.WriteError(w, err)
		return
	}
	out, err := s.statusPageResponse(r, &page)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, out)
}

func (s *Service) listStatusPages(w http.ResponseWriter, r *http.Request) {
	var pages []schemas.StatusPage
	if err := s.db.WithContext(r.Context()).Order("id").Find(&pages).Error; err != nil {
		httpjson.WriteError(w, errors.Internal("could not list status pages", err))
		return
	}
	out := make([]StatusPageResponse, 0, len(pages))
	for i := range pages {
		page, err := s.statusPageResponse(r, &pages[i])
		if err != nil {
			httpjson.WriteError(w, err)
			return
		}
		out = append(out, *page)
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) loadStatusPage(r *http.Request) (*schemas.StatusPage, error) {
	id, err := monitorID(r)
	if err != nil {
		return nil, err
	}
	var page schemas.StatusPage
	if dbErr := s.db.WithContext(r.Context()).First(&page, id).Error; dbErr != nil {
		return nil, errors.NotFound("status page not found")
	}
	return &page, nil
}

func (s *Service) getStatusPage(w http.ResponseWriter, r *http.Request) {
	page, err := s.loadStatusPage(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out, err := s.statusPageResponse(r, page)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) updateStatusPage(w http.ResponseWriter, r *http.Request) {
	var input StatusPageInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	page, err := s.loadStatusPage(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := s.validateStatusPageInput(&input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	page.Slug = input.Slug
	page.Title = input.Title
	if err := s.db.WithContext(r.Context()).Save(page).Error; err != nil {
		httpjson.WriteError(w, errors.Conflict("slug already exists"))
		return
	}
	if err := s.setMembership(r, page.ID, input.MonitorIDs); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out, err := s.statusPageResponse(r, page)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}

func (s *Service) deleteStatusPage(w http.ResponseWriter, r *http.Request) {
	page, err := s.loadStatusPage(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if dbErr := s.db.WithContext(r.Context()).Delete(page).Error; dbErr != nil {
		httpjson.WriteError(w, errors.Internal("could not delete status page", dbErr))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
