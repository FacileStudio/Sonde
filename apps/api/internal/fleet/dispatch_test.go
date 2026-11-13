package fleet

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/githubapp"
	"github.com/FacileStudio/tronc/errors"
)

type dispatchCall struct {
	path, auth, ref string
	mint            map[string]any
}

func fakeApp(t *testing.T, dispatchStatus int, got *dispatchCall) *githubapp.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":3,"account":{"login":"FacileStudio"}}]`))
	})
	mux.HandleFunc("POST /app/installations/3/access_tokens", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got.mint)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":"ghs_t0k","expires_at":"2999-01-01T00:00:00Z"}`))
	})
	mux.HandleFunc("POST /repos/FacileStudio/Sonde/actions/workflows/fleet-scan.yml/dispatches", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Ref string `json:"ref"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		got.path, got.auth, got.ref = r.URL.Path, r.Header.Get("Authorization"), body.Ref
		w.WriteHeader(dispatchStatus)
		_, _ = w.Write([]byte(`{"workflow_run_id":1,"html_url":"https://github.com/run/1"}`))
	})
	return appAgainst(t, mux)
}

func appAgainst(t *testing.T, handler http.Handler) *githubapp.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	app, err := githubapp.New(githubapp.Config{
		ClientID: "Iv23test", PrivateKey: base64.StdEncoding.EncodeToString(pemKey), BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func TestDispatchQueuesTheScanWorkflowWithAScopedInstallationToken(t *testing.T) {
	var got dispatchCall
	result, err := Dispatcher{App: fakeApp(t, http.StatusOK, &got)}.Dispatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.auth != "Bearer ghs_t0k" || got.ref != "master" {
		t.Fatalf("auth %q ref %q", got.auth, got.ref)
	}
	repos, _ := json.Marshal(got.mint["repositories"])
	perms, _ := json.Marshal(got.mint["permissions"])
	if string(repos) != `["Sonde"]` || string(perms) != `{"actions":"write"}` {
		t.Fatalf("token asked for repositories %s permissions %s", repos, perms)
	}
	if !result.Queued || result.RunURL != "https://github.com/run/1" {
		t.Fatalf("result %+v", result)
	}
}

func TestDispatchRefusals(t *testing.T) {
	if _, err := (Dispatcher{}).Dispatch(context.Background()); errors.Status(err) != http.StatusPreconditionFailed {
		t.Fatalf("unconfigured: status %d", errors.Status(err))
	}
	var got dispatchCall
	_, err := Dispatcher{App: fakeApp(t, http.StatusForbidden, &got)}.Dispatch(context.Background())
	if errors.Status(err) != http.StatusPreconditionFailed || err.Error() != missingActions {
		t.Fatalf("403 from GitHub: %v", err)
	}
	for status, want := range map[int]int{
		http.StatusNoContent:           0,
		http.StatusNotFound:            http.StatusPreconditionFailed,
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
