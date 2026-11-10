package fleet

import (
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type storedName struct {
	Key       string `gorm:"column:key"`
	Advisory  string `gorm:"column:advisory"`
	Aliases   string `gorm:"column:aliases"`
	Ecosystem string `gorm:"column:ecosystem"`
	Package   string `gorm:"column:package"`
	Installed string `gorm:"column:installed"`
}

func nameKey(ecosystem, pkg, installed, name string) string {
	return ecosystem + "|" + pkg + "|" + installed + "|" + name
}

// knownKeys maps every name a project's stored findings go by to the key they
// were filed under. A flaw first reported under its GHSA keeps that key when
// its CVE is assigned later and becomes the smallest name, so the new alias does
// not read as one finding resolved and one new.
func knownKeys(tx *gorm.DB, projectID int64) (map[string]string, error) {
	var rows []storedName
	err := tx.Model(&schemas.FleetFinding{}).
		Select("key, advisory, aliases, ecosystem, package, installed").
		Where("project_id = ?", projectID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	known := make(map[string]string, len(rows)*2)
	for _, row := range rows {
		for _, name := range cleanNames(append(strings.Split(row.Aliases, ","), row.Advisory)) {
			known[nameKey(row.Ecosystem, row.Package, row.Installed, name)] = row.Key
		}
	}
	return known, nil
}

func keyFor(f FindingReport, known map[string]string) string {
	for _, name := range f.Names() {
		if key, ok := known[nameKey(f.Ecosystem, f.Package, f.Installed, name)]; ok {
			return key
		}
	}
	return f.Key()
}

func findingRow(projectID int64, key string, f FindingReport, now time.Time) schemas.FleetFinding {
	return schemas.FleetFinding{
		ProjectID: projectID, Key: key,
		FleetAdvisory: schemas.FleetAdvisory{
			Advisory: f.ID, Aliases: strings.Join(f.Aliases, ","), Summary: f.Summary,
			Severity: normalizeSeverity(f.Severity), KEV: f.KEV, EPSS: f.EPSS,
		},
		FleetPackage: schemas.FleetPackage{Package: f.Package, Ecosystem: f.Ecosystem, Installed: f.Installed, FixedIn: f.FixedIn},
		Scope:        f.Scope, Target: f.Target, FirstSeen: now, LastSeen: now,
	}
}

func upsertFindings(tx *gorm.DB, projectID int64, findings []FindingReport, now time.Time) (int, error) {
	known, err := knownKeys(tx, projectID)
	if err != nil {
		return 0, err
	}
	batch := dedupedRows(projectID, findings, known, now)
	if len(batch) == 0 {
		return 0, nil
	}
	assignments := append(
		clause.AssignmentColumns([]string{"advisory", "aliases", "summary", "severity", "fixed_in", "kev", "epss", "scope", "target", "last_seen"}),
		clause.Assignment{Column: clause.Column{Name: "resolved_at"}, Value: gorm.Expr("NULL")},
	)
	err = tx.Clauses(clause.OnConflict{
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

// dedupedRows builds one row per key. The same flaw reported from two lockfiles
// keeps its most severe report, since an upsert cannot touch a row twice.
func dedupedRows(projectID int64, findings []FindingReport, known map[string]string, now time.Time) []schemas.FleetFinding {
	rows := make(map[string]schemas.FleetFinding, len(findings))
	for _, f := range findings {
		key := keyFor(f, known)
		row := findingRow(projectID, key, f, now)
		if kept, seen := rows[key]; seen && severityRank(kept.Severity) >= severityRank(row.Severity) {
			continue
		}
		rows[key] = row
	}
	batch := make([]schemas.FleetFinding, 0, len(rows))
	for _, row := range rows {
		batch = append(batch, row)
	}
	return batch
}
