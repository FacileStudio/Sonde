package httpapi

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// MonitorProjectInput links a monitor to a project, or unlinks it with null.
type MonitorProjectInput struct {
	ProjectID *int64 `json:"project_id"`
}

// monitorResponses renders monitors with the project each one links to.
func (s *Service) monitorResponses(ctx context.Context, found []schemas.Monitor) ([]MonitorResponse, error) {
	ids := make([]int64, 0, len(found))
	for _, monitor := range found {
		if monitor.ProjectID != nil {
			ids = append(ids, *monitor.ProjectID)
		}
	}
	names, err := s.projects.Names(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]MonitorResponse, 0, len(found))
	for i := range found {
		response := newMonitorResponse(&found[i])
		if id := found[i].ProjectID; id != nil {
			if ref, ok := names[*id]; ok {
				response.Project = &ref
			}
		}
		out = append(out, response)
	}
	return out, nil
}

func (s *Service) writeMonitor(w http.ResponseWriter, r *http.Request, status int, monitor *schemas.Monitor) {
	out, err := s.monitorResponses(r.Context(), []schemas.Monitor{*monitor})
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, status, out[0])
}

// mayLink checks the monitor is the caller's and, when linking rather than
// unlinking, that the caller may read the project.
func (s *Service) mayLink(ctx context.Context, monitorID int64, projectID *int64) (kit.Identity, error) {
	identity, ok := kit.FromContext(ctx)
	if !ok {
		return kit.Identity{}, errors.Unauthorized("missing auth")
	}
	if _, err := s.monitors.Get(ctx, identity.UserID, monitorID); err != nil {
		return kit.Identity{}, err
	}
	if projectID != nil {
		if _, err := s.projects.Check(ctx, identity, *projectID); err != nil {
			return kit.Identity{}, err
		}
	}
	return identity, nil
}

// linkMonitorProject sets or clears the project of one of the caller's
// monitors. Linking needs at least GitHub read on the repository.
func (s *Service) linkMonitorProject(w http.ResponseWriter, r *http.Request) {
	var input MonitorProjectInput
	if err := httpjson.DecodeJSON(w, r, &input); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	identity, err := s.mayLink(r.Context(), id, input.ProjectID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	monitor, err := s.monitors.SetProject(r.Context(), identity.UserID, id, input.ProjectID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	s.writeMonitor(w, r, http.StatusOK, monitor)
}
