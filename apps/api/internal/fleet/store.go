package fleet

import (
	"context"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Store reads and writes the fleet tables.
type Store struct{ db *gorm.DB }

// NewStore opens the fleet tables on db.
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

type openRow struct {
	ProjectID   int64     `gorm:"column:project_id"`
	ProjectName string    `gorm:"column:project_name"`
	Tier        string    `gorm:"column:tier"`
	Advisory    string    `gorm:"column:advisory"`
	Severity    string    `gorm:"column:severity"`
	Package     string    `gorm:"column:package"`
	Ecosystem   string    `gorm:"column:ecosystem"`
	Installed   string    `gorm:"column:installed"`
	FixedIn     string    `gorm:"column:fixed_in"`
	KEV         bool      `gorm:"column:kev"`
	EPSS        float64   `gorm:"column:epss"`
	FirstSeen   time.Time `gorm:"column:first_seen"`
}

// Actions ranks every open finding.
func (s *Store) Actions(ctx context.Context) ([]Action, error) {
	var rows []openRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT f.project_id, p.name AS project_name, p.tier, f.advisory, f.severity, f.package,
		       f.ecosystem, f.installed, f.fixed_in, f.kev, f.epss, f.first_seen
		FROM fleet_findings f JOIN fleet_projects p ON p.id = f.project_id
		WHERE f.resolved_at IS NULL`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to read the fleet actions", err)
	}
	findings := make([]OpenFinding, 0, len(rows))
	for _, r := range rows {
		findings = append(findings, OpenFinding{
			ProjectID: r.ProjectID, ProjectName: r.ProjectName, Tier: Tier(r.Tier), Advisory: r.Advisory,
			Severity: r.Severity, Package: r.Package, Ecosystem: r.Ecosystem, Installed: r.Installed,
			FixedIn: r.FixedIn, KEV: r.KEV, EPSS: r.EPSS, FirstSeen: r.FirstSeen,
		})
	}
	return BuildActions(findings), nil
}

type pinRow struct {
	ProjectID   int64  `gorm:"column:project_id"`
	ProjectName string `gorm:"column:project_name"`
	Package     string `gorm:"column:package"`
	Resolved    string `gorm:"column:resolved"`
}

// Drift compares every stored pin with the latest known version.
func (s *Store) Drift(ctx context.Context, now time.Time) ([]DriftRow, error) {
	db := s.db.WithContext(ctx)
	var latestRows []schemas.FleetLatest
	if err := db.Find(&latestRows).Error; err != nil {
		return nil, errors.Internal("failed to read the latest versions", err)
	}
	latest := make(map[string]string, len(latestRows))
	for _, row := range latestRows {
		latest[row.Package] = row.Version
	}
	var rows []pinRow
	err := db.Raw(`
		SELECT x.project_id, p.name AS project_name, x.package, x.resolved
		FROM fleet_pins x JOIN fleet_projects p ON p.id = x.project_id
		WHERE p.tier NOT IN ('archive', 'experiment')`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to read the fleet pins", err)
	}
	pins := make([]Pin, 0, len(rows))
	for _, r := range rows {
		pins = append(pins, Pin{ProjectID: r.ProjectID, ProjectName: r.ProjectName, Package: r.Package, Resolved: r.Resolved})
	}
	policies, err := s.Policies(ctx)
	if err != nil {
		return nil, err
	}
	return BuildDrift(pins, latest, policies, now), nil
}

// ProjectRow is a project with its open finding count.
type ProjectRow struct {
	schemas.FleetProject
	OpenFindings int `gorm:"column:open_findings"`
}

// Projects lists every project, the ones carrying the most open findings first.
// Those the runner could not scan come last; an incomplete run is announced on
// its own, so a failed fetch is not lost at the bottom.
func (s *Store) Projects(ctx context.Context) ([]ProjectRow, error) {
	var rows []ProjectRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT p.*, (SELECT count(*) FROM fleet_findings f WHERE f.project_id = p.id AND f.resolved_at IS NULL) AS open_findings
		FROM fleet_projects p ORDER BY p.covered DESC, open_findings DESC, lower(p.name)`).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list the fleet projects", err)
	}
	return rows, nil
}

// ProjectUpdate is the part of a project that people own.
type ProjectUpdate struct {
	Tier       *string `json:"tier"`
	VersionURL *string `json:"version_url"`
	JournalApp *string `json:"journal_app"`
}

// UpdateProject changes the fields a person owns and leaves scan data alone.
func (s *Store) UpdateProject(ctx context.Context, id int64, in ProjectUpdate) (*schemas.FleetProject, error) {
	changes := map[string]any{}
	if in.Tier != nil {
		if !ValidTier(*in.Tier) {
			return nil, errors.Invalid("tier is not one of the known tiers")
		}
		changes["tier"] = *in.Tier
	}
	if in.VersionURL != nil {
		if *in.VersionURL != "" && !strings.HasPrefix(*in.VersionURL, "https://") {
			return nil, errors.Invalid("version_url must be empty or start with https://")
		}
		changes["version_url"] = *in.VersionURL
	}
	if in.JournalApp != nil {
		changes["journal_app"] = strings.TrimSpace(*in.JournalApp)
	}
	var project schemas.FleetProject
	db := s.db.WithContext(ctx)
	if err := db.First(&project, id).Error; err != nil {
		return nil, errors.NotFound("project not found")
	}
	if len(changes) > 0 {
		if err := db.Model(&project).Updates(changes).Error; err != nil {
			return nil, errors.Internal("failed to update the project", err)
		}
	}
	return &project, nil
}

// Runs lists the newest deliveries.
func (s *Store) Runs(ctx context.Context, limit int) ([]schemas.FleetRun, error) {
	var runs []schemas.FleetRun
	if err := s.db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&runs).Error; err != nil {
		return nil, errors.Internal("failed to list the fleet runs", err)
	}
	return runs, nil
}
