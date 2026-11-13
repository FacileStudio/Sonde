package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const (
	progressEvery   = time.Second
	progressTimeout = 5 * time.Second
)

// progress posts the runner's position to Sonde so the Maintenance page can
// show it live. It is best effort: an update that fails is dropped without a
// word, since the workflow log is public and a scan must never fail over it.
// Only the latest state is sent, at most once per interval. A nil *progress
// is off, which is what a run outside GitHub Actions or with -out gets.
type progress struct {
	url, token string
	client     *http.Client
	every      time.Duration
	mu         sync.Mutex
	state      fleet.ProgressUpdate
	wake       chan struct{}
}

func startProgress(ctx context.Context, cfg config) *progress {
	if cfg.out != "" || cfg.runID <= 0 || cfg.sondeURL == "" {
		return nil
	}
	p := &progress{
		url:    strings.TrimRight(cfg.sondeURL, "/") + "/api/fleet/scans/progress",
		token:  cfg.fleetToken,
		client: &http.Client{Timeout: progressTimeout},
		every:  cmp.Or(cfg.progressEvery, progressEvery),
		state:  fleet.ProgressUpdate{RunID: cfg.runID, Phase: fleet.PhaseListing},
		wake:   make(chan struct{}, 1),
	}
	go p.loop(ctx)
	p.notify()
	return p
}

func (p *progress) update(change func(*fleet.ProgressUpdate)) {
	if p == nil {
		return
	}
	p.mu.Lock()
	change(&p.state)
	p.mu.Unlock()
	p.notify()
}

func (p *progress) notify() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *progress) phase(name string, total int) {
	p.update(func(s *fleet.ProgressUpdate) { s.Phase, s.ReposTotal, s.Current = name, total, "" })
}

func (p *progress) started(name string) {
	p.update(func(s *fleet.ProgressUpdate) { s.Current = name })
}

func (p *progress) finished() {
	p.update(func(s *fleet.ProgressUpdate) { s.ReposDone = min(s.ReposDone+1, s.ReposTotal) })
}

func (p *progress) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.wake:
		}
		p.send(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(p.every):
		}
	}
}

func (p *progress) send(ctx context.Context) {
	p.mu.Lock()
	body, err := json.Marshal(p.state)
	p.mu.Unlock()
	if err != nil {
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return
	}
	request.Header.Set("Authorization", "Bearer "+p.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return
	}
	_ = response.Body.Close()
}
