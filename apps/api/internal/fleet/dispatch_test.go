package fleet

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/tronc/errors"
)

func TestDispatchQueuesTheScanWorkflow(t *testing.T) {
	var gotPath, gotAuth, gotRef string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		var body struct {
			Ref string `json:"ref"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotRef = body.Ref
		_, _ = w.Write([]byte(`{"workflow_run_id":1,"html_url":"https://github.com/run/1"}`))
	}))
	defer server.Close()

	result, err := Dispatcher{Token: "t0k", BaseURL: server.URL}.Dispatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/repos/FacileStudio/Sonde/actions/workflows/fleet-scan.yml/dispatches" || gotAuth != "Bearer t0k" || gotRef != "master" {
		t.Fatalf("path %q auth %q ref %q", gotPath, gotAuth, gotRef)
	}
	if !result.Queued || result.RunURL != "https://github.com/run/1" {
		t.Fatalf("result %+v", result)
	}
}

func TestDispatchRefusals(t *testing.T) {
	if _, err := (Dispatcher{}).Dispatch(context.Background()); errors.Status(err) != http.StatusPreconditionFailed {
		t.Fatalf("unconfigured: status %d", errors.Status(err))
	}
	for status, want := range map[int]int{
		http.StatusNoContent:           0,
		http.StatusForbidden:           http.StatusPreconditionFailed,
		http.StatusUnprocessableEntity: http.StatusServiceUnavailable,
	} {
		result, err := dispatchResult(status, nil)
		if want == 0 {
			if err != nil || !result.Queued {
				t.Fatalf("status %d: %+v %v", status, result, err)
			}
			continue
		}
		if errors.Status(err) != want {
			t.Fatalf("status %d mapped to %d, want %d", status, errors.Status(err), want)
		}
	}
}
