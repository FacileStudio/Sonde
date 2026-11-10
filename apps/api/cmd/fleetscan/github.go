package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	pageSize     = 100
	maxAttempts  = 3
	maxBodyBytes = 64 << 20
	maxRateWait  = 60 * time.Second
)

type githubClient struct {
	base    string
	token   string
	http    *http.Client
	backoff time.Duration
	maxBody int64
}

func newGithubClient(base, token string, backoff time.Duration) *githubClient {
	return &githubClient{
		base: strings.TrimRight(base, "/"), token: token, backoff: backoff, maxBody: maxBodyBytes,
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

type statusError struct {
	status int
	path   string
}

func (e statusError) Error() string { return fmt.Sprintf("github %s: status %d", e.path, e.status) }

type attempt struct {
	body  []byte
	retry bool
	wait  time.Duration
	err   error
}

func (c *githubClient) get(ctx context.Context, path, accept string) ([]byte, error) {
	var last attempt
	for try := range maxAttempts {
		last = c.once(ctx, path, accept)
		if last.err == nil || !last.retry || try == maxAttempts-1 {
			break
		}
		wait := last.wait
		if wait == 0 {
			wait = time.Duration(try+1) * c.backoff
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return last.body, last.err
}

func (c *githubClient) once(ctx context.Context, path, accept string) attempt {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return attempt{err: err}
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Accept", accept)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := c.http.Do(request)
	if err != nil {
		return attempt{retry: true, err: fmt.Errorf("github %s: %w", path, err)}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= 400 {
		retry, wait := throttle(response)
		return attempt{retry: retry || response.StatusCode >= 500, wait: wait, err: statusError{status: response.StatusCode, path: path}}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxBody+1))
	if err != nil {
		return attempt{retry: true, err: fmt.Errorf("github %s: %w", path, err)}
	}
	if int64(len(body)) > c.maxBody {
		return attempt{err: fmt.Errorf("github %s: response larger than %d bytes", path, c.maxBody)}
	}
	return attempt{body: body}
}

func throttle(response *http.Response) (bool, time.Duration) {
	retryAfter := response.Header.Get("Retry-After")
	exhausted := response.Header.Get("X-RateLimit-Remaining") == "0"
	limited := response.StatusCode == http.StatusTooManyRequests ||
		(response.StatusCode == http.StatusForbidden && (retryAfter != "" || exhausted))
	if !limited {
		return false, 0
	}
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return true, min(time.Duration(seconds)*time.Second, maxRateWait)
	}
	if reset, err := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil && exhausted {
		return true, min(max(time.Until(time.Unix(reset, 0)), 0), maxRateWait)
	}
	return true, 0
}

func (c *githubClient) json(ctx context.Context, path string, out any) error {
	body, err := c.get(ctx, path, "application/vnd.github+json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("github %s: %w", path, err)
	}
	return nil
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
	path := fmt.Sprintf("/repos/%s/git/trees/%s?recursive=1", repo.FullName, escapePath(repo.DefaultBranch))
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
	var names []string
	for page := 1; ; page++ {
		var batch []struct {
			Name string `json:"name"`
		}
		if err := c.json(ctx, fmt.Sprintf("/repos/%s/tags?per_page=%d&page=%d", repo, pageSize, page), &batch); err != nil {
			return nil, err
		}
		for _, tag := range batch {
			names = append(names, tag.Name)
		}
		if len(batch) < pageSize {
			return names, nil
		}
	}
}
