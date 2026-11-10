package fleet

import (
	"context"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"gorm.io/gorm"
)

// Policies lists every hold, expired ones included, so the page can show a hold
// that has lapsed and the project it released.
func (s *Store) Policies(ctx context.Context) ([]Policy, error) {
	var rows []schemas.FleetPolicy
	if err := s.db.WithContext(ctx).Order("expires_at").Find(&rows).Error; err != nil {
		return nil, errors.Internal("failed to list the fleet policies", err)
	}
	out := make([]Policy, 0, len(rows))
	for _, r := range rows {
		created := r.CreatedAt
		out = append(out, Policy{
			ID: r.ID, Package: r.Package, ProjectID: r.ProjectID, HeldVersion: r.HeldVersion,
			Reason: r.Reason, ExpiresAt: r.ExpiresAt, CreatedBy: r.CreatedBy, CreatedAt: &created,
		})
	}
	return out, nil
}

// PolicyInput is a request to hold a package on a version.
type PolicyInput struct {
	Package     string    `json:"package"`
	ProjectID   *int64    `json:"project_id"`
	HeldVersion string    `json:"held_version"`
	Reason      string    `json:"reason"`
	ExpiresAt   time.Time `json:"expires_at"`
}

const maxHold = 366 * 24 * time.Hour

// CreatePolicy records a hold. It must carry a reason and an end date within a
// year, because a hold with neither is a decision nobody can revisit.
func (s *Store) CreatePolicy(ctx context.Context, in PolicyInput, by string, now time.Time) (*Policy, error) {
	in, err := in.checked(now)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	if err := projectExists(db, in.ProjectID); err != nil {
		return nil, err
	}
	row := schemas.FleetPolicy{
		Package: in.Package, ProjectID: in.ProjectID, HeldVersion: in.HeldVersion,
		Reason: in.Reason, ExpiresAt: in.ExpiresAt, CreatedBy: by,
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, errors.Internal("failed to create the policy", err)
	}
	created := row.CreatedAt
	return &Policy{
		ID: row.ID, Package: row.Package, ProjectID: row.ProjectID, HeldVersion: row.HeldVersion,
		Reason: row.Reason, ExpiresAt: row.ExpiresAt, CreatedBy: row.CreatedBy, CreatedAt: &created,
	}, nil
}

// DeletePolicy removes a hold.
func (s *Store) DeletePolicy(ctx context.Context, id int64) error {
	done := s.db.WithContext(ctx).Delete(&schemas.FleetPolicy{}, id)
	if done.Error != nil {
		return errors.Internal("failed to delete the policy", done.Error)
	}
	if done.RowsAffected == 0 {
		return errors.NotFound("policy not found")
	}
	return nil
}

// checked trims the hold and refuses one with no reason or an end date outside
// the next year.
func (in PolicyInput) checked(now time.Time) (PolicyInput, error) {
	in.Package, in.HeldVersion, in.Reason = strings.TrimSpace(in.Package), strings.TrimSpace(in.HeldVersion), strings.TrimSpace(in.Reason)
	switch {
	case in.Package == "" || in.HeldVersion == "" || in.Reason == "":
		return in, errors.Invalid("package, held_version and reason are required")
	case !in.ExpiresAt.After(now):
		return in, errors.Invalid("expires_at must be in the future")
	case in.ExpiresAt.After(now.Add(maxHold)):
		return in, errors.Invalid("expires_at must be within a year")
	}
	return in, nil
}

func projectExists(db *gorm.DB, id *int64) error {
	if id == nil {
		return nil
	}
	var count int64
	if err := db.Model(&schemas.FleetProject{}).Where("id = ?", *id).Count(&count).Error; err != nil {
		return errors.Internal("failed to check the policy's project", err)
	}
	if count == 0 {
		return errors.Invalid("project_id does not name a project")
	}
	return nil
}
