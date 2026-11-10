// Command fleetscan is the scheduled half of fleet maintenance. It lists the
// organisation's repositories through the GitHub API, fetches their lockfiles
// without cloning, runs douane over them, and delivers one report to Sonde.
//
// It prints counts and never findings, so a public workflow log does not turn
// into a second list of weaknesses.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const (
	defaultWorkers = 8
	runTimeout     = 40 * time.Minute
)

type config struct {
	org, githubAPI, githubToken string
	sondeURL, fleetToken        string
	douaneBin, out              string
	workers                     int
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadConfig(args []string) (config, error) {
	var cfg config
	flags := flag.NewFlagSet("fleetscan", flag.ContinueOnError)
	flags.StringVar(&cfg.org, "org", envOr("FLEET_ORG", "FacileStudio"), "GitHub organisation to scan")
	flags.StringVar(&cfg.githubAPI, "github-api", envOr("GITHUB_API_URL", "https://api.github.com"), "GitHub API base URL")
	flags.StringVar(&cfg.douaneBin, "douane", envOr("DOUANE_BIN", "douane"), "path to the douane binary")
	flags.StringVar(&cfg.out, "out", "", "write the report as JSON to this file instead of delivering it")
	flags.IntVar(&cfg.workers, "workers", defaultWorkers, "repositories fetched in parallel")
	if err := flags.Parse(args); err != nil {
		return cfg, err
	}
	cfg.githubToken = os.Getenv("GITHUB_TOKEN")
	cfg.sondeURL = os.Getenv("SONDE_URL")
	cfg.fleetToken = os.Getenv("SONDE_FLEET_TOKEN")
	return cfg, cfg.validate()
}

func (c config) validate() error {
	switch {
	case c.githubToken == "":
		return fmt.Errorf("GITHUB_TOKEN is required")
	case c.out == "" && (c.sondeURL == "" || c.fleetToken == ""):
		return fmt.Errorf("SONDE_URL and SONDE_FLEET_TOKEN are required unless -out is set")
	case c.workers < 1:
		return fmt.Errorf("-workers must be at least 1")
	}
	return nil
}

type runner struct {
	cfg    config
	github *githubClient
	root   string
}

func main() { os.Exit(run()) }

func run() int {
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		log.Printf("fleetscan: %v", err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	if err := scan(ctx, cfg); err != nil {
		log.Printf("fleetscan: %v", err)
		return 1
	}
	return 0
}

func scan(ctx context.Context, cfg config) error {
	started := time.Now().UTC()
	root, err := os.MkdirTemp("", "fleetscan-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(root) }()
	r := &runner{cfg: cfg, github: newGithubClient(cfg.githubAPI, cfg.githubToken), root: root}

	repos, err := r.liveRepos(ctx)
	if err != nil {
		return fmt.Errorf("list repositories: %w", err)
	}
	fetched := r.collectAll(ctx, repos)
	findings, err := r.sweep(ctx, fetched)
	if err != nil {
		return err
	}
	report := buildReport(runInfo{
		started: started, finished: time.Now().UTC(), douaneVersion: douaneVersion(ctx, cfg.douaneBin),
	}, fetched, findings, r.latest(ctx))
	log.Printf("fleetscan: %d repositories listed, %d expected, %d scanned", len(repos), report.Run.ReposExpected, report.Run.ReposScanned)
	return r.finish(ctx, report)
}

func (r *runner) liveRepos(ctx context.Context) ([]repoInfo, error) {
	all, err := r.github.repos(ctx, r.cfg.org)
	if err != nil {
		return nil, err
	}
	live := all[:0]
	for _, repo := range all {
		if !repo.Archived && repo.DefaultBranch != "" {
			live = append(live, repo)
		}
	}
	slices.SortFunc(live, func(a, b repoInfo) int { return compareNames(a.Name, b.Name) })
	return live, nil
}

func compareNames(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
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

func (r *runner) sweep(ctx context.Context, fetched []fetchedRepo) (map[string][]fleet.FindingReport, error) {
	if !slices.ContainsFunc(fetched, func(f fetchedRepo) bool { return f.covered }) {
		return map[string][]fleet.FindingReport{}, nil
	}
	sweep, err := runDouane(ctx, r.cfg.douaneBin, r.root)
	if err != nil {
		return nil, err
	}
	return sweep.byRepo(), nil
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
