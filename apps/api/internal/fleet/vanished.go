package fleet

import (
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"gorm.io/gorm"
)

func coveredCount(tx *gorm.DB) (int, error) {
	var count int64
	err := tx.Model(&schemas.Project{}).Where("covered").Count(&count).Error
	return int(count), err
}

// shrank reports whether a listing is too short to be trusted as the whole
// fleet: a narrowed GitHub App installation or a partial listing would
// otherwise delete every project it no longer sees.
func shrank(listed, covered int) bool {
	if covered == 0 {
		return false
	}
	return float64(listed) < shrinkRatio*float64(covered) || covered-listed > shrinkLimit
}

func removeVanished(tx *gorm.DB, report Report, covered int, result *IngestResult) error {
	if !result.Complete {
		return nil
	}
	if !report.Run.AcceptShrink && shrank(len(report.Projects), covered) {
		result.Complete, result.CoverageDropped = false, true
		return nil
	}
	return removeUnlisted(tx, report, result)
}

// removeUnlisted deletes the projects a complete run no longer lists. The
// foreign keys cascade to their findings, pins, checks, edges and holds, and
// unlink their monitors; edges other projects drew to them go by name. Their
// open findings count as resolved.
func removeUnlisted(tx *gorm.DB, report Report, result *IngestResult) error {
	listed := make([]int64, 0, len(report.Projects))
	for _, project := range report.Projects {
		listed = append(listed, project.GithubRepoID)
	}
	var gone []schemas.Project
	if err := tx.Select("id", "name").Where("github_repo_id NOT IN ?", listed).Order("name").Find(&gone).Error; err != nil {
		return err
	}
	if len(gone) == 0 {
		return nil
	}
	ids, names := make([]int64, 0, len(gone)), make([]string, 0, len(gone))
	for _, project := range gone {
		ids, names = append(ids, project.ID), append(names, strings.ToLower(project.Name))
		result.Removed = append(result.Removed, project.Name)
	}
	var open int64
	if err := tx.Model(&schemas.FleetFinding{}).Where("resolved_at IS NULL AND project_id IN ?", ids).Count(&open).Error; err != nil {
		return err
	}
	result.Resolved += int(open)
	if err := tx.Where("lower(to_name) IN ?", names).Delete(&schemas.FleetEdge{}).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", ids).Delete(&schemas.Project{}).Error
}
