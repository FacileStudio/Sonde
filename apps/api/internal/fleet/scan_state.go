package fleet

import "time"

const (
	dispatchGrace = 2 * time.Minute
	dispatchSkew  = 10 * time.Second
	recentRun     = 10 * time.Minute
	progressFresh = 30 * time.Second
)

// tracked is the runner's latest update and when it arrived.
type tracked struct {
	update ProgressUpdate
	at     time.Time
}

// scanView joins what GitHub said with what the runner and the Rescan button
// told this instance, at one instant.
type scanView struct {
	obs          observation
	progress     tracked
	dispatchedAt time.Time
	now          time.Time
}

func (v scanView) status() ScanStatus {
	run := v.obs.run
	switch {
	case v.awaitingDispatch():
		at := v.dispatchedAt
		return ScanStatus{State: ScanQueued, StartedAt: &at, Steps: []ScanStep{}}
	case run == nil:
		return v.fromProgress()
	}
	status := ScanStatus{
		State: runState(*run, v.phaseOf(run.ID)), Conclusion: run.Conclusion, RunURL: run.HTMLURL,
		StartedAt: startOf(*run), Steps: v.obs.steps, Progress: v.progressOf(run.ID),
	}
	if status.Steps == nil {
		status.Steps = []ScanStep{}
	}
	if run.Status == "completed" && v.now.Sub(run.UpdatedAt) > recentRun {
		status.State = ScanIdle
	}
	return status
}

// awaitingDispatch covers the seconds between a dispatch and GitHub listing
// its run, when the newest run GitHub knows is an older, finished one.
func (v scanView) awaitingDispatch() bool {
	if v.dispatchedAt.IsZero() || v.now.Sub(v.dispatchedAt) > dispatchGrace {
		return false
	}
	run := v.obs.run
	return run == nil || (run.Status == "completed" && run.CreatedAt.Before(v.dispatchedAt.Add(-dispatchSkew)))
}

// fromProgress answers from the runner alone, for an instance without the
// GitHub App or while GitHub does not answer.
func (v scanView) fromProgress() ScanStatus {
	if v.progress.update.RunID == 0 || v.now.Sub(v.progress.at) > progressFresh {
		return ScanStatus{State: ScanIdle, Steps: []ScanStep{}}
	}
	run := workflowRun{ID: v.progress.update.RunID, Status: "in_progress"}
	return ScanStatus{
		State: runState(run, v.progress.update.Phase), Steps: []ScanStep{}, Progress: v.progressOf(run.ID),
	}
}

func (v scanView) phaseOf(runID int64) string {
	if v.progress.update.RunID != runID {
		return ""
	}
	return v.progress.update.Phase
}

func (v scanView) progressOf(runID int64) *ScanProgress {
	update := v.progress.update
	if update.RunID == 0 || update.RunID != runID {
		return nil
	}
	return &ScanProgress{Phase: update.Phase, ReposDone: update.ReposDone, ReposTotal: update.ReposTotal, Current: update.Current}
}

// runState maps a GitHub run to a scan state. Only a successful run counts as
// completed: a cancelled, skipped or timed-out one delivered nothing.
func runState(run workflowRun, phase string) string {
	switch run.Status {
	case "completed":
		if run.Conclusion == "success" {
			return ScanCompleted
		}
		return ScanFailed
	case "in_progress":
		if phase == PhaseDelivering {
			return ScanDelivering
		}
		return ScanInProgress
	}
	return ScanQueued
}

func startOf(run workflowRun) *time.Time {
	at := run.RunStartedAt
	if at.IsZero() {
		at = run.CreatedAt
	}
	if at.IsZero() {
		return nil
	}
	return &at
}
