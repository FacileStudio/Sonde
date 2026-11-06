package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/testdb"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func liveAPI(t *testing.T) (chi.Router, *gorm.DB) {
	t.Helper()
	url, ok := testdb.URL()
	if !ok {
		testdb.Announce("sh scripts/postgres-up.sh && export " + testdb.EnvVar + "=postgres://postgres:postgres@127.0.0.1:5603/sonde_test?sslmode=disable")
		t.Skip(testdb.SkipReason("sh scripts/postgres-up.sh"))
	}
	db, err := testdb.Open(url, testdb.Config{Prefix: "sonde_auth_test", Migrate: applySchema})
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	t.Cleanup(func() {
		if err := testdb.Truncate(db, testdb.Config{Prefix: "sonde_auth_test"}); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	})
	router, err := buildRouter(routerDeps{DB: db, SQL: rawSQL(t, db), Env: env.Config{}, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("buildRouter: %v", err)
	}
	return router, db
}

func csrfHeaders(token string) map[string]string {
	return map[string]string{
		"Cookie":        porte.SessionCookieName + "=" + token,
		"X-Facile-CSRF": "1",
	}
}

func adminSession(t *testing.T, db *gorm.DB) signedSession {
	t.Helper()
	return signIn(t, db, "admin@facile.studio")
}

func TestMonitorRoutesRejectAnonymousCallers(t *testing.T) {
	router, _ := liveAPI(t)

	for _, probe := range []struct{ method, path string }{
		{http.MethodPost, "/api/monitors"},
		{http.MethodGet, "/api/monitors"},
		{http.MethodGet, "/api/monitors/1"},
		{http.MethodPatch, "/api/monitors/1"},
		{http.MethodDelete, "/api/monitors/1"},
		{http.MethodGet, "/api/monitors/1/incidents"},
		{http.MethodGet, "/api/monitors/1/uptime"},
		{http.MethodGet, "/api/incidents"},
		{http.MethodGet, "/api/config/export"},
		{http.MethodGet, "/api/status-pages"},
	} {
		response := call(t, router, probe.method, probe.path, nil, map[string]any{})
		if response.Code != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous = %d, want 401", probe.method, probe.path, response.Code)
		}
	}
}

func TestExportAndStatusPagesRequireTheAdminRole(t *testing.T) {
	router, db := liveAPI(t)
	admin := adminSession(t, db)
	plain := signIn(t, db, "plain@facile.studio")

	for _, probe := range []struct{ method, path string }{
		{http.MethodGet, "/api/config/export"},
		{http.MethodPost, "/api/status-pages"},
		{http.MethodGet, "/api/status-pages"},
		{http.MethodDelete, "/api/status-pages/999"},
	} {
		headers := cookieHeader(plain.Token)
		headers["X-Facile-CSRF"] = "1"
		response := call(t, router, probe.method, probe.path, headers, map[string]any{})
		if response.Code != http.StatusForbidden {
			t.Errorf("%s %s as a non-admin = %d, want 403", probe.method, probe.path, response.Code)
		}
	}

	if ok := call(t, router, http.MethodGet, "/api/config/export", cookieHeader(admin.Token), nil); ok.Code != http.StatusOK {
		t.Fatalf("export as admin = %d: %s", ok.Code, ok.Body)
	}
}

func TestMonitorCRUDAndPerTypeValidation(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, db)
	writeHeaders := csrfHeaders(session.Token)

	created := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, map[string]any{
		"slug": "website", "name": "Website", "type": "http", "target": "https://facile.studio",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body)
	}
	var monitor struct {
		ID    int64  `json:"id"`
		Slug  string `json:"slug"`
		Token string `json:"push_token"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &monitor); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if monitor.Slug != "website" || monitor.Token != "" {
		t.Fatalf("an http monitor must not carry a push token: %+v", monitor)
	}

	for _, bad := range []map[string]any{
		{"slug": "Bad Slug", "name": "n", "type": "http", "target": "https://x.example"},
		{"slug": "short-interval", "name": "n", "type": "http", "target": "https://x.example", "interval_seconds": 5},
		{"slug": "bad-http", "name": "n", "type": "http", "target": "ftp://x.example"},
		{"slug": "bad-tcp", "name": "n", "type": "tcp", "target": "host-only-no-port"},
		{"slug": "bad-push", "name": "n", "type": "push", "target": "https://x.example"},
		{"slug": "unknown-type", "name": "n", "type": "icmp", "target": "host"},
	} {
		response := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, bad)
		if response.Code != http.StatusBadRequest {
			t.Errorf("create %+v = %d, want 400: %s", bad["slug"], response.Code, response.Body)
		}
	}

	duplicate := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, map[string]any{
		"slug": "website", "name": "Again", "type": "tcp", "target": "localhost:5432",
	})
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate slug = %d, want 409", duplicate.Code)
	}

	listed := call(t, router, http.MethodGet, "/api/monitors", cookieHeader(session.Token), nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list = %d: %s", listed.Code, listed.Body)
	}
	var monitors []map[string]any
	if err := json.Unmarshal(listed.Body.Bytes(), &monitors); err != nil || len(monitors) != 1 {
		t.Fatalf("list body: %s (%v)", listed.Body, err)
	}

	patched := call(t, router, http.MethodPatch, "/api/monitors/"+intToStr(monitor.ID), writeHeaders, map[string]any{
		"slug": "website", "name": "Renamed", "type": "http",
		"target": "https://facile.studio", "enabled": false,
	})
	if patched.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", patched.Code, patched.Body)
	}
	if !strings.Contains(patched.Body.String(), `"enabled":false`) {
		t.Errorf("the patch did not disable the monitor: %s", patched.Body)
	}

	foreign := signIn(t, db, "other@facile.studio")
	if stolen := call(t, router, http.MethodGet, "/api/monitors/"+intToStr(monitor.ID), cookieHeader(foreign.Token), nil); stolen.Code != http.StatusNotFound {
		t.Errorf("another user's monitor read = %d, want 404", stolen.Code)
	}

	if gone := call(t, router, http.MethodDelete, "/api/monitors/"+intToStr(monitor.ID), writeHeaders, nil); gone.Code != http.StatusNoContent {
		t.Fatalf("delete = %d: %s", gone.Code, gone.Body)
	}
	if after := call(t, router, http.MethodGet, "/api/monitors/"+intToStr(monitor.ID), cookieHeader(session.Token), nil); after.Code != http.StatusNotFound {
		t.Errorf("the monitor survived deletion: %d", after.Code)
	}
}

func TestCookieMutationWithoutCSRFIsRefusedOnMonitors(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, db)

	refused := call(t, router, http.MethodPost, "/api/monitors", cookieHeader(session.Token), map[string]any{
		"slug": "no-csrf", "name": "n", "type": "push",
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("cookie mutation without CSRF = %d, want 403: %s", refused.Code, refused.Body)
	}
}

// The export is import-shaped config, not a backup of the world: a push token
// or any other credential that slips into it turns a snapshot into a leak.
func TestConfigExportContainsNoSecrets(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, db)
	writeHeaders := csrfHeaders(session.Token)

	pushCreated := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, map[string]any{
		"slug": "cron-job", "name": "Cron job", "type": "push",
	})
	var pushMonitor struct {
		PushToken string `json:"push_token"`
	}
	if err := json.Unmarshal(pushCreated.Body.Bytes(), &pushMonitor); err != nil || pushMonitor.PushToken == "" {
		t.Fatalf("push creation did not mint a token: %s", pushCreated.Body)
	}

	call(t, router, http.MethodPost, "/api/status-pages", writeHeaders, map[string]any{
		"slug": "acme", "title": "Acme status", "monitor_ids": []int64{},
	})

	exported := call(t, router, http.MethodGet, "/api/config/export", cookieHeader(session.Token), nil)
	if exported.Code != http.StatusOK {
		t.Fatalf("export = %d: %s", exported.Code, exported.Body)
	}
	body := exported.Body.String()
	if strings.Contains(body, pushMonitor.PushToken) {
		t.Fatal("the export leaked the push token")
	}
	if strings.Contains(body, "push_token") || strings.Contains(body, "owner_id") || strings.Contains(body, "password") {
		t.Fatalf("the export carries secret-shaped fields: %s", body)
	}
}

func TestIncidentsAndUptimeServeTheOwnersOnly(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, db)
	writeHeaders := csrfHeaders(session.Token)

	created := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, map[string]any{
		"slug": "db", "name": "DB", "type": "tcp", "target": "127.0.0.1:5603",
	})
	var monitor struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &monitor); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	if err := db.WithContext(context.Background()).Create(&schemas.CheckRaw{
		MonitorID: monitor.ID, OK: false, Error: "connection refused", CreatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed check: %v", err)
	}
	if err := db.WithContext(context.Background()).Create(&schemas.Incident{
		MonitorID: monitor.ID, OpenedAt: time.Now(), Cause: "connection refused",
	}).Error; err != nil {
		t.Fatalf("seed incident: %v", err)
	}
	if err := db.WithContext(context.Background()).Exec(
		`INSERT INTO checks_hourly (monitor_id, bucket_start, total, failed, avg_latency_ms)
         VALUES (?, date_trunc('hour', now()), 10, 2, 15)`, monitor.ID,
	).Error; err != nil {
		t.Fatalf("seed rollup: %v", err)
	}

	incidents := call(t, router, http.MethodGet, "/api/incidents", cookieHeader(session.Token), nil)
	if incidents.Code != http.StatusOK || !strings.Contains(incidents.Body.String(), "connection refused") {
		t.Fatalf("incidents = %d: %s", incidents.Code, incidents.Body)
	}
	filtered := call(t, router, http.MethodGet, "/api/incidents?monitor_id="+intToStr(monitor.ID), cookieHeader(session.Token), nil)
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered incidents = %d: %s", filtered.Code, filtered.Body)
	}
	badFilter := call(t, router, http.MethodGet, "/api/incidents?monitor_id=abc", cookieHeader(session.Token), nil)
	if badFilter.Code != http.StatusBadRequest {
		t.Errorf("a garbage monitor_id filter = %d, want 400", badFilter.Code)
	}

	uptimeResponse := call(t, router, http.MethodGet, "/api/monitors/"+intToStr(monitor.ID)+"/uptime?window=24h", cookieHeader(session.Token), nil)
	if uptimeResponse.Code != http.StatusOK {
		t.Fatalf("uptime = %d: %s", uptimeResponse.Code, uptimeResponse.Body)
	}
	var stat struct {
		Window string   `json:"window"`
		Uptime *float64 `json:"uptime"`
	}
	if err := json.Unmarshal(uptimeResponse.Body.Bytes(), &stat); err != nil {
		t.Fatalf("decode uptime: %v", err)
	}
	if stat.Window != "24h" || stat.Uptime == nil || *stat.Uptime != 80 {
		t.Fatalf("uptime stat wrong: %+v", stat)
	}
	if badWindow := call(t, router, http.MethodGet, "/api/monitors/"+intToStr(monitor.ID)+"/uptime?window=13m", cookieHeader(session.Token), nil); badWindow.Code != http.StatusBadRequest {
		t.Errorf("an unknown window = %d, want 400", badWindow.Code)
	}
}

// The public page is the one route the internet may hit logged-out; it must
// answer and it must not carry owner ids or push tokens out with it.
func TestPublicStatusPageServesLoggedOutWithoutPrivateFields(t *testing.T) {
	router, db := liveAPI(t)
	session := adminSession(t, db)
	writeHeaders := csrfHeaders(session.Token)

	httpMonitor := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, map[string]any{
		"slug": "site", "name": "Site", "type": "http", "target": "https://example.com",
	})
	var site struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(httpMonitor.Body.Bytes(), &site); err != nil {
		t.Fatalf("decode: %v", err)
	}
	pushMonitor := call(t, router, http.MethodPost, "/api/monitors", writeHeaders, map[string]any{
		"slug": "cron", "name": "Cron", "type": "push",
	})
	var cron struct {
		ID        int64  `json:"id"`
		PushToken string `json:"push_token"`
	}
	if err := json.Unmarshal(pushMonitor.Body.Bytes(), &cron); err != nil {
		t.Fatalf("decode push: %v", err)
	}

	pageCreated := call(t, router, http.MethodPost, "/api/status-pages", writeHeaders, map[string]any{
		"slug": "acme", "title": "Acme status", "monitor_ids": []int64{site.ID, cron.ID},
	})
	if pageCreated.Code != http.StatusCreated {
		t.Fatalf("status page create = %d: %s", pageCreated.Code, pageCreated.Body)
	}

	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&schemas.CheckRaw{MonitorID: site.ID, OK: true, CreatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("seed check: %v", err)
	}
	if err := db.WithContext(ctx).Create(&schemas.CheckRaw{MonitorID: cron.ID, OK: false, Error: "heartbeat missing", CreatedAt: time.Now()}).Error; err != nil {
		t.Fatalf("seed check: %v", err)
	}
	if err := db.WithContext(ctx).Create(&schemas.Incident{MonitorID: cron.ID, OpenedAt: time.Now(), Cause: "heartbeat missing"}).Error; err != nil {
		t.Fatalf("seed incident: %v", err)
	}

	public := call(t, router, http.MethodGet, "/api/public/status/acme", nil, nil)
	if public.Code != http.StatusOK {
		t.Fatalf("public page = %d: %s", public.Code, public.Body)
	}
	body := public.Body.String()
	if !strings.Contains(body, `"title":"Acme status"`) || !strings.Contains(body, `"current_status":"down"`) {
		t.Fatalf("public page content wrong: %s", body)
	}
	if strings.Contains(body, cron.PushToken) || strings.Contains(body, "push_token") ||
		strings.Contains(body, "owner_id") || strings.Contains(body, session.Token) {
		t.Fatalf("the public page leaks private data: %s", body)
	}

	var page struct {
		Slug     string `json:"slug"`
		Title    string `json:"title"`
		Monitors []struct {
			ID            int64    `json:"id"`
			Slug          string   `json:"slug"`
			Name          string   `json:"name"`
			CurrentStatus string   `json:"current_status"`
			Uptime24h     *float64 `json:"uptime_24h"`
			Uptime7d      *float64 `json:"uptime_7d"`
			OpenIncidents []struct {
				ID       int64  `json:"id"`
				OpenedAt string `json:"opened_at"`
				Cause    string `json:"cause"`
			} `json:"open_incidents"`
		} `json:"monitors"`
	}
	if err := json.Unmarshal(public.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode public page: %v", err)
	}
	if page.Slug != "acme" || len(page.Monitors) != 2 {
		t.Fatalf("page shape wrong: %+v", page)
	}
	if page.Monitors[0].Slug == "cron" {
		page.Monitors[0], page.Monitors[1] = page.Monitors[1], page.Monitors[0]
	}
	siteLine, cronLine := page.Monitors[0], page.Monitors[1]
	if siteLine.CurrentStatus != "up" {
		t.Fatalf("site line wrong: %+v", siteLine)
	}
	if cronLine.CurrentStatus != "down" || len(cronLine.OpenIncidents) != 1 || cronLine.OpenIncidents[0].Cause != "heartbeat missing" {
		t.Fatalf("cron line wrong: %+v", cronLine)
	}
	_ = cronLine
	_ = siteLine

	if missing := call(t, router, http.MethodGet, "/api/public/status/nobody", nil, nil); missing.Code != http.StatusNotFound {
		t.Errorf("an unknown slug = %d, want 404", missing.Code)
	}
}

func intToStr(id int64) string {
	return strconv.FormatInt(id, 10)
}
