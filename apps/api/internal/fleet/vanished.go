package fleet

import (
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"gorm.io/gorm"
)

func coveredCount(tx *gorm.DB) (int, error) {
	var count int64
	err := tx.Model(&schemas.FleetProject{}).Where("covered").Count(&count).Error
	return int(count), err
}

// shrank reports whether a listing is too short to be trusted as the whole
// fleet: a narrowed GitHub App installation or a partial listing would
// otherwise resolve every project it no longer sees.
func shrank(listed, covered int) bool {
	if covered == 0 {
		return false
	}
	return float64(listed) < shrinkRatio*float64(covered) || covered-listed > shrinkLimit
}

func resolveVanished(tx *gorm.DB, report Report, covered int, now time.Time, result *IngestResult) error {
	if !result.Complete {
		return nil
	}
	if !report.Run.AcceptShrink && shrank(len(report.Projects), covered) {
		result.Complete, result.CoverageDropped = false, true
		return nil
	}
	return resolveListed(tx, report, now, result)
}

func resolveListed(tx *gorm.DB, report Report, now time.Time, result *IngestResult) error {
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
