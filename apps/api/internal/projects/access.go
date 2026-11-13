// Package projects answers which GitHub repositories a signed-in person may
// link a monitor to. A fleet admin may link any project; anyone else needs a
// GitHub account linked in Registre with at least read on the repository, as
// the GitHub App reports it.
package projects

import (
	"context"
	"fmt"
	"sync"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
	"github.com/FacileStudio/Sonde/apps/api/internal/githubapp"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Owner is the GitHub organisation every project belongs to: the fleet scan
// lists its repositories and stores their bare names.
const Owner = "FacileStudio"

// NoGitHubAccount is the reason a person without a linked GitHub account gets.
const NoGitHubAccount = "link your GitHub account at https://sso.facile.studio/account, then sign in to Sonde again"

const checkWorkers = 8

// Ref names one project.
type Ref struct {
	ID   int64  `json:"id" gorm:"column:id"`
	Name string `json:"name" gorm:"column:name"`
}

// Access decides who may see and link which project.
type Access struct {
	db     *gorm.DB
	app    *githubapp.Client
	admins []string
}

// NewAccess builds the access checks. A nil app leaves fleet admins working
// and answers failed_precondition to everyone else.
func NewAccess(db *gorm.DB, app *githubapp.Client, admins []string) *Access {
	return &Access{db: db, app: app, admins: admins}
}

// Readable lists the projects identity may link, by name.
func (a *Access) Readable(ctx context.Context, identity kit.Identity) ([]Ref, error) {
	all, err := a.all(ctx)
	if err != nil || fleet.IsFleetAdmin(identity, a.admins) {
		return all, err
	}
	githubID, err := a.githubID(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}
	return a.filter(ctx, all, githubID)
}

// Check returns the project id names when identity may link it: not_found
// for an unknown id, permission_denied when GitHub grants less than read.
func (a *Access) Check(ctx context.Context, identity kit.Identity, id int64) (Ref, error) {
	var ref Ref
	found := a.db.WithContext(ctx).Table("projects").Select("id, name").Where("id = ?", id).Limit(1).Scan(&ref)
	if found.Error != nil {
		return Ref{}, errors.Internal("failed to read the project", found.Error)
	}
	if found.RowsAffected == 0 {
		return Ref{}, errors.NotFound("project not found")
	}
	if fleet.IsFleetAdmin(identity, a.admins) {
		return ref, nil
	}
	githubID, err := a.githubID(ctx, identity.UserID)
	if err != nil {
		return Ref{}, err
	}
	level, err := a.app.Permission(ctx, Owner, ref.Name, githubID)
	if err != nil {
		return Ref{}, err
	}
	if !githubapp.AtLeast(level, githubapp.Read) {
		return Ref{}, errors.Forbidden(fmt.Sprintf("your GitHub account cannot read %s/%s", Owner, ref.Name))
	}
	return ref, nil
}

// githubID reads the GitHub account Registre reported for the user, refusing
// with a plain reason when the App is off or no account is linked.
func (a *Access) githubID(ctx context.Context, userID int64) (string, error) {
	if !a.app.Enabled() {
		return "", errors.Failed(githubapp.NotConfigured)
	}
	var githubID string
	if err := a.db.WithContext(ctx).Table("users").Select("github_id").Where("id = ?", userID).Scan(&githubID).Error; err != nil {
		return "", errors.Internal("failed to read the GitHub account", err)
	}
	if githubID == "" {
		return "", errors.Failed(NoGitHubAccount)
	}
	return githubID, nil
}

// filter keeps the projects githubID may read, asking GitHub with a few
// requests in flight. The first failure wins: a partial list would read as
// "you cannot see these" when GitHub simply did not answer.
func (a *Access) filter(ctx context.Context, all []Ref, githubID string) ([]Ref, error) {
	readable := make([]bool, len(all))
	errs := make([]error, len(all))
	slots := make(chan struct{}, checkWorkers)
	var wg sync.WaitGroup
	for i, ref := range all {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			level, err := a.app.Permission(ctx, Owner, ref.Name, githubID)
			readable[i], errs[i] = githubapp.AtLeast(level, githubapp.Read), err
		})
	}
	wg.Wait()
	out := make([]Ref, 0, len(all))
	for i, ref := range all {
		if errs[i] != nil {
			return nil, errs[i]
		}
		if readable[i] {
			out = append(out, ref)
		}
	}
	return out, nil
}
