package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	pageSize   = 100
	maxRetries = 3
)

type githubClient struct {
	base    string
	token   string
	http    *http.Client
	backoff time.Duration
}

func newGithubClient(base, token string) *githubClient {
	return &githubClient{base: strings.TrimRight(base, "/"), token: token, http: &http.Client{Timeout: 60 * time.Second}, backoff: time.Second}
}

type statusError struct {
	status int
	path   string
}

func (e statusError) Error() string { return fmt.Sprintf("github %s: status %d", e.path, e.status) }

func (c *githubClient) get(ctx context.Context, path, accept string) ([]byte, error) {
	var last error
	for attempt := range maxRetries {
		body, retry, err := c.once(ctx, path, accept)
		if err == nil || !retry {
			return body, err
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * c.backoff):
		}
	}
	return nil, last
}

func (c *githubClient) once(ctx context.Context, path, accept string) ([]byte, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, false, err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", accept)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return nil, true, err
	}
	if response.StatusCode >= 400 {
		throttled := response.StatusCode == http.StatusTooManyRequests || response.Header.Get("X-RateLimit-Remaining") == "0"
		return nil, throttled || response.StatusCode >= 500, statusError{status: response.StatusCode, path: path}
	}
	return body, false, nil
}

func (c *githubClient) json(ctx context.Context, path string, out any) error {
	body, err := c.get(ctx, path, "application/vnd.github+json")
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *githubClient) raw(ctx context.Context, repo, ref, path string) ([]byte, error) {
	return c.get(ctx, fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, escapePath(path), url.QueryEscape(ref)), "application/vnd.github.raw")
}

func escapePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

type repoInfo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Archived      bool   `json:"archived"`
	DefaultBranch string `json:"default_branch"`
}

func (c *githubClient) repos(ctx context.Context, org string) ([]repoInfo, error) {
	var all []repoInfo
	for page := 1; ; page++ {
		var batch []repoInfo
		path := fmt.Sprintf("/orgs/%s/repos?type=all&per_page=%d&page=%d", url.PathEscape(org), pageSize, page)
		if err := c.json(ctx, path, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			return all, nil
		}
	}
}

type treeResponse struct {
	Truncated bool `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

func (c *githubClient) paths(ctx context.Context, repo repoInfo) ([]string, bool, error) {
	var tree treeResponse
	path := fmt.Sprintf("/repos/%s/git/trees/%s?recursive=1", repo.FullName, url.PathEscape(repo.DefaultBranch))
	if err := c.json(ctx, path, &tree); err != nil {
		return nil, false, err
	}
	out := make([]string, 0, len(tree.Tree))
	for _, entry := range tree.Tree {
		if entry.Type == "blob" {
			out = append(out, entry.Path)
		}
	}
	return out, tree.Truncated, nil
}

func (c *githubClient) conclusion(ctx context.Context, repo repoInfo) string {
	var runs struct {
		WorkflowRuns []struct {
			Conclusion string `json:"conclusion"`
		} `json:"workflow_runs"`
	}
	path := fmt.Sprintf("/repos/%s/actions/runs?branch=%s&status=completed&per_page=1", repo.FullName, url.QueryEscape(repo.DefaultBranch))
	if err := c.json(ctx, path, &runs); err != nil || len(runs.WorkflowRuns) == 0 {
		return ""
	}
	return runs.WorkflowRuns[0].Conclusion
}

func (c *githubClient) tags(ctx context.Context, repo string) ([]string, error) {
	var tags []struct {
		Name string `json:"name"`
	}
	if err := c.json(ctx, fmt.Sprintf("/repos/%s/tags?per_page=%d", repo, pageSize), &tags); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	return names, nil
}
