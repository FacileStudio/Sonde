package fleet

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

const routeToken = "progress-token-progress-token-0123456789"

func progressRouter() chi.Router {
	router := chi.NewRouter()
	refuse := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	}
	RegisterRoutes(router, nil, refuse, Config{Token: routeToken})
	return router
}

func postProgress(router chi.Router, token, body string) int {
	request := httptest.NewRequest(http.MethodPost, "/fleet/scans/progress", strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder.Code
}

func TestProgressNeedsTheFleetTokenAndAStrictBody(t *testing.T) {
	router := progressRouter()
	valid := `{"run_id":7,"phase":"scanning","repos_done":3,"repos_total":94,"current":"Sonde"}`
	cases := []struct {
		token, body string
		want        int
	}{
		{"", valid, http.StatusUnauthorized},
		{"wrong", valid, http.StatusUnauthorized},
		{routeToken, valid, http.StatusNoContent},
		{routeToken, `{"run_id":7,"phase":"scanning","repos_done":3,"repos_total":94,"findings":[]}`, http.StatusBadRequest},
		{routeToken, `{"run_id":7,"phase":"napping","repos_done":0,"repos_total":1}`, http.StatusBadRequest},
		{routeToken, `{"run_id":7,"phase":"scanning","repos_done":5,"repos_total":4}`, http.StatusBadRequest},
		{routeToken, `{"run_id":0,"phase":"scanning","repos_done":0,"repos_total":4}`, http.StatusBadRequest},
		{routeToken, `{"run_id":7,"phase":"scanning","repos_done":0,"repos_total":4,"current":"a\nb"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		if got := postProgress(router, c.token, c.body); got != c.want {
			t.Errorf("token %q body %s: status %d, want %d", c.token, c.body, got, c.want)
		}
	}
}

func TestTheCurrentScanRouteSitsBehindTheGate(t *testing.T) {
	recorder := httptest.NewRecorder()
	progressRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/fleet/scans/current", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous current scan = %d, want 401", recorder.Code)
	}
}
