package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/githubapp"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/middleware"
)

const (
	scanOwner          = "FacileStudio"
	scanRepository     = "Sonde"
	scanWorkflow       = "fleet-scan.yml"
	scanRef            = "master"
	rescanPerSecond    = 1.0 / 300
	maxDispatchReply   = 64 << 10
	dispatchTimeout    = 15 * time.Second
	githubAPIVersion   = "2022-11-28"
	githubAcceptHeader = "application/vnd.github+json"
	missingActions     = "the GitHub App lacks Actions write: approve the new permission on the App"
)

// ScanResult says the scan workflow was queued. RunURL is empty when GitHub
// answered without run details.
type ScanResult struct {
	Queued bool   `json:"queued"`
	RunURL string `json:"run_url,omitempty"`
}

// Dispatcher queues the fleet-scan workflow through GitHub's workflow
// dispatch API, with a one-hour installation token of the GitHub App limited to
// Actions write on the Sonde repository. A nil App turns rescans off.
type Dispatcher struct {
	App    *githubapp.Client
	Client *http.Client
}

// Dispatch queues one run of the scan workflow on the default branch.
func (d Dispatcher) Dispatch(ctx context.Context) (ScanResult, error) {
	token, err := d.App.Token(ctx, githubapp.Scope{
		Owner:        scanOwner,
		Repositories: []string{scanRepository},
		Permissions:  map[string]string{"actions": "write"},
	})
	if err != nil {
		return ScanResult{}, err
	}
	request, err := d.request(ctx, token)
	if err != nil {
		return ScanResult{}, errors.Internal("build dispatch request", err)
	}
	response, err := d.client().Do(request)
	if err != nil {
		return ScanResult{}, errors.Unavailable("GitHub did not answer the dispatch")
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, maxDispatchReply))
	return dispatchResult(response.StatusCode, body)
}

func (d Dispatcher) request(ctx context.Context, token string) (*http.Request, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/dispatches",
		d.App.API(), scanOwner, scanRepository, scanWorkflow)
	payload, err := json.Marshal(map[string]string{"ref": scanRef})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", githubAcceptHeader)
	request.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

func (d Dispatcher) client() *http.Client {
	if d.Client != nil {
		return d.Client
	}
	return &http.Client{Timeout: dispatchTimeout}
}

func dispatchResult(status int, body []byte) (ScanResult, error) {
	switch {
	case status == http.StatusNoContent:
		return ScanResult{Queued: true}, nil
	case status == http.StatusOK:
		var reply struct {
			HTMLURL string `json:"html_url"`
		}
		_ = json.Unmarshal(body, &reply)
		return ScanResult{Queued: true, RunURL: reply.HTMLURL}, nil
	case status == http.StatusForbidden:
		return ScanResult{}, errors.Failed(missingActions)
	case status == http.StatusUnauthorized || status == http.StatusNotFound:
		return ScanResult{}, errors.Failed(fmt.Sprintf("GitHub refused the dispatch token (status %d)", status))
	default:
		return ScanResult{}, errors.Unavailable(fmt.Sprintf("GitHub refused the dispatch (status %d)", status))
	}
}

// rescanLimit allows one dispatch every five minutes across the instance. The
// workflow already serialises runs; this only stops a double click from
// queueing a second scan.
func rescanLimit() func(http.Handler) http.Handler {
	return middleware.Ratelimit(middleware.RatelimitConfig{
		Limit:   rescanPerSecond,
		Burst:   1,
		KeyFunc: func(*http.Request) string { return "fleet-rescan" },
	})
}

// Rescan queues the scan workflow.
func (h *Handler) Rescan(w http.ResponseWriter, r *http.Request) {
	result, err := h.dispatcher.Dispatch(r.Context())
	if err != nil {
		h.logger.Warn("fleet rescan refused", "error", err)
		httpjson.WriteError(w, err)
		return
	}
	h.logger.Info("fleet rescan queued", "run_url", result.RunURL)
	httpjson.WriteJSON(w, http.StatusAccepted, result)
}
