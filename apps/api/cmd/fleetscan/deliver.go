package main

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
)

func encodeReport(report fleet.Report) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if err := json.NewEncoder(writer).Encode(report); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// deliveryError carries the status only: the reply can name a project, and the
// workflow log is public.
type deliveryError struct{ status int }

func (e deliveryError) Error() string { return fmt.Sprintf("sonde answered %d", e.status) }

func deliver(ctx context.Context, cfg config, report fleet.Report) (fleet.IngestResult, error) {
	body, err := encodeReport(report)
	if err != nil {
		return fleet.IngestResult{}, err
	}
	var result fleet.IngestResult
	for try := range deliveryAttempts {
		result, err = post(ctx, cfg, body)
		if stored := alreadyStored(try, err); stored != nil || final(try, err) {
			return result, cmp.Or(stored, err)
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(time.Duration(try+1) * cfg.retryWait):
		}
	}
	return result, err
}

// alreadyStored recognises a retry refused as a duplicate: an earlier attempt
// was stored and only its reply was lost.
func alreadyStored(try int, err error) error {
	var refused deliveryError
	if try > 0 && errors.As(err, &refused) && refused.status == http.StatusConflict {
		return errAlreadyStored
	}
	return nil
}

// final reports an attempt not worth repeating: a success, a refusal, or the
// last attempt.
func final(try int, err error) bool {
	var refused deliveryError
	return err == nil || (errors.As(err, &refused) && refused.status < 500) || try == deliveryAttempts-1
}

func post(ctx context.Context, cfg config, body []byte) (fleet.IngestResult, error) {
	var result fleet.IngestResult
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.sondeURL, "/")+"/api/fleet/reports", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	request.Header.Set("Authorization", "Bearer "+cfg.fleetToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Do(request)
	if err != nil {
		return result, fmt.Errorf("deliver: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	reply, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode != http.StatusOK {
		return result, deliveryError{status: response.StatusCode}
	}
	return result, json.Unmarshal(reply, &result)
}
