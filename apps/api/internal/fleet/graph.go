package fleet

import (
	"slices"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Edge kinds name why one repository depends on another.
const (
	EdgeLib      = "lib"
	EdgeCall     = "call"
	EdgeEvent    = "event"
	EdgeIdentity = "identity"
	EdgeLogs     = "logs"
)

const (
	maxEdgesPerProject  = 1000
	maxChecksPerProject = 100
	maxCheckIDLength    = 64
)

var (
	edgeKinds = []string{EdgeLib, EdgeCall, EdgeEvent, EdgeIdentity, EdgeLogs}
	callAuths = []string{"audience", "static_key", "webhook", "url"}
)

// EdgeReport is one dependency of the reporting project on another repository.
// Auth is set on call edges only and says how the caller authenticates:
// audience, static_key, webhook or url. Evidence is the pin spec or the
// environment variable the edge was read from.
type EdgeReport struct {
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Auth     string `json:"auth,omitempty"`
	Evidence string `json:"evidence"`
}

// CheckReport is one conformance check the runner asked of a project. A check
// that does not apply is left out rather than reported as failed.
type CheckReport struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func (p ProjectReport) validateGraph() error {
	if len(p.Edges) > maxEdgesPerProject || len(p.Conformance) > maxChecksPerProject {
		return errors.TooLarge("project " + p.Name + " reports too many edges or checks")
	}
	if !p.Covered && (len(p.Edges) > 0 || len(p.Conformance) > 0) {
		return errors.Invalid("project " + p.Name + " is not covered but carries edges or checks")
	}
	for _, edge := range p.Edges {
		if err := edge.validate(p.Name); err != nil {
			return err
		}
	}
	if slices.ContainsFunc(p.Conformance, CheckReport.invalid) {
		return errors.Invalid("project " + p.Name + " has a check without a valid id or detail")
	}
	return nil
}

func (c CheckReport) invalid() bool {
	id := strings.TrimSpace(c.ID)
	return id == "" || len(id) > maxCheckIDLength || len(c.Detail) > maxFieldLength || hasNUL(c.ID, c.Detail)
}

func (e EdgeReport) validate(project string) error {
	switch {
	case strings.TrimSpace(e.To) == "" || len(e.To) > maxFieldLength || len(e.Evidence) > maxFieldLength:
		return errors.Invalid("project " + project + " has an edge without a valid target or evidence")
	case !slices.Contains(edgeKinds, e.Kind):
		return errors.Invalid("project " + project + " has an edge of an unknown kind")
	case e.Kind == EdgeCall && !slices.Contains(callAuths, e.Auth), e.Kind != EdgeCall && e.Auth != "":
		return errors.Invalid("project " + project + " has an edge whose auth does not match its kind")
	case hasNUL(e.To, e.Kind, e.Auth, e.Evidence):
		return errors.Invalid("project " + project + " has an edge carrying a NUL byte")
	}
	return nil
}

func replaceGraph(tx *gorm.DB, projectID int64, project ProjectReport) error {
	if err := tx.Where("project_id = ?", projectID).Delete(&schemas.FleetEdge{}).Error; err != nil {
		return err
	}
	if err := tx.Where("project_id = ?", projectID).Delete(&schemas.FleetCheck{}).Error; err != nil {
		return err
	}
	edges := map[EdgeReport]schemas.FleetEdge{}
	for _, edge := range project.Edges {
		edges[edge] = schemas.FleetEdge{ProjectID: projectID, ToName: edge.To, Kind: edge.Kind, Auth: edge.Auth, Evidence: edge.Evidence}
	}
	checks := map[string]schemas.FleetCheck{}
	for _, check := range project.Conformance {
		checks[check.ID] = schemas.FleetCheck{ProjectID: projectID, CheckID: check.ID, OK: check.OK, Detail: check.Detail}
	}
	if err := insertAll(tx, edges); err != nil {
		return err
	}
	return insertAll(tx, checks)
}

func insertAll[K comparable, V any](tx *gorm.DB, rows map[K]V) error {
	if len(rows) == 0 {
		return nil
	}
	values := make([]V, 0, len(rows))
	for _, row := range rows {
		values = append(values, row)
	}
	return tx.CreateInBatches(values, insertBatch).Error
}
