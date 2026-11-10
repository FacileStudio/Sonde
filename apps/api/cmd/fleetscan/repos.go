package main

import (
	"context"
	"log"
	"slices"
	"strings"
	"sync"
)

// listRepos splits the organisation's repositories into the ones to scan and
// the archived ones. A repository without a default branch has no history yet
// and is left out of both.
func (r *runner) listRepos(ctx context.Context) ([]repoInfo, []repoInfo, error) {
	all, err := r.github.repos(ctx, r.cfg.org)
	if err != nil {
		return nil, nil, err
	}
	var live, archived []repoInfo
	for _, repo := range all {
		switch {
		case repo.DefaultBranch == "":
		case repo.Archived:
			archived = append(archived, repo)
		default:
			live = append(live, repo)
		}
	}
	byName := func(a, b repoInfo) int { return strings.Compare(a.Name, b.Name) }
	slices.SortFunc(live, byName)
	slices.SortFunc(archived, byName)
	return live, archived, nil
}

func archivedRepos(repos []repoInfo) []fetchedRepo {
	out := make([]fetchedRepo, 0, len(repos))
	for _, repo := range repos {
		out = append(out, archivedRepo(repo))
	}
	return out
}

func (r *runner) collectAll(ctx context.Context, repos []repoInfo) []fetchedRepo {
	out := make([]fetchedRepo, len(repos))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range r.cfg.workers {
		wg.Go(func() {
			for i := range jobs {
				out[i] = r.collect(ctx, repos[i])
			}
		})
	}
	for i := range repos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}

func (r *runner) sweep(ctx context.Context, fetched []fetchedRepo) (map[string]verdict, error) {
	if !slices.ContainsFunc(fetched, func(f fetchedRepo) bool { return f.dir != "" }) {
		return map[string]verdict{}, nil
	}
	sweep, err := runDouane(ctx, r.cfg.douaneBin, r.root)
	if err != nil {
		return nil, err
	}
	return sweep.verdicts(), nil
}

func (r *runner) latest(ctx context.Context) map[string]string {
	out := map[string]string{}
	for pkg, repo := range trackedByRepo {
		tags, err := r.github.tags(ctx, repo)
		if err != nil {
			log.Printf("fleetscan: no tags for %s: %v", repo, err)
			continue
		}
		if tag := highestTag(tags); tag != "" {
			out[pkg] = tag
		}
	}
	return out
}
