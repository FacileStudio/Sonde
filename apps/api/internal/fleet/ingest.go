package fleet

import (
	"context"
	"database/sql"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	insertBatch = 500

	// ingestLock serialises deliveries. Two reports landing together would lock
	// project rows in different orders and could interleave their resolves.
	ingestLock int64 = 7301942118

	notInLatestScan = "not in the latest scan"
)

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

// Complete reports whether a run scanned everything it set out to. A run that
// expected nothing proves nothing, so it is never complete.
func (r RunInfo) Complete() bool {
	return r.ReposExpected > 0 && r.ReposScanned == r.ReposExpected
}

// Ingest stores one report in a single transaction. A project the runner could
// not cover keeps its findings: absence from a scan that did not happen is not
// evidence the flaw is gone. A complete run is evidence, so a project it no
// longer lists has its findings resolved. A report that is not newer than the
// last one stored is refused, so a retried or late delivery cannot roll the
// picture back.
func (s *Store) Ingest(ctx context.Context, report Report, now time.Time) (IngestResult, error) {
	result := IngestResult{Projects: len(report.Projects), Complete: report.Run.Complete()}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := claim(tx, report.Run); err != nil {
			return err
		}
		for _, project := range report.Projects {
			if err := ingestProject(tx, project, now, &result); err != nil {
				return err
			}
		}
		if err := resolveVanished(tx, report, now, &result); err != nil {
			return err
		}
		if err := upsertLatest(tx, report.Latest, now); err != nil {
			return err
		}
		return recordRun(tx, report.Run, &result)
	})
	var known *errors.Error
	if stderrors.As(err, &known) {
		return IngestResult{}, err
	}
	if err != nil {
		return IngestResult{}, errors.Internal("failed to store the fleet report", err)
	}
	return result, nil
}

func claim(tx *gorm.DB, run RunInfo) error {
	if err := tx.Exec(`SELECT pg_advisory_xact_lock(?)`, ingestLock).Error; err != nil {
		return err
	}
	var newest sql.NullTime
	if err := tx.Raw(`SELECT max(finished_at) FROM fleet_runs`).Row().Scan(&newest); err != nil {
		return err
	}
	if newest.Valid && !run.FinishedAt.Truncate(time.Microsecond).After(newest.Time) {
		return errors.Conflict("a fleet report at least as recent is already stored")
	}
	return nil
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

func resolveMissing(tx *gorm.DB, projectID int64, now time.Time) (int, error) {
	done := tx.Model(&schemas.FleetFinding{}).
		Where("project_id = ? AND resolved_at IS NULL AND last_seen < ?", projectID, now).
		Update("resolved_at", now)
	return int(done.RowsAffected), done.Error
}

func resolveVanished(tx *gorm.DB, report Report, now time.Time, result *IngestResult) error {
	if !report.Run.Complete() {
		return nil
	}
	listed := make([]int64, 0, len(report.Projects))
	for _, project := range report.Projects {
		listed = append(listed, project.GithubRepoID)
	}
	gone := tx.Model(&schemas.FleetProject{}).Select("id").Where("github_repo_id NOT IN ?", listed)
	done := tx.Model(&schemas.FleetFinding{}).
		Where("resolved_at IS NULL AND project_id IN (?)", gone).
		Update("resolved_at", now)
	if done.Error != nil {
		return done.Error
	}
	result.Resolved += int(done.RowsAffected)
	return tx.Model(&schemas.FleetProject{}).
		Where("github_repo_id NOT IN ?", listed).
		Updates(map[string]any{"covered": false, "not_covered": notInLatestScan}).Error
}
