package fleet

import (
	"context"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const insertBatch = 500

// IngestResult is what one delivery changed.
type IngestResult struct {
	RunID        int64 `json:"run_id"`
	Projects     int   `json:"projects"`
	Covered      int   `json:"covered"`
	NewFindings  int   `json:"new_findings"`
	Resolved     int   `json:"resolved"`
	FindingsOpen int   `json:"findings_open"`
	Complete     bool  `json:"complete"`
}

// Ingest stores one report in a single transaction. A project the runner could
// not cover keeps its findings: absence from a scan that did not happen is not
// evidence the flaw is gone.
func (s *Store) Ingest(ctx context.Context, report Report, now time.Time) (IngestResult, error) {
	var result IngestResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result.Projects = len(report.Projects)
		for _, project := range report.Projects {
			if err := ingestProject(tx, project, now, &result); err != nil {
				return err
			}
		}
		if err := upsertLatest(tx, report.Latest, now); err != nil {
			return err
		}
		return recordRun(tx, report.Run, &result)
	})
	if err != nil {
		return IngestResult{}, errors.Internal("failed to store the fleet report", err)
	}
	return result, nil
}

func ingestProject(tx *gorm.DB, project ProjectReport, now time.Time, result *IngestResult) error {
	id, err := upsertProject(tx, project, now)
	if err != nil || !project.Covered {
		return err
	}
	fresh, err := upsertFindings(tx, id, project.Findings, now)
	if err != nil {
		return err
	}
	resolved, err := resolveMissing(tx, id, now)
	if err != nil {
		return err
	}
	if err := replacePins(tx, id, project.Pins); err != nil {
		return err
	}
	result.Covered++
	result.NewFindings += fresh
	result.Resolved += resolved
	return nil
}

func upsertProject(tx *gorm.DB, project ProjectReport, now time.Time) (int64, error) {
	var id int64
	err := tx.Raw(`
		INSERT INTO fleet_projects
			(github_repo_id, name, default_branch, covered, not_covered, last_scanned_at,
			 ci_conclusion, filet_wired, filet_pin, deployed_version, go_directive)
		VALUES (?, ?, ?, ?, ?, CASE WHEN ? THEN ?::timestamptz END, ?, ?, ?, ?, ?)
		ON CONFLICT (github_repo_id) DO UPDATE SET
			name = EXCLUDED.name,
			default_branch = EXCLUDED.default_branch,
			covered = EXCLUDED.covered,
			not_covered = EXCLUDED.not_covered,
			last_scanned_at = COALESCE(EXCLUDED.last_scanned_at, fleet_projects.last_scanned_at),
			ci_conclusion = EXCLUDED.ci_conclusion,
			filet_wired = EXCLUDED.filet_wired,
			filet_pin = EXCLUDED.filet_pin,
			deployed_version = EXCLUDED.deployed_version,
			go_directive = EXCLUDED.go_directive,
			updated_at = now()
		RETURNING id`,
		project.GithubRepoID, project.Name, project.DefaultBranch, project.Covered, project.NotCovered,
		project.Covered, now,
		project.CIConclusion, project.FiletWired, project.FiletPin, project.DeployedVersion, project.GoDirective,
	).Scan(&id).Error
	return id, err
}

func upsertFindings(tx *gorm.DB, projectID int64, findings []FindingReport, now time.Time) (int, error) {
	rows := make(map[string]schemas.FleetFinding, len(findings))
	for _, f := range findings {
		key := f.Key()
		row := schemas.FleetFinding{
			ProjectID: projectID, Key: key,
			FleetAdvisory: schemas.FleetAdvisory{
				Advisory: f.ID, Aliases: strings.Join(f.Aliases, ","), Summary: f.Summary,
				Severity: normalizeSeverity(f.Severity), KEV: f.KEV, EPSS: f.EPSS,
			},
			FleetPackage: schemas.FleetPackage{Package: f.Package, Ecosystem: f.Ecosystem, Installed: f.Installed, FixedIn: f.FixedIn},
			Scope:        f.Scope, Target: f.Target, FirstSeen: now, LastSeen: now,
		}
		if kept, seen := rows[key]; seen && severityRank(kept.Severity) >= severityRank(row.Severity) {
			continue
		}
		rows[key] = row
	}
	if len(rows) == 0 {
		return 0, nil
	}
	batch := make([]schemas.FleetFinding, 0, len(rows))
	for _, row := range rows {
		batch = append(batch, row)
	}
	assignments := append(
		clause.AssignmentColumns([]string{"advisory", "aliases", "summary", "severity", "fixed_in", "kev", "epss", "scope", "target", "last_seen"}),
		clause.Assignment{Column: clause.Column{Name: "resolved_at"}, Value: gorm.Expr("NULL")},
	)
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "key"}},
		DoUpdates: assignments,
	}).CreateInBatches(batch, insertBatch).Error
	if err != nil {
		return 0, err
	}
	var fresh int64
	err = tx.Model(&schemas.FleetFinding{}).
		Where("project_id = ? AND first_seen = ?", projectID, now).Count(&fresh).Error
	return int(fresh), err
}

func resolveMissing(tx *gorm.DB, projectID int64, now time.Time) (int, error) {
	done := tx.Model(&schemas.FleetFinding{}).
		Where("project_id = ? AND resolved_at IS NULL AND last_seen < ?", projectID, now).
		Update("resolved_at", now)
	return int(done.RowsAffected), done.Error
}

func replacePins(tx *gorm.DB, projectID int64, pins []PinReport) error {
	if err := tx.Where("project_id = ?", projectID).Delete(&schemas.FleetPin{}).Error; err != nil {
		return err
	}
	unique := map[string]schemas.FleetPin{}
	for _, pin := range pins {
		unique[pin.Path+"|"+pin.Package] = schemas.FleetPin{
			ProjectID: projectID, Path: pin.Path, Package: pin.Package, Spec: pin.Spec, Resolved: pin.Resolved,
		}
	}
	if len(unique) == 0 {
		return nil
	}
	rows := make([]schemas.FleetPin, 0, len(unique))
	for _, row := range unique {
		rows = append(rows, row)
	}
	return tx.CreateInBatches(rows, insertBatch).Error
}

func upsertLatest(tx *gorm.DB, latest map[string]string, now time.Time) error {
	if len(latest) == 0 {
		return nil
	}
	rows := make([]schemas.FleetLatest, 0, len(latest))
	for name, version := range latest {
		rows = append(rows, schemas.FleetLatest{Package: name, Version: version, UpdatedAt: now})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "package"}},
		DoUpdates: clause.AssignmentColumns([]string{"version", "updated_at"}),
	}).Create(&rows).Error
}

func recordRun(tx *gorm.DB, info RunInfo, result *IngestResult) error {
	var open int64
	if err := tx.Model(&schemas.FleetFinding{}).Where("resolved_at IS NULL").Count(&open).Error; err != nil {
		return err
	}
	result.FindingsOpen = int(open)
	result.Complete = info.ReposScanned == info.ReposExpected && result.Covered == result.Projects
	run := schemas.FleetRun{
		StartedAt: info.StartedAt, FinishedAt: info.FinishedAt, DouaneVersion: info.DouaneVersion,
		ReposExpected: info.ReposExpected, ReposScanned: info.ReposScanned,
		Complete: result.Complete, FindingsOpen: result.FindingsOpen,
	}
	if err := tx.Create(&run).Error; err != nil {
		return err
	}
	result.RunID = run.ID
	return nil
}
