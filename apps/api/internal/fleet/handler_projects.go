package fleet

import (
	"context"
	"net/http"
	"time"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// ProjectScan is whether the last scan covered a project, and when.
type ProjectScan struct {
	Covered       bool       `json:"covered"`
	NotCovered    string     `json:"not_covered"`
	LastScannedAt *time.Time `json:"last_scanned_at"`
}

// LinkedMonitor is a monitor linked to a project, whoever owns it.
type LinkedMonitor struct {
	ID        int64  `json:"id" gorm:"column:id"`
	Name      string `json:"name" gorm:"column:name"`
	ProjectID int64  `json:"-" gorm:"column:project_id"`
}

// ProjectResponse is one project as the page shows it, with the monitors
// linked to it.
type ProjectResponse struct {
	ID            int64  `json:"id"`
	GithubRepoID  int64  `json:"github_repo_id"`
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	Tier          string `json:"tier"`
	VersionURL    string `json:"version_url"`
	JournalApp    string `json:"journal_app"`
	ProjectScan
	Health       Health          `json:"health"`
	OpenFindings int             `json:"open_findings"`
	Monitors     []LinkedMonitor `json:"monitors"`
}

// linkedMonitors groups every monitor linked to a project by that project.
func (s *Store) linkedMonitors(ctx context.Context) (map[int64][]LinkedMonitor, error) {
	var rows []LinkedMonitor
	err := s.db.WithContext(ctx).Raw(`
		SELECT id, name, project_id FROM monitors
		WHERE project_id IS NOT NULL ORDER BY lower(name), id`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to read the linked monitors", err)
	}
	out := map[int64][]LinkedMonitor{}
	for _, row := range rows {
		out[row.ProjectID] = append(out[row.ProjectID], row)
	}
	return out, nil
}

func projectResponse(row ProjectRow, monitors []LinkedMonitor) ProjectResponse {
	if monitors == nil {
		monitors = []LinkedMonitor{}
	}
	return ProjectResponse{
		ID: row.ID, GithubRepoID: row.GithubRepoID, Name: row.Name, DefaultBranch: row.DefaultBranch,
		Tier: row.Tier, VersionURL: row.VersionURL, JournalApp: row.JournalApp,
		ProjectScan:  ProjectScan{Covered: row.Covered, NotCovered: row.NotCovered, LastScannedAt: row.LastScannedAt},
		OpenFindings: row.OpenFindings, Monitors: monitors,
		Health: Health{
			CIConclusion: row.CIConclusion, FiletWired: row.FiletWired, FiletPin: row.FiletPin,
			DeployedVersion: row.DeployedVersion, GoDirective: row.GoDirective,
		},
	}
}

// Projects lists every project with its open findings and linked monitors.
func (h *Handler) Projects(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.Projects(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	linked, err := h.store.linkedMonitors(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out := make([]ProjectResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, projectResponse(row, linked[row.ID]))
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}
