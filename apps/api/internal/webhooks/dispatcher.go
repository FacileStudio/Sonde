package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/checker"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/envelope"
)

const (
	defaultRequestTimeout = 10 * time.Second
)

type WebhookStore interface {
	ListEnabledForOwner(ctx context.Context, ownerID int64) ([]schemas.Webhook, error)
}

type MonitorStore interface {
	OwnerEmail(ctx context.Context, monitorID int64) (string, error)
	OwnerID(ctx context.Context, monitorID int64) (int64, error)
}

type Payload struct {
	envelope.Monitor
	ActorEmail string `json:"actor_email,omitempty"`
}

type TestResult struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type Dispatcher struct {
	webhooks WebhookStore
	monitors MonitorStore
	client   *http.Client
	logger   *slog.Logger
}

func NewDispatcher(webhooks WebhookStore, monitors MonitorStore, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		webhooks: webhooks,
		monitors: monitors,
		client:   &http.Client{Timeout: defaultRequestTimeout},
		logger:   logger,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, transition checker.Transition) {
	if d == nil || d.webhooks == nil || d.monitors == nil {
		return
	}

	ownerID, err := d.monitors.OwnerID(ctx, transition.MonitorID)
	if err != nil || ownerID == 0 {
		return
	}

	targets, err := d.webhooks.ListEnabledForOwner(ctx, ownerID)
	if err != nil || len(targets) == 0 {
		return
	}

	email, _ := d.monitors.OwnerEmail(ctx, transition.MonitorID)
	body, err := d.buildPayload(transition, email)
	if err != nil {
		d.logger.Warn("failed to serialize webhook payload", slog.Any("error", err))
		return
	}

	for _, target := range targets {
		go d.send(context.WithoutCancel(ctx), target, body, transition.MonitorID)
	}
}

func (d *Dispatcher) Test(ctx context.Context, target schemas.Webhook) (*TestResult, error) {
	sampleTransition := checker.Transition{
		MonitorID:  0,
		Slug:       "test-monitor",
		Name:       "Test Monitor",
		Up:         false,
		LatencyMS:  45,
		Error:      "test probe simulated failure",
		IncidentAt: time.Now().UTC(),
	}

	body, err := d.buildPayload(sampleTransition, "test@sonde.local")
	if err != nil {
		return nil, err
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sonde/1.0")

	if target.Secret != "" {
		sig := signHMAC(body, target.Secret)
		header := target.SecretHeader
		if header == "" {
			header = DefaultSecretHeader
		}
		req.Header.Set(header, sig)
	}

	resp, err := d.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &TestResult{
			OK:        false,
			LatencyMS: latency,
			Error:     err.Error(),
		}, nil
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	result := &TestResult{
		OK:         ok,
		StatusCode: resp.StatusCode,
		LatencyMS:  latency,
	}
	if !ok {
		result.Error = fmt.Sprintf("endpoint returned status %d", resp.StatusCode)
	}
	return result, nil
}

func (d *Dispatcher) send(ctx context.Context, target schemas.Webhook, body []byte, monitorID int64) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(body))
	if err != nil {
		d.logger.Warn("failed to create webhook request", slog.Any("error", err), slog.Int64("webhook_id", target.ID))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sonde/1.0")

	if target.Secret != "" {
		sig := signHMAC(body, target.Secret)
		header := target.SecretHeader
		if header == "" {
			header = DefaultSecretHeader
		}
		req.Header.Set(header, sig)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		d.logger.Warn("webhook delivery failed",
			slog.Any("error", err),
			slog.Int64("webhook_id", target.ID),
			slog.Int64("monitor_id", monitorID))
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		d.logger.Info("webhook delivered",
			slog.Int64("webhook_id", target.ID),
			slog.Int64("monitor_id", monitorID),
			slog.Int("status", resp.StatusCode))
	} else {
		d.logger.Warn("webhook rejected",
			slog.Int64("webhook_id", target.ID),
			slog.Int64("monitor_id", monitorID),
			slog.Int("status", resp.StatusCode))
	}
}

func (d *Dispatcher) buildPayload(transition checker.Transition, actorEmail string) ([]byte, error) {
	status := "down"
	if transition.Up {
		status = "up"
	}

	var failure *string
	if transition.Error != "" {
		failure = &transition.Error
	}

	payload := Payload{
		Monitor: envelope.Monitor{
			MonitorID: strconv.FormatInt(transition.MonitorID, 10),
			FacileID:  transition.Slug,
			Slug:      transition.Slug,
			Name:      transition.Name,
			Status:    status,
			LatencyMS: transition.LatencyMS,
			Error:     failure,
		},
		ActorEmail: actorEmail,
	}

	event := envelope.Event[Payload]{
		Version:        envelope.EventVersion,
		App:            envelope.AppSonde,
		Object:         envelope.ObjectMonitor,
		Action:         envelope.ActionUpdated,
		FacileID:       payload.FacileID,
		Payload:        payload,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		IdempotencyKey: payload.MonitorID + ":" + status + ":" + strconv.FormatInt(transition.IncidentAt.UnixMilli(), 10),
	}

	return json.Marshal(event)
}

// signHMAC signs the body the way every receiver in the suite verifies it:
// the algorithm named, then the hex digest.
//
// The prefix is not decoration. Antenne verifies with
// envelope.VerifyHexSignature(secret, "sha256=", …) and compares the whole
// string, so a bare digest was rejected 401 by the one target this feature
// exists to reach — and only appeared to work when the Antenne provider had no
// secret set, which is the configuration where nothing is verified at all.
// GitHub, Stripe and Antenne's own outbound signatures all carry it.
func signHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
