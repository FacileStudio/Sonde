package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

type runInfo struct {
	started, finished time.Time
	douaneVersion     string
}

func buildReport(info runInfo, repos []fetchedRepo, findings map[string][]fleet.FindingReport, latest map[string]string) fleet.Report {
	report := fleet.Report{
		SchemaVersion: fleet.SchemaVersion,
		Run:           fleet.RunInfo{StartedAt: info.started, FinishedAt: info.finished, DouaneVersion: info.douaneVersion},
		Latest:        latest,
		Projects:      make([]fleet.ProjectReport, 0, len(repos)),
	}
	for _, repo := range repos {
		project := projectOf(repo, findings)
		if !repo.nothing {
			report.Run.ReposExpected++
		}
		if project.Covered {
			report.Run.ReposScanned++
		}
		report.Projects = append(report.Projects, project)
	}
	return report
}

func projectOf(repo fetchedRepo, findings map[string][]fleet.FindingReport) fleet.ProjectReport {
	project := fleet.ProjectReport{
		GithubRepoID: repo.repo.ID, Name: repo.repo.Name, DefaultBranch: repo.repo.DefaultBranch,
		NotCovered: repo.reason, CIConclusion: repo.ci, FiletWired: repo.filetWired, FiletPin: repo.filetPin,
		GoDirective: repo.goDirective,
	}
	if !repo.covered {
		return project
	}
	found, scanned := findings[repo.repo.Name]
	if !scanned {
		project.NotCovered = "douane returned nothing for this repository"
		return project
	}
	project.Covered, project.Findings, project.Pins = true, found, repo.pins
	return project
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

func deliver(ctx context.Context, sondeURL, token string, report fleet.Report) (fleet.IngestResult, error) {
	var result fleet.IngestResult
	body, err := encodeReport(report)
	if err != nil {
		return result, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(sondeURL, "/")+"/api/fleet/reports", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Do(request)
	if err != nil {
		return result, err
	}
	defer func() { _ = response.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("sonde answered %d: %s", response.StatusCode, strings.TrimSpace(tail(string(reply))))
	}
	return result, json.Unmarshal(reply, &result)
}

func (r *runner) finish(ctx context.Context, report fleet.Report) error {
	if r.cfg.out != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(r.cfg.out, data, 0o600)
	}
	result, err := deliver(ctx, r.cfg.sondeURL, r.cfg.fleetToken, report)
	if err != nil {
		return err
	}
	log.Printf("fleetscan: delivered run %d: %d new, %d resolved, %d open, complete=%t",
		result.RunID, result.NewFindings, result.Resolved, result.FindingsOpen, result.Complete)
	return nil
}
