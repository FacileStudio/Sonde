package fleet

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/tronc/httpjson"
)

const (
	maxCompressedReport   = 8 << 20
	maxDecompressedReport = 96 << 20
	defaultRunLimit       = 30
)

// Handler serves the fleet routes.
type Handler struct {
	store      *Store
	now        func() time.Time
	logger     *slog.Logger
	dispatcher Dispatcher
	scans      *scanTracker
}

// NewHandler serves the fleet routes over store and reports deliveries to logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, now: time.Now, logger: logger, scans: &scanTracker{}}
}

// Ingest accepts a gzip-compressed report. Compression is required rather than
// optional: a full report is several megabytes of repeated keys, and the server
// reads and writes under tight timeouts.
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	result, err := h.ingest(w, r)
	if err != nil {
		h.logRefusal(err)
		httpjson.WriteError(w, err)
		return
	}
	h.logDelivery(result)
	httpjson.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) (IngestResult, error) {
	var report Report
	if err := httpjson.DecodeGzipJSONLimit(w, r, &report, maxCompressedReport, maxDecompressedReport); err != nil {
		return IngestResult{}, err
	}
	now := h.now().UTC().Truncate(time.Microsecond)
	if err := report.Validate(now); err != nil {
		return IngestResult{}, err
	}
	result, err := h.store.Ingest(r.Context(), report, now)
	if err == nil {
		h.scans.delivered(Delivery{
			RunID: result.RunID, ReposScanned: report.Run.ReposScanned, ReposExpected: report.Run.ReposExpected,
			NewFindings: result.NewFindings, Resolved: result.Resolved, Removed: len(result.Removed), At: now,
		})
	}
	return result, err
}

// ActionsResponse is the ranked inbox.
type ActionsResponse struct {
	Actions []Action `json:"actions"`
	Counts  Counts   `json:"counts"`
}

// Counts totals the inbox by level.
type Counts struct {
	Act      int `json:"act"`
	Schedule int `json:"schedule"`
	Track    int `json:"track"`
}

func (h *Handler) Actions(w http.ResponseWriter, r *http.Request) {
	actions, err := h.store.Actions(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	want := Level(r.URL.Query().Get("level"))
	var counts Counts
	filtered := make([]Action, 0, len(actions))
	for _, action := range actions {
		switch action.Level {
		case LevelAct:
			counts.Act++
		case LevelSchedule:
			counts.Schedule++
		default:
			counts.Track++
		}
		if want == "" || want == action.Level {
			filtered = append(filtered, action)
		}
	}
	httpjson.WriteJSON(w, http.StatusOK, ActionsResponse{Actions: filtered, Counts: counts})
}

// DriftResponse is the version drift of the tracked packages.
type DriftResponse struct {
	Packages []DriftRow `json:"packages"`
}

func (h *Handler) Drift(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.Drift(r.Context(), h.now())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, DriftResponse{Packages: rows})
}

// Health is what the last scan saw of a project's state.
type Health struct {
	CIConclusion    string `json:"ci_conclusion"`
	FiletWired      bool   `json:"filet_wired"`
	FiletPin        string `json:"filet_pin"`
	DeployedVersion string `json:"deployed_version"`
	GoDirective     string `json:"go_directive"`
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	var in ProjectUpdate
	if err := httpjson.DecodeJSON(w, r, &in); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	project, err := h.store.UpdateProject(r.Context(), id, in)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{
		"id": project.ID, "tier": project.Tier, "version_url": project.VersionURL, "journal_app": project.JournalApp,
	})
}

// RunResponse is one delivery from the runner.
type RunResponse struct {
	ID            int64     `json:"id"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    time.Time `json:"finished_at"`
	DouaneVersion string    `json:"douane_version"`
	ReposExpected int       `json:"repos_expected"`
	ReposScanned  int       `json:"repos_scanned"`
	Complete      bool      `json:"complete"`
	FindingsOpen  int       `json:"findings_open"`
	ReceivedAt    time.Time `json:"received_at"`
}

func (h *Handler) Runs(w http.ResponseWriter, r *http.Request) {
	runs, err := h.store.Runs(r.Context(), defaultRunLimit)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	out := make([]RunResponse, 0, len(runs))
	for _, run := range runs {
		out = append(out, RunResponse{
			ID: run.ID, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt, DouaneVersion: run.DouaneVersion,
			ReposExpected: run.ReposExpected, ReposScanned: run.ReposScanned, Complete: run.Complete,
			FindingsOpen: run.FindingsOpen, ReceivedAt: run.CreatedAt,
		})
	}
	httpjson.WriteJSON(w, http.StatusOK, out)
}
