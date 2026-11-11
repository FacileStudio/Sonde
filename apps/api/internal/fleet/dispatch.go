package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/middleware"
)

const (
	defaultGithubAPI   = "https://api.github.com"
	scanRepository     = "FacileStudio/Sonde"
	scanWorkflow       = "fleet-scan.yml"
	scanRef            = "master"
	rescanPerSecond    = 1.0 / 300
	maxDispatchReply   = 64 << 10
	dispatchTimeout    = 15 * time.Second
	githubAPIVersion   = "2022-11-28"
	githubAcceptHeader = "application/vnd.github+json"
)

// ScanResult says the scan workflow was queued. RunURL is empty when GitHub
// answered without run details.
type ScanResult struct {
	Queued bool   `json:"queued"`
	RunURL string `json:"run_url,omitempty"`
}

// Dispatcher queues the fleet-scan workflow through GitHub's workflow
// dispatch API. Its token is a fine-grained token with Actions write on the
// Sonde repository only; an empty token turns rescans off.
type Dispatcher struct {
	Token   string
	BaseURL string
	Client  *http.Client
}

// Enabled reports whether a token is configured.
func (d Dispatcher) Enabled() bool { return d.Token != "" }

// Dispatch queues one run of the scan workflow on the default branch.
func (d Dispatcher) Dispatch(ctx context.Context) (ScanResult, error) {
	if !d.Enabled() {
		return ScanResult{}, errors.Failed("rescan is not configured: set SONDE_FLEET_DISPATCH_TOKEN")
	}
	request, err := d.request(ctx)
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

func (d Dispatcher) request(ctx context.Context) (*http.Request, error) {
	base := d.BaseURL
	if base == "" {
		base = defaultGithubAPI
	}
	url := fmt.Sprintf("%s/repos/%s/actions/workflows/%s/dispatches", base, scanRepository, scanWorkflow)
	payload, err := json.Marshal(map[string]string{"ref": scanRef})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+d.Token)
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
	case status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound:
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
