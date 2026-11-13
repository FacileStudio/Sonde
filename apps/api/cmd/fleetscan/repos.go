package main

import (
	"context"
	"log"
	"slices"
	"strings"
	"sync"
)

// listRepos lists the organisation's repositories to scan. An archived
// repository is left out entirely, so a complete run removes it from Sonde, and
// so is one without a default branch, which has no history yet.
func (r *runner) listRepos(ctx context.Context) ([]repoInfo, error) {
	all, err := r.github.repos(ctx, r.cfg.org)
	if err != nil {
		return nil, err
	}
	live := slices.DeleteFunc(all, func(repo repoInfo) bool { return repo.Archived || repo.DefaultBranch == "" })
	slices.SortFunc(live, func(a, b repoInfo) int { return strings.Compare(a.Name, b.Name) })
	return live, nil
}

func (r *runner) collectAll(ctx context.Context, repos []repoInfo) []fetchedRepo {
	out := make([]fetchedRepo, len(repos))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range r.cfg.workers {
		wg.Go(func() {
			for i := range jobs {
				r.progress.started(repos[i].Name)
				out[i] = r.collect(ctx, repos[i])
				r.progress.finished()
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
