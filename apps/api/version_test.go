package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestVersionMatchesTheVersionFile(t *testing.T) {
	if version == "dev" {
		t.Skip("built without -X version; scripts/check.sh sets it from VERSION")
	}
	raw, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	want := "v" + strings.TrimSpace(string(raw))
	rec := httptest.NewRecorder()
	versionHandler(rec, httptest.NewRequest(http.MethodGet, "/version", nil))
	if want != version || !strings.Contains(rec.Body.String(), `"version":"`+want+`"`) {
		t.Fatalf("/version = %s, VERSION file says %s", rec.Body.String(), want)
	}
}
