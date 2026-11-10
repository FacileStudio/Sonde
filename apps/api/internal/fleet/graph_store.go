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
// enabled monitors named like the project: up, down or unknown, and empty when
// no monitor carries that name. LatestTag is known for tracked libraries only.
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
		FROM fleet_projects p LEFT JOIN fleet_checks c ON c.project_id = p.id
		GROUP BY p.id ORDER BY lower(p.name)`).Scan(&rows).Error
	if err != nil {
		return GraphResponse{}, errors.Internal("failed to read the fleet graph", err)
	}
	out := GraphResponse{Edges: []GraphEdge{}}
	err = db.Raw(`
		SELECT p.name AS from_name, e.to_name, e.kind, e.auth, e.evidence
		FROM fleet_edges e JOIN fleet_projects p ON p.id = e.project_id
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
	uptime  map[string]string
	openAct map[int64]int
}

func (s *Store) nodeSignals(ctx context.Context) (nodeSignals, error) {
	db := s.db.WithContext(ctx)
	signals := nodeSignals{latest: map[string]string{}, uptime: map[string]string{}, openAct: map[int64]int{}}
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
	signals.uptime, err = s.uptimeByName(ctx)
	return signals, err
}

type monitorState struct {
	Name string       `gorm:"column:name"`
	OK   sql.NullBool `gorm:"column:ok"`
}

// uptimeByName reads the newest check of every enabled monitor, keyed by its
// lower-cased name. One monitor down marks the name down; a monitor never
// checked is unknown, never a fabricated green.
func (s *Store) uptimeByName(ctx context.Context) (map[string]string, error) {
	var rows []monitorState
	err := s.db.WithContext(ctx).Raw(`
		SELECT lower(m.name) AS name, r.ok
		FROM monitors m
		LEFT JOIN LATERAL (
			SELECT ok FROM checks_raw WHERE monitor_id = m.id ORDER BY created_at DESC LIMIT 1
		) r ON true
		WHERE m.enabled`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to read the monitor states", err)
	}
	out := map[string]string{}
	for _, row := range rows {
		status := "unknown"
		if row.OK.Valid && row.OK.Bool {
			status = "up"
		} else if row.OK.Valid {
			status = "down"
		}
		if out[row.Name] != "down" && (out[row.Name] == "" || status != "unknown") {
			out[row.Name] = status
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
			LatestTag: signals.latest[key], UptimeStatus: signals.uptime[key],
			ChecksOK: row.ChecksOK, ChecksTotal: row.ChecksTotal, OpenAct: signals.openAct[row.ID],
		})
	}
	for _, edge := range edges {
		if !known[edge.To] {
			known[edge.To] = true
			key := strings.ToLower(edge.To)
			nodes = append(nodes, GraphNode{Name: edge.To, LatestTag: signals.latest[key], UptimeStatus: signals.uptime[key]})
		}
	}
	slices.SortFunc(nodes, func(a, b GraphNode) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return nodes
}
