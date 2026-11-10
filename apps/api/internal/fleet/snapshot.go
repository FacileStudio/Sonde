package fleet

import (
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
