package fleet

import (
	"context"
	"slices"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

// ConformanceProject is one project's row of the conformance matrix. A check
// missing from Checks did not apply to the project.
type ConformanceProject struct {
	ID     int64         `json:"id"`
	Name   string        `json:"name"`
	Tier   string        `json:"tier"`
	Passed int           `json:"passed"`
	Total  int           `json:"total"`
	Checks []CheckReport `json:"checks"`
}

// ConformanceResponse lists every check id seen and the projects, worst first.
type ConformanceResponse struct {
	Checks   []string             `json:"checks"`
	Projects []ConformanceProject `json:"projects"`
}

type checkRow struct {
	ProjectColumns
	CheckID string `gorm:"column:check_id"`
	OK      bool   `gorm:"column:ok"`
	Detail  string `gorm:"column:detail"`
}

// Conformance reads the stored checks as a matrix of projects and check ids.
func (s *Store) Conformance(ctx context.Context) (ConformanceResponse, error) {
	var rows []checkRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT c.project_id, p.name AS project_name, p.tier, c.check_id, c.ok, c.detail
		FROM fleet_checks c JOIN fleet_projects p ON p.id = c.project_id
		ORDER BY lower(p.name), p.id, c.check_id`).Scan(&rows).Error
	if err != nil {
		return ConformanceResponse{}, errors.Internal("failed to read the fleet conformance", err)
	}
	return buildConformance(rows), nil
}

// buildConformance groups check rows by project and sorts the projects by the
// share of checks they pass, lowest first, then by name.
func buildConformance(rows []checkRow) ConformanceResponse {
	out := ConformanceResponse{Checks: []string{}, Projects: []ConformanceProject{}}
	index := map[int64]int{}
	for _, row := range rows {
		if !slices.Contains(out.Checks, row.CheckID) {
			out.Checks = append(out.Checks, row.CheckID)
		}
		at, seen := index[row.ProjectID]
		if !seen {
			at = len(out.Projects)
			index[row.ProjectID] = at
			out.Projects = append(out.Projects, ConformanceProject{ID: row.ProjectID, Name: row.ProjectName, Tier: row.Tier})
		}
		project := &out.Projects[at]
		project.Checks = append(project.Checks, CheckReport{ID: row.CheckID, OK: row.OK, Detail: row.Detail})
		project.Total++
		if row.OK {
			project.Passed++
		}
	}
	slices.Sort(out.Checks)
	slices.SortStableFunc(out.Projects, worstFirst)
	return out
}

func worstFirst(a, b ConformanceProject) int {
	if left, right := a.Passed*b.Total, b.Passed*a.Total; left != right {
		return left - right
	}
	if a.Total-a.Passed != b.Total-b.Passed {
		return (b.Total - b.Passed) - (a.Total - a.Passed)
	}
	return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
}
