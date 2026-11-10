package httpapi

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Sonde/apps/api/internal/stats"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// maxIncidentDays bounds the incident histogram. Incidents are never pruned,
// so an unbounded `days` would let one request scan the whole table.
const maxIncidentDays = 365

const defaultIncidentDays = 30

// monitorsOverview answers the whole monitors page in one request: every
// monitor with its current status and 24-hour history, plus the fleet totals.
//
// It exists because the page it backs used to fire one /checks request per
// monitor from the browser, which is an N+1 over HTTP that gets worse with
// every monitor somebody adds.
func (s *Service) monitorsOverview(w http.ResponseWriter, r *http.Request) {
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	fleet, err := s.stats.Fleet(r.Context(), ownerID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, fleet)
}

// monitorHistory returns one monitor's uptime and latency series over a
// window, read from the rollup tables rather than reassembled from raw checks.
func (s *Service) monitorHistory(w http.ResponseWriter, r *http.Request) {
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	name := r.URL.Query().Get("window")
	if name == "" {
		name = "24h"
	}
	window, ok := stats.LookupWindow(name)
	if !ok {
		httpjson.WriteError(w, errors.Invalid("window must be one of 24h, 7d, 30d, 90d"))
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if _, err := s.monitors.Get(r.Context(), ownerID, id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	series, err := s.stats.Buckets(r.Context(), id, window)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, series)
}

// monitorUptimeSummary returns all four windows at once. The service already
// computed them together; the single-window route threw three away and made
// the detail page ask four times.
func (s *Service) monitorUptimeSummary(w http.ResponseWriter, r *http.Request) {
	id, err := monitorID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if _, err := s.monitors.Get(r.Context(), ownerID, id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	summary, err := s.uptime.Summary(r.Context(), id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, summary)
}

// incidentStats aggregates the caller's incidents over a day window.
func (s *Service) incidentStats(w http.ResponseWriter, r *http.Request) {
	ownerID, err := s.ownerID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	days := defaultIncidentDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > maxIncidentDays {
			httpjson.WriteError(w, errors.Invalid("days must be between 1 and 365"))
			return
		}
		days = parsed
	}
	summary, err := s.stats.Incidents(r.Context(), ownerID, days)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, summary)
}

// IntegrationsResponse is what this instance is wired to. It carries no
// secrets: the Antenne shared secret, the Journal token and the OIDC client
// secret are all deliberately absent, and adding one here would sync it into
// every browser that opens settings.
type IntegrationsResponse struct {
	Bus       BusResponse       `json:"bus"`
	Journal   JournalResponse   `json:"journal"`
	Retention RetentionResponse `json:"retention"`
}

// BusResponse is the Antenne connection as an operator reads it. The state is
// not a boolean: "not connected" hides three situations with three different
// fixes, so disabled, connecting, connected and reconnecting are distinct.
type BusResponse struct {
	Configured  bool     `json:"configured"`
	State       string   `json:"state"`
	Instance    string   `json:"instance"`
	App         string   `json:"app"`
	InstanceID  string   `json:"instance_id,omitempty"`
	Channels    []string `json:"channels"`
	Published   int64    `json:"published"`
	Pending     int64    `json:"pending"`
	LastEventAt *string  `json:"last_event_at"`
	LastError   string   `json:"last_error,omitempty"`
}

// JournalResponse reports whether logs are shipped off this container.
type JournalResponse struct {
	Configured bool   `json:"configured"`
	URL        string `json:"url,omitempty"`
}

// RetentionResponse is how far back this instance can answer, and how many
// consecutive failures open an incident.
type RetentionResponse struct {
	RawDays       int `json:"raw_days"`
	HourlyMonths  int `json:"hourly_months"`
	FailThreshold int `json:"failure_threshold"`
}

func (s *Service) integrations(w http.ResponseWriter, r *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, s.integrationsSnapshot(r.Context()))
}
