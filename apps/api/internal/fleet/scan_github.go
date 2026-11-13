package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/githubapp"
	"github.com/FacileStudio/tronc/errors"
)

const maxRunsReply = 1 << 20

type workflowRun struct {
	ID           int64     `json:"id"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	RunStartedAt time.Time `json:"run_started_at"`
}

// observation is what GitHub says of the newest scan run, nil when the App is
// not configured or the workflow never ran.
type observation struct {
	run   *workflowRun
	steps []ScanStep
}

// Observe reads the newest runs of the scan workflow, whatever started them,
// and the steps of the one worth showing, with a token limited to Actions read.
func (d Dispatcher) Observe(ctx context.Context) (observation, error) {
	if !d.App.Enabled() {
		return observation{}, nil
	}
	token, err := d.App.Token(ctx, githubapp.Scope{
		Owner: scanOwner, Repositories: []string{scanRepository}, Permissions: map[string]string{"actions": "read"},
	})
	if err != nil {
		return observation{}, err
	}
	var runs struct {
		WorkflowRuns []workflowRun `json:"workflow_runs"`
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/runs?per_page=5", scanOwner, scanRepository, scanWorkflow)
	if err := d.get(ctx, token, path, &runs); err != nil {
		return observation{}, err
	}
	run := pickRun(runs.WorkflowRuns)
	if run == nil {
		return observation{}, nil
	}
	steps, err := d.steps(ctx, token, run.ID)
	return observation{run: run, steps: steps}, err
}

// pickRun prefers a run that is still going over a newer one queued behind it,
// since the workflow runs one scan at a time; otherwise the newest.
func pickRun(runs []workflowRun) *workflowRun {
	for i := range runs {
		if runs[i].Status == "in_progress" {
			return &runs[i]
		}
	}
	if len(runs) == 0 {
		return nil
	}
	return &runs[0]
}

func (d Dispatcher) steps(ctx context.Context, token string, runID int64) ([]ScanStep, error) {
	var jobs struct {
		Jobs []struct {
			Steps []ScanStep `json:"steps"`
		} `json:"jobs"`
	}
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/jobs", scanOwner, scanRepository, runID)
	if err := d.get(ctx, token, path, &jobs); err != nil {
		return nil, err
	}
	steps := []ScanStep{}
	for _, job := range jobs.Jobs {
		steps = append(steps, job.Steps...)
	}
	return steps, nil
}

func (d Dispatcher) get(ctx context.Context, token, path string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, d.App.API()+path, nil)
	if err != nil {
		return errors.Internal("build run request", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", githubAcceptHeader)
	request.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	response, err := d.client().Do(request)
	if err != nil {
		return errors.Unavailable("GitHub did not answer the run lookup")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.Unavailable(fmt.Sprintf("GitHub refused the run lookup (status %d)", response.StatusCode))
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxRunsReply)).Decode(out); err != nil {
		return errors.Unavailable("GitHub sent a run list that is not JSON")
	}
	return nil
}
