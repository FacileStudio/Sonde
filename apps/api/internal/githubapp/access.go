package githubapp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

// The repository permission levels GitHub reports, weakest first.
const (
	None  = "none"
	Read  = "read"
	Write = "write"
	Admin = "admin"
)

const accessTTL = 10 * time.Minute

type cached struct {
	value   string
	expires time.Time
}

// rank orders the levels; anything GitHub adds later ranks as none.
func rank(level string) int {
	switch level {
	case Read:
		return 1
	case Write:
		return 2
	case Admin:
		return 3
	}
	return 0
}

// AtLeast reports whether have grants at least want.
func AtLeast(have, want string) bool { return rank(have) >= rank(want) && rank(want) > 0 }

func (c *Client) remember(key string, compute func() (string, error)) (string, error) {
	c.mu.Lock()
	hit, ok := c.access[key]
	c.mu.Unlock()
	if ok && c.now().Before(hit.expires) {
		return hit.value, nil
	}
	value, err := compute()
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.access[key] = cached{value: value, expires: c.now().Add(accessTTL)}
	c.mu.Unlock()
	return value, nil
}

// Permission answers what the GitHub account githubID may do on owner/repo:
// none, read, write or admin. It needs only Metadata read, and caches each
// answer ten minutes, so a revoked collaborator keeps access that long at most.
func (c *Client) Permission(ctx context.Context, owner, repo, githubID string) (string, error) {
	if !c.Enabled() {
		return "", errors.Failed(NotConfigured)
	}
	return c.remember("perm|"+githubID+"|"+owner+"/"+repo, func() (string, error) {
		login, err := c.login(ctx, owner, githubID)
		if err != nil {
			return "", err
		}
		bearer, err := c.Token(ctx, metadataScope(owner))
		if err != nil {
			return "", err
		}
		var reply struct {
			Permission string `json:"permission"`
		}
		path := fmt.Sprintf("/repos/%s/%s/collaborators/%s/permission",
			url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(login))
		status, err := c.call(ctx, path, bearer, nil, &reply)
		switch {
		case err != nil:
			return "", err
		case status == http.StatusOK:
			return reply.Permission, nil
		case status == http.StatusNotFound || status == http.StatusForbidden:
			return None, nil
		}
		return "", errors.Unavailable(fmt.Sprintf("GitHub refused the permission check (status %d)", status))
	})
}

func metadataScope(owner string) Scope {
	return Scope{Owner: owner, Permissions: map[string]string{"metadata": "read"}}
}

// login resolves a numeric GitHub account id, which never changes, to the
// login the permission endpoint takes, which can.
func (c *Client) login(ctx context.Context, owner, githubID string) (string, error) {
	return c.remember("login|"+githubID, func() (string, error) {
		bearer, err := c.Token(ctx, metadataScope(owner))
		if err != nil {
			return "", err
		}
		var reply struct {
			Login string `json:"login"`
		}
		status, err := c.call(ctx, "/user/"+url.PathEscape(githubID), bearer, nil, &reply)
		switch {
		case err != nil:
			return "", err
		case status == http.StatusOK && reply.Login != "":
			return reply.Login, nil
		case status == http.StatusNotFound:
			return "", errors.Failed("the GitHub account linked in Registre no longer exists: link it again")
		}
		return "", errors.Unavailable(fmt.Sprintf("GitHub refused the account lookup (status %d)", status))
	})
}
