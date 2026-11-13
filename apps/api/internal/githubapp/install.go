package githubapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/FacileStudio/tronc/errors"
)

const (
	apiVersion   = "2022-11-28"
	acceptHeader = "application/vnd.github+json"
	maxReply     = 1 << 20
)

type installationRow struct {
	ID      int64 `json:"id"`
	Account struct {
		Login string `json:"login"`
	} `json:"account"`
}

// installation finds the App's installation on owner, an organisation or a
// user, by listing the App's installations once and caching every one.
func (c *Client) installation(ctx context.Context, owner string) (int64, error) {
	want := strings.ToLower(owner)
	c.mu.Lock()
	id, ok := c.installs[want]
	c.mu.Unlock()
	if ok {
		return id, nil
	}
	bearer, err := appJWT(c.clientID, c.key, c.now())
	if err != nil {
		return 0, errors.Internal("failed to sign the GitHub App token", err)
	}
	var rows []installationRow
	status, err := c.call(ctx, "/app/installations?per_page=100", bearer, nil, &rows)
	if err != nil {
		return 0, err
	}
	if status != http.StatusOK {
		return 0, errors.Failed(fmt.Sprintf("GitHub refused the App credentials (status %d): check the client id and key", status))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, row := range rows {
		c.installs[strings.ToLower(row.Account.Login)] = row.ID
	}
	if id, ok := c.installs[want]; ok {
		return id, nil
	}
	return 0, errors.Failed(fmt.Sprintf("the GitHub App is not installed on %s", owner))
}

// mint asks GitHub for an installation token narrowed to scope. GitHub answers
// 422 when a requested permission is one the installation was never granted,
// which names the fix: approve the permission on the App.
func (c *Client) mint(ctx context.Context, installation int64, scope Scope) (token, error) {
	bearer, err := appJWT(c.clientID, c.key, c.now())
	if err != nil {
		return token{}, errors.Internal("failed to sign the GitHub App token", err)
	}
	body := map[string]any{"permissions": scope.Permissions}
	if len(scope.Repositories) > 0 {
		body["repositories"] = scope.Repositories
	}
	var reply struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	path := fmt.Sprintf("/app/installations/%d/access_tokens", installation)
	status, err := c.call(ctx, path, bearer, body, &reply)
	if err != nil {
		return token{}, err
	}
	switch {
	case status == http.StatusCreated && reply.Token != "":
		return token{value: reply.Token, expires: reply.ExpiresAt}, nil
	case status == http.StatusUnprocessableEntity || status == http.StatusForbidden:
		return token{}, errors.Failed(fmt.Sprintf(
			"the GitHub App lacks a permission it needs (%s): approve the new permission on the App", scope.key()))
	default:
		return token{}, errors.Failed(fmt.Sprintf("GitHub refused an installation token (status %d)", status))
	}
}

// call sends one request to the GitHub API, a POST of body when there is one
// and a GET otherwise, and decodes a 2xx answer into out. It returns the status
// for the caller to map; only a transport failure is an error here. The bearer
// never reaches an error message or a log.
func (c *Client) call(ctx context.Context, path, bearer string, body, out any) (int, error) {
	method, payload := http.MethodGet, io.Reader(nil)
	if body != nil {
		method = http.MethodPost
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, errors.Internal("failed to encode a GitHub request", err)
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return 0, errors.Internal("failed to build a GitHub request", err)
	}
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("Accept", acceptHeader)
	request.Header.Set("X-GitHub-Api-Version", apiVersion)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return 0, errors.Unavailable("GitHub did not answer")
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(response.Body, maxReply))
	if out != nil && response.StatusCode/100 == 2 && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return 0, errors.Unavailable("GitHub sent an answer that is not JSON")
		}
	}
	return response.StatusCode, nil
}
