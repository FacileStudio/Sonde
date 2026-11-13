package monitors

import (
	"context"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// SetProject links the owner's monitor to a project, or unlinks it when
// projectID is nil. The caller has already checked the person may read the
// project; this only enforces that the monitor is theirs.
func (s *Store) SetProject(ctx context.Context, ownerID, id int64, projectID *int64) (*schemas.Monitor, error) {
	result := s.db.WithContext(ctx).Model(&schemas.Monitor{}).
		Where("id = ? AND owner_id = ?", id, ownerID).
		Updates(map[string]any{"project_id": projectID, "updated_at": time.Now()})
	if result.Error != nil {
		return nil, errors.Internal("could not link the monitor", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, errors.NotFound("monitor not found")
	}
	return s.get(ctx, ownerID, id)
}
