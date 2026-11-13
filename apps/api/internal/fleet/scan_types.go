package fleet

import (
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/FacileStudio/tronc/errors"
)

// The phases a running scan reports, in order.
const (
	PhaseListing    = "listing"
	PhaseScanning   = "scanning"
	PhaseDouane     = "douane"
	PhaseDelivering = "delivering"
)

// The states GET /fleet/scans/current answers.
const (
	ScanIdle       = "idle"
	ScanQueued     = "queued"
	ScanInProgress = "in_progress"
	ScanDelivering = "delivering"
	ScanCompleted  = "completed"
	ScanFailed     = "failed"
)

const (
	maxProgressRepos   = 10_000
	maxProgressCurrent = 200
)

// ProgressUpdate is what the runner posts while it works. RunID is the
// workflow's GITHUB_RUN_ID, so Sonde can tell which run it belongs to.
type ProgressUpdate struct {
	RunID      int64  `json:"run_id"`
	Phase      string `json:"phase"`
	ReposDone  int    `json:"repos_done"`
	ReposTotal int    `json:"repos_total"`
	Current    string `json:"current"`
}

// Validate refuses an update the page could not show truthfully.
func (p ProgressUpdate) Validate() error {
	switch {
	case p.RunID <= 0:
		return errors.Invalid("run_id is required")
	case p.Phase != PhaseListing && p.Phase != PhaseScanning && p.Phase != PhaseDouane && p.Phase != PhaseDelivering:
		return errors.Invalid(fmt.Sprintf("unknown phase %q", p.Phase))
	case p.ReposDone < 0 || p.ReposTotal < 0 || p.ReposDone > p.ReposTotal || p.ReposTotal > maxProgressRepos:
		return errors.Invalid("repos_done must be between 0 and repos_total")
	case !printable(p.Current):
		return errors.Invalid("current must be a short printable name")
	}
	return nil
}

func printable(s string) bool {
	if len(s) > maxProgressCurrent || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// ScanStep is one step of the scan job on GitHub Actions.
type ScanStep struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// ScanProgress is the runner's latest update for the run on screen.
type ScanProgress struct {
	Phase      string `json:"phase"`
	ReposDone  int    `json:"repos_done"`
	ReposTotal int    `json:"repos_total"`
	Current    string `json:"current"`
}

// Delivery is what the last report stored, kept in memory until a restart.
type Delivery struct {
	RunID         int64     `json:"run_id"`
	ReposScanned  int       `json:"repos_scanned"`
	ReposExpected int       `json:"repos_expected"`
	NewFindings   int       `json:"new_findings"`
	Resolved      int       `json:"resolved"`
	Removed       int       `json:"removed"`
	At            time.Time `json:"at"`
}

// ScanStatus is the state of the newest scan. Progress is null until the runner
// of the run on screen reports; LastDelivery is null after a restart.
type ScanStatus struct {
	State        string        `json:"state"`
	Conclusion   string        `json:"conclusion"`
	RunURL       string        `json:"run_url"`
	StartedAt    *time.Time    `json:"started_at"`
	Steps        []ScanStep    `json:"steps"`
	Progress     *ScanProgress `json:"progress"`
	LastReportAt *time.Time    `json:"last_report_at"`
	LastDelivery *Delivery     `json:"last_delivery"`
}
