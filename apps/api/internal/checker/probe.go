package checker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
)

const maxBodyBytes = 1 << 20

type probeResult struct {
	ok           bool
	latencyMS    int64
	errText      string
	tlsExpiresAt *time.Time
}

func (e *Engine) probe(ctx context.Context, state *monitorState) {
	var result probeResult
	switch state.monitor.Type {
	case "http":
		result = e.probeHTTP(ctx, state.monitor)
	case "tcp":
		result = e.probeTCP(ctx, state.monitor)
	default:
		return
	}
	e.applyResult(ctx, state, result)
}

func (e *Engine) probeHTTP(ctx context.Context, monitor schemas.Monitor) probeResult {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(monitor.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, monitor.Target, nil)
	if err != nil {
		return probeResult{errText: err.Error()}
	}
	response, err := e.client.Do(request)
	if err != nil {
		return probeResult{errText: err.Error(), latencyMS: elapsed(start)}
	}
	defer func() {
		if _, copyErr := io.Copy(io.Discard, response.Body); copyErr != nil {
			e.logger.Debug("draining body", slog.Any("error", copyErr))
		}
		_ = response.Body.Close()
	}()

	latency := elapsed(start)

	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
	if err != nil {
		return probeResult{ok: false, latencyMS: latency, errText: err.Error()}
	}

	ok := response.StatusCode == monitor.ExpectedStatus
	if ok && monitor.ExpectedKeyword != "" && !strings.Contains(string(body), monitor.ExpectedKeyword) {
		ok = false
	}

	var tlsExpiry *time.Time
	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		notAfter := response.TLS.PeerCertificates[0].NotAfter
		tlsExpiry = &notAfter
	}

	if !ok {
		return probeResult{ok: false, latencyMS: latency, errText: "unexpected status " + response.Status, tlsExpiresAt: tlsExpiry}
	}
	if monitor.ExpectedKeyword != "" {
		return probeResult{ok: true, latencyMS: latency, tlsExpiresAt: tlsExpiry}
	}
	return probeResult{ok: true, latencyMS: latency, tlsExpiresAt: tlsExpiry}
}

func (e *Engine) probeTCP(ctx context.Context, monitor schemas.Monitor) probeResult {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(monitor.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	connection, err := e.dialer.DialContext(ctx, "tcp", monitor.Target)
	latency := elapsed(start)
	if err != nil {
		return probeResult{ok: false, latencyMS: latency, errText: err.Error()}
	}
	_ = connection.Close()
	return probeResult{ok: true, latencyMS: latency}
}

func elapsed(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
