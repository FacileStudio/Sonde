package main

import (
	"context"
	"fmt"
	"net/url"
)

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
