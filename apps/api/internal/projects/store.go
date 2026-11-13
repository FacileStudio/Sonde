package projects

import (
	"context"

	"github.com/FacileStudio/tronc/errors"
)

func (a *Access) all(ctx context.Context) ([]Ref, error) {
	refs := []Ref{}
	if err := a.db.WithContext(ctx).Table("projects").Select("id, name").Order("lower(name)").Scan(&refs).Error; err != nil {
		return nil, errors.Internal("failed to list the projects", err)
	}
	return refs, nil
}

// Names maps each project id to its ref, for decorating monitors. An id with
// no project is absent.
func (a *Access) Names(ctx context.Context, ids []int64) (map[int64]Ref, error) {
	out := map[int64]Ref{}
	if len(ids) == 0 {
		return out, nil
	}
	var refs []Ref
	if err := a.db.WithContext(ctx).Table("projects").Select("id, name").Where("id IN ?", ids).Scan(&refs).Error; err != nil {
		return nil, errors.Internal("failed to read the linked projects", err)
	}
	for _, ref := range refs {
		out[ref.ID] = ref
	}
	return out, nil
}
