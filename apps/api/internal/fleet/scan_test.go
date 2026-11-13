package fleet

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

var scanNow = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

func githubRuns(t *testing.T, runs, jobs string) Dispatcher {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":3,"account":{"login":"FacileStudio"}}]`))
	})
	mux.HandleFunc("POST /app/installations/3/access_tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":"ghs_read","expires_at":"2999-01-01T00:00:00Z"}`))
	})
	mux.HandleFunc("GET /repos/FacileStudio/Sonde/actions/workflows/fleet-scan.yml/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ghs_read" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"workflow_runs":` + runs + `}`))
	})
	mux.HandleFunc("GET /repos/FacileStudio/Sonde/actions/runs/{id}/jobs", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(jobs))
	})
	return Dispatcher{App: appAgainst(t, mux)}
}

func run(id int, status, conclusion string, created time.Time) string {
	return fmt.Sprintf(`{"id":%d,"status":%q,"conclusion":%q,"html_url":"https://github.com/run/%d",`+
		`"created_at":%q,"updated_at":%q,"run_started_at":%q}`, id, status, conclusion, id,
		created.Format(time.RFC3339), created.Format(time.RFC3339), created.Format(time.RFC3339))
}

const scanJobs = `{"jobs":[{"steps":[{"name":"Build fleetscan","status":"completed","conclusion":"success","number":4},` +
	`{"name":"Scan and deliver","status":"in_progress","conclusion":null,"number":5}]}]}`

func observed(t *testing.T, runs string) observation {
	t.Helper()
	obs, err := githubRuns(t, runs, scanJobs).Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return obs
}

func TestARunningScanShowsItsStepsAndTheRunnersProgress(t *testing.T) {
	older := run(8, "completed", "success", scanNow.Add(-time.Hour))
	obs := observed(t, "["+run(9, "queued", "", scanNow)+","+run(7, "in_progress", "", scanNow.Add(-time.Minute))+","+older+"]")
	progress := tracked{update: ProgressUpdate{RunID: 7, Phase: PhaseScanning, ReposDone: 40, ReposTotal: 94, Current: "Sonde"}, at: scanNow}

	got := scanView{obs: obs, progress: progress, now: scanNow}.status()

	if got.State != ScanInProgress || got.RunURL != "https://github.com/run/7" || len(got.Steps) != 2 || got.Steps[1].Status != "in_progress" {
		t.Fatalf("status = %+v: the run in progress wins over the one queued behind it", got)
	}
	if got.Progress == nil || got.Progress.ReposDone != 40 || got.Progress.Current != "Sonde" {
		t.Fatalf("progress = %+v", got.Progress)
	}
	progress.update.Phase = PhaseDelivering
	if got := (scanView{obs: obs, progress: progress, now: scanNow}).status(); got.State != ScanDelivering {
		t.Fatalf("state = %s, want delivering", got.State)
	}
	progress.update.RunID = 99
	if got := (scanView{obs: obs, progress: progress, now: scanNow}).status(); got.Progress != nil {
		t.Fatalf("progress of another run shown: %+v", got.Progress)
	}
}

func TestRunStatesFromGitHub(t *testing.T) {
	cases := []struct {
		runs, want string
	}{
		{"[]", ScanIdle},
		{"[" + run(1, "queued", "", scanNow) + "]", ScanQueued},
		{"[" + run(1, "waiting", "", scanNow) + "]", ScanQueued},
		{"[" + run(1, "completed", "success", scanNow.Add(-time.Minute)) + "]", ScanCompleted},
		{"[" + run(1, "completed", "failure", scanNow.Add(-time.Minute)) + "]", ScanFailed},
		{"[" + run(1, "completed", "cancelled", scanNow.Add(-time.Minute)) + "]", ScanFailed},
		{"[" + run(1, "completed", "failure", scanNow.Add(-time.Hour)) + "]", ScanIdle},
	}
	for _, c := range cases {
		if got := (scanView{obs: observed(t, c.runs), now: scanNow}).status(); got.State != c.want {
			t.Errorf("%s: state = %s, want %s", c.runs, got.State, c.want)
		}
	}
}

func TestADispatchReadsAsQueuedUntilGitHubListsItsRun(t *testing.T) {
	obs := observed(t, "["+run(1, "completed", "success", scanNow.Add(-time.Hour))+"]")
	view := scanView{obs: obs, dispatchedAt: scanNow.Add(-5 * time.Second), now: scanNow}
	if got := view.status(); got.State != ScanQueued || got.StartedAt == nil {
		t.Fatalf("status = %+v: just dispatched is queued", got)
	}
	view.dispatchedAt = scanNow.Add(-5 * time.Minute)
	if got := view.status(); got.State != ScanIdle {
		t.Fatalf("state = %s: a dispatch GitHub never listed stops reading as queued", got.State)
	}
}

func TestWithoutTheAppTheRunnersProgressStillShows(t *testing.T) {
	obs, err := Dispatcher{}.Observe(context.Background())
	if err != nil || obs.run != nil {
		t.Fatalf("unconfigured App: %+v %v", obs, err)
	}
	progress := tracked{update: ProgressUpdate{RunID: 5, Phase: PhaseDouane, ReposDone: 3, ReposTotal: 3}, at: scanNow.Add(-time.Second)}
	if got := (scanView{progress: progress, now: scanNow}).status(); got.State != ScanInProgress || got.Progress == nil {
		t.Fatalf("status = %+v", got)
	}
	progress.at = scanNow.Add(-time.Minute)
	if got := (scanView{progress: progress, now: scanNow}).status(); got.State != ScanIdle {
		t.Fatalf("state = %s: stale progress is idle", got.State)
	}
}

func TestAGitHubRefusalIsAnError(t *testing.T) {
	d := githubRuns(t, "[]", scanJobs)
	d.App = appAgainst(t, http.NotFoundHandler())
	if _, err := d.Observe(context.Background()); err == nil {
		t.Fatal("a GitHub that answers 404 must surface as an error")
	}
}
