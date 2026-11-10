package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

const (
	deliveryAttempts = 3
	invalidScan      = "invalid scan data"
)

type runInfo struct {
	started, finished time.Time
	douaneVersion     string
	acceptShrink      bool
}

var (
	errAlreadyStored = errors.New("already stored")
	errIncomplete    = errors.New("run incomplete")
)

func buildReport(info runInfo, repos []fetchedRepo, verdicts map[string]verdict, latest map[string]string) fleet.Report {
	report := fleet.Report{
		SchemaVersion: fleet.SchemaVersion,
		Run: fleet.RunInfo{
			StartedAt: info.started, FinishedAt: info.finished, DouaneVersion: info.douaneVersion,
			AcceptShrink: info.acceptShrink,
		},
		Latest:   latest,
		Projects: make([]fleet.ProjectReport, 0, len(repos)),
	}
	for _, repo := range repos {
		report.Projects = append(report.Projects, projectOf(repo, verdicts))
	}
	sanitize(&report, info.finished)
	return report
}

func projectOf(repo fetchedRepo, verdicts map[string]verdict) fleet.ProjectReport {
	project := fleet.ProjectReport{
		GithubRepoID: repo.repo.ID, Name: repo.repo.Name, DefaultBranch: repo.repo.DefaultBranch,
		NotCovered: repo.reason, CIConclusion: repo.ci, FiletWired: repo.filetWired, FiletPin: repo.filetPin,
		GoDirective: repo.goDirective,
	}
	switch {
	case repo.nothing:
		project.Covered = true
		return project
	case repo.archived, !repo.covered:
		return project
	}
	result, scanned := verdicts[repo.dir]
	switch {
	case !scanned:
		project.NotCovered = "douane returned nothing for this repository"
	case !result.covered:
		project.NotCovered = result.reason
	default:
		project.Covered, project.Findings, project.Pins = true, result.findings, repo.pins
	}
	return project
}

// sanitize demotes any project the server would refuse and counts the run. An
// archived repository is reported but counted in neither total.
func sanitize(report *fleet.Report, now time.Time) {
	report.Run.ReposExpected, report.Run.ReposScanned = 0, 0
	for i := range report.Projects {
		if !projectValid(report.Projects[i], now) {
			demote(&report.Projects[i])
		}
		if report.Projects[i].NotCovered != archivedReason {
			report.Run.ReposExpected++
		}
		if report.Projects[i].Covered {
			report.Run.ReposScanned++
		}
	}
}

func projectValid(project fleet.ProjectReport, now time.Time) bool {
	single := fleet.Report{
		SchemaVersion: fleet.SchemaVersion,
		Run:           fleet.RunInfo{StartedAt: now, FinishedAt: now, ReposExpected: 1},
		Projects:      []fleet.ProjectReport{project},
	}
	if project.Covered {
		single.Run.ReposScanned = 1
	}
	return single.Validate(now) == nil
}

func demote(project *fleet.ProjectReport) {
	project.Covered, project.NotCovered, project.Findings, project.Pins = false, invalidScan, nil, nil
}

func encodeReport(report fleet.Report) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if err := json.NewEncoder(writer).Encode(report); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// deliveryError carries the status only: the reply can name a project, and the
// workflow log is public.
type deliveryError struct{ status int }

func (e deliveryError) Error() string { return fmt.Sprintf("sonde answered %d", e.status) }

func deliver(ctx context.Context, cfg config, report fleet.Report) (fleet.IngestResult, error) {
	body, err := encodeReport(report)
	if err != nil {
		return fleet.IngestResult{}, err
	}
	var result fleet.IngestResult
	for try := range deliveryAttempts {
		result, err = post(ctx, cfg, body)
		var refused deliveryError
		if try > 0 && errors.As(err, &refused) && refused.status == http.StatusConflict {
			return result, errAlreadyStored
		}
		if err == nil || (errors.As(err, &refused) && refused.status < 500) || try == deliveryAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(time.Duration(try+1) * cfg.retryWait):
		}
	}
	return result, err
}

func post(ctx context.Context, cfg config, body []byte) (fleet.IngestResult, error) {
	var result fleet.IngestResult
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.sondeURL, "/")+"/api/fleet/reports", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	request.Header.Set("Authorization", "Bearer "+cfg.fleetToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Do(request)
	if err != nil {
		return result, fmt.Errorf("deliver: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusOK {
		return result, deliveryError{status: response.StatusCode}
	}
	return result, json.Unmarshal(reply, &result)
}

// finish writes the report with -out, or delivers it. A retry refused as a
// duplicate means an earlier attempt was stored and only its reply was lost.
// A run Sonde stored as incomplete fails the job, so the workflow sends no
// heartbeat and a coverage gap that persists opens an incident.
func (r *runner) finish(ctx context.Context, report fleet.Report) error {
	if r.cfg.out != "" {
		return writeReport(r.cfg.out, report)
	}
	result, err := deliver(ctx, r.cfg, report)
	switch {
	case errors.Is(err, errAlreadyStored):
		log.Printf("fleetscan: an earlier attempt was stored; its reply was lost")
		result.Complete = report.Run.Complete()
	case err != nil:
		return err
	default:
		log.Printf("fleetscan: delivered run %d: %d new, %d resolved, %d open, complete=%t",
			result.RunID, result.NewFindings, result.Resolved, result.FindingsOpen, result.Complete)
	}
	return completeness(report, result)
}

func writeReport(path string, report fleet.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func completeness(report fleet.Report, result fleet.IngestResult) error {
	if result.Complete {
		return nil
	}
	return fmt.Errorf("%w: scanned %d of %d", errIncomplete, report.Run.ReposScanned, report.Run.ReposExpected)
}
