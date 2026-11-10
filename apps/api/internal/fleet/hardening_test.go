package fleet

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/tronc/errors"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestShrankFlagsANarrowedListing(t *testing.T) {
	cases := []struct {
		listed, covered int
		want            bool
	}{{88, 88, false}, {83, 88, false}, {82, 88, true}, {1, 88, true}, {3, 3, false}, {1, 0, false}, {94, 100, true}, {9, 10, false}, {8, 10, true}}
	for _, c := range cases {
		if got := shrank(c.listed, c.covered); got != c.want {
			t.Errorf("shrank(%d, %d) = %v, want %v", c.listed, c.covered, got, c.want)
		}
	}
}

func TestALockTimeoutIsARetryableUnavailable(t *testing.T) {
	err := storeError(&pgconn.PgError{Code: "55P03"})

	if errors.Status(err) != http.StatusServiceUnavailable {
		t.Fatalf("status = %d: a lock wait that ran out is worth retrying", errors.Status(err))
	}
}

func TestDeliveriesAndRefusedTokensAreLogged(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, nil))
	h := NewHandler(nil, logger)
	h.logDelivery(IngestResult{RunID: 4, Projects: 88, Covered: 87, CoverageDropped: true})
	h.logRefusal(errors.Internal("failed to store the fleet report", http.ErrHandlerTimeout))

	request := httptest.NewRequest(http.MethodPost, "/reports", nil)
	request.Header.Set("Authorization", "Bearer wrong")
	IngestAuth("right-token-right-token-right-token", logger)(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), request)

	for _, want := range []string{"run_id=4", "covered=87", "far fewer projects", "cause=", "refused a token", "remote_addr="} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("log lacks %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "right-token") || strings.Contains(out.String(), "wrong") {
		t.Fatalf("a token reached the log:\n%s", out.String())
	}
}
