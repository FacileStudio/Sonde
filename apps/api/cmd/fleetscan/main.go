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
	"time"
)

const (
	defaultWorkers   = 8
	defaultRetryWait = time.Second
	runTimeout       = 40 * time.Minute
)

type config struct {
	org, githubAPI, githubToken string
	sondeURL, fleetToken        string
	douaneBin, out              string
	workers                     int
	retryWait                   time.Duration
	acceptShrink                bool
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadConfig(args []string) (config, error) {
	cfg := config{retryWait: defaultRetryWait}
	flags := flag.NewFlagSet("fleetscan", flag.ContinueOnError)
	flags.StringVar(&cfg.org, "org", envOr("FLEET_ORG", "FacileStudio"), "GitHub organisation to scan")
	flags.StringVar(&cfg.githubAPI, "github-api", envOr("GITHUB_API_URL", "https://api.github.com"), "GitHub API base URL")
	flags.StringVar(&cfg.douaneBin, "douane", envOr("DOUANE_BIN", "douane"), "path to the douane binary")
	flags.StringVar(&cfg.out, "out", "", "write the report as JSON to this file instead of delivering it")
	flags.IntVar(&cfg.workers, "workers", defaultWorkers, "repositories fetched in parallel")
	flags.BoolVar(&cfg.acceptShrink, "accept-shrink", os.Getenv("FLEET_ACCEPT_SHRINK") == "true",
		"confirm that repositories really left the organisation, so a much shorter listing may resolve them")
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
	r := &runner{cfg: cfg, github: newGithubClient(cfg.githubAPI, cfg.githubToken, cfg.retryWait), root: root}

	repos, archived, err := r.listRepos(ctx)
	if err != nil {
		return fmt.Errorf("list repositories: %w", err)
	}
	fetched := append(r.collectAll(ctx, repos), archivedRepos(archived)...)
	verdicts, err := r.sweep(ctx, fetched)
	if err != nil {
		return err
	}
	report := buildReport(runInfo{
		started: started, finished: time.Now().UTC(), douaneVersion: douaneVersion(ctx, cfg.douaneBin),
		acceptShrink: cfg.acceptShrink,
	}, fetched, verdicts, r.latest(ctx))
	log.Printf("fleetscan: %d repositories listed, %d expected, %d scanned", len(repos), report.Run.ReposExpected, report.Run.ReposScanned)
	return r.finish(ctx, report)
}
