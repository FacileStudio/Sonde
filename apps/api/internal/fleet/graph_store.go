package fleet

import (
	"context"
	"database/sql"
	"path"
	"slices"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// GraphNode is one repository on the dependency graph. ID is 0 for a target
// that edges name but the runner has never listed. UptimeStatus comes from the
// enabled monitors linked to the project: up, down or unknown, and empty when
// none is linked. LatestTag is known for tracked libraries only.
type GraphNode struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Tier            string `json:"tier"`
	DeployedVersion string `json:"deployed_version"`
	LatestTag       string `json:"latest_tag"`
	UptimeStatus    string `json:"uptime_status"`
	ChecksOK        int    `json:"checks_ok"`
	ChecksTotal     int    `json:"checks_total"`
	OpenAct         int    `json:"open_act"`
}

// GraphEdge is one stored edge, named from its source project.
type GraphEdge struct {
	From     string `json:"from" gorm:"column:from_name"`
	To       string `json:"to" gorm:"column:to_name"`
	Kind     string `json:"kind" gorm:"column:kind"`
	Auth     string `json:"auth" gorm:"column:auth"`
	Evidence string `json:"evidence" gorm:"column:evidence"`
}

// GraphResponse is the whole fleet graph.
type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type nodeRow struct {
	ID              int64  `gorm:"column:id"`
	Name            string `gorm:"column:name"`
	Tier            string `gorm:"column:tier"`
	DeployedVersion string `gorm:"column:deployed_version"`
	ChecksOK        int    `gorm:"column:checks_ok"`
	ChecksTotal     int    `gorm:"column:checks_total"`
}

// Graph reads every project as a node and every stored edge between them.
func (s *Store) Graph(ctx context.Context) (GraphResponse, error) {
	db := s.db.WithContext(ctx)
	var rows []nodeRow
	err := db.Raw(`
		SELECT p.id, p.name, p.tier, p.deployed_version,
		       count(c.id) FILTER (WHERE c.ok) AS checks_ok, count(c.id) AS checks_total
		FROM projects p LEFT JOIN fleet_checks c ON c.project_id = p.id
		GROUP BY p.id ORDER BY lower(p.name)`).Scan(&rows).Error
	if err != nil {
		return GraphResponse{}, errors.Internal("failed to read the fleet graph", err)
	}
	out := GraphResponse{Edges: []GraphEdge{}}
	err = db.Raw(`
		SELECT p.name AS from_name, e.to_name, e.kind, e.auth, e.evidence
		FROM fleet_edges e JOIN projects p ON p.id = e.project_id
		ORDER BY lower(p.name), lower(e.to_name), e.kind, e.evidence`).Scan(&out.Edges).Error
	if err != nil {
		return GraphResponse{}, errors.Internal("failed to read the fleet edges", err)
	}
	signals, err := s.nodeSignals(ctx)
	if err != nil {
		return GraphResponse{}, err
	}
	out.Nodes = buildNodes(rows, out.Edges, signals)
	return out, nil
}

type nodeSignals struct {
	latest  map[string]string
	uptime  map[int64]string
	openAct map[int64]int
}

func (s *Store) nodeSignals(ctx context.Context) (nodeSignals, error) {
	db := s.db.WithContext(ctx)
	signals := nodeSignals{latest: map[string]string{}, uptime: map[int64]string{}, openAct: map[int64]int{}}
	var latest []schemas.FleetLatest
	if err := db.Find(&latest).Error; err != nil {
		return signals, errors.Internal("failed to read the latest versions", err)
	}
	for _, row := range latest {
		signals.latest[strings.ToLower(path.Base(row.Package))] = row.Version
	}
	actions, err := s.Actions(ctx)
	if err != nil {
		return signals, err
	}
	for _, action := range actions {
		for _, project := range action.Projects {
			if action.Level == LevelAct {
				signals.openAct[project.ID]++
			}
		}
	}
	signals.uptime, err = s.uptimeByProject(ctx)
	return signals, err
}

type monitorState struct {
	ProjectID int64        `gorm:"column:project_id"`
	OK        sql.NullBool `gorm:"column:ok"`
}

// uptimeByProject reads the newest check of every enabled monitor linked to a
// project, keyed by that project. One monitor down marks the project down; a
// monitor never checked is unknown, never a fabricated green.
func (s *Store) uptimeByProject(ctx context.Context) (map[int64]string, error) {
	var rows []monitorState
	err := s.db.WithContext(ctx).Raw(`
		SELECT m.project_id, r.ok
		FROM monitors m
		LEFT JOIN LATERAL (
			SELECT ok FROM checks_raw WHERE monitor_id = m.id ORDER BY created_at DESC LIMIT 1
		) r ON true
		WHERE m.enabled AND m.project_id IS NOT NULL`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to read the monitor states", err)
	}
	out := map[int64]string{}
	for _, row := range rows {
		status := "unknown"
		if row.OK.Valid && row.OK.Bool {
			status = "up"
		} else if row.OK.Valid {
			status = "down"
		}
		if out[row.ProjectID] != "down" && (out[row.ProjectID] == "" || status != "unknown") {
			out[row.ProjectID] = status
		}
	}
	return out, nil
}

func buildNodes(rows []nodeRow, edges []GraphEdge, signals nodeSignals) []GraphNode {
	nodes := make([]GraphNode, 0, len(rows))
	known := map[string]bool{}
	for _, row := range rows {
		known[row.Name] = true
		key := strings.ToLower(row.Name)
		nodes = append(nodes, GraphNode{
			ID: row.ID, Name: row.Name, Tier: row.Tier, DeployedVersion: row.DeployedVersion,
			LatestTag: signals.latest[key], UptimeStatus: signals.uptime[row.ID],
			ChecksOK: row.ChecksOK, ChecksTotal: row.ChecksTotal, OpenAct: signals.openAct[row.ID],
		})
	}
	for _, edge := range edges {
		if !known[edge.To] {
			known[edge.To] = true
			nodes = append(nodes, GraphNode{Name: edge.To, LatestTag: signals.latest[strings.ToLower(edge.To)]})
		}
	}
	slices.SortFunc(nodes, func(a, b GraphNode) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return nodes
}
