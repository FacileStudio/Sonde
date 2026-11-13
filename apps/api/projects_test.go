package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/Sonde/apps/api/internal/fleet"
	"github.com/FacileStudio/Sonde/apps/api/internal/githubapp"
	"github.com/FacileStudio/Sonde/apps/api/internal/httpapi"
	"github.com/FacileStudio/Sonde/apps/api/internal/projects"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func fakeGitHubApp(t *testing.T) *githubapp.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /app/installations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":9,"account":{"login":"FacileStudio"}}]`))
	})
	mux.HandleFunc("POST /app/installations/9/access_tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":"ghs_test","expires_at":"2999-01-01T00:00:00Z"}`))
	})
	mux.HandleFunc("GET /user/42", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"login":"octo"}`))
	})
	mux.HandleFunc("GET /repos/FacileStudio/{repo}/collaborators/octo/permission", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("repo") != "Courrier" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"permission":"read"}`))
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

func projectsRouter(t *testing.T, app *githubapp.Client) (chi.Router, *gorm.DB) {
	t.Helper()
	db := openAuthDatabase(t)
	router, err := buildRouter(routerDeps{
		DB: db, SQL: rawSQL(t, db), Users: userStore(t, db), GitHub: app,
		Env: env.Config{FleetAdmins: fleetAdmins}, Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("buildRouter: %v", err)
	}
	for id, name := range map[int64]string{101: "Courrier", 102: "Nuage"} {
		if err := db.Exec(`INSERT INTO projects (github_repo_id, name) VALUES (?, ?) ON CONFLICT (github_repo_id) DO NOTHING`, id, name).Error; err != nil {
			t.Fatalf("project: %v", err)
		}
	}
	return router, db
}

func projectID(t *testing.T, db *gorm.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.Raw(`SELECT id FROM projects WHERE name = ?`, name).Scan(&id).Error; err != nil || id == 0 {
		t.Fatalf("project %s: %v", name, err)
	}
	return id
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", response.Body, err)
	}
	return out
}

func maintenanceShows(t *testing.T, router chi.Router, token, project, monitor string) {
	t.Helper()
	for _, row := range readFleet[[]fleet.ProjectResponse](t, router, "/api/fleet/projects", token) {
		if row.Name == project && (len(row.Monitors) != 1 || row.Monitors[0].Name != monitor) {
			t.Fatalf("maintenance shows %+v for %s", row.Monitors, project)
		}
	}
}

func TestSignInStoresTheGitHubAccountRegistreReports(t *testing.T) {
	db := openAuthDatabase(t)
	ctx := context.Background()
	withGitHub := claims("sub-gh", "gh@facile.studio", "GH", true)
	withGitHub.Extra = map[string]json.RawMessage{"connections": json.RawMessage(`{"github":"42"}`)}
	userID, err := userStore(t, db).UpsertFromOIDC(ctx, withGitHub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := userStore(t, db).UpsertFromOIDC(ctx, claims("sub-gh", "gh@facile.studio", "GH", true)); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.Raw(`SELECT github_id FROM users WHERE id = ?`, userID).Scan(&stored).Error; err != nil || stored != "42" {
		t.Fatalf("github_id = %v, %v: a sign-in without the claim must keep it", stored, err)
	}
}

func TestProjectsAnswerAPlainPreconditionUntilGitHubIsSetUp(t *testing.T) {
	router, db := projectsRouter(t, nil)
	admin := signIn(t, db, "admin@facile.studio")
	person := signIn(t, db, "person@facile.studio")

	refused := call(t, router, http.MethodGet, "/api/projects", cookieHeader(person.Token), nil)
	if refused.Code != http.StatusPreconditionFailed || !strings.Contains(refused.Body.String(), "SONDE_GITHUB_APP") {
		t.Fatalf("no App = %d: %s", refused.Code, refused.Body)
	}
	if all := decode[[]projects.Ref](t, call(t, router, http.MethodGet, "/api/projects", cookieHeader(admin.Token), nil)); len(all) != 2 {
		t.Fatalf("fleet admin sees %+v, want both projects", all)
	}

	router, db = projectsRouter(t, fakeGitHubApp(t))
	person = signIn(t, db, "person@facile.studio")
	unlinked := call(t, router, http.MethodGet, "/api/projects", cookieHeader(person.Token), nil)
	if unlinked.Code != http.StatusPreconditionFailed || !strings.Contains(unlinked.Body.String(), "sso.facile.studio/account") {
		t.Fatalf("no GitHub account = %d: %s", unlinked.Code, unlinked.Body)
	}
}

func TestLinkingAMonitorFollowsGitHubReadAccess(t *testing.T) {
	router, db := projectsRouter(t, fakeGitHubApp(t))
	admin := signIn(t, db, "admin@facile.studio")
	person := signIn(t, db, "person@facile.studio")
	db.Exec(`UPDATE users SET github_id = '42' WHERE id = ?`, person.User.ID)
	headers := csrfHeaders(person.Token)
	if listed := decode[[]projects.Ref](t, call(t, router, http.MethodGet, "/api/projects", headers, nil)); len(listed) != 1 || listed[0].Name != "Courrier" {
		t.Fatalf("listed %+v, want Courrier only", listed)
	}
	created := decode[httpapi.MonitorResponse](t, call(t, router, http.MethodPost, "/api/monitors", headers, map[string]any{
		"slug": "mail", "name": "Mail", "type": "http", "target": "https://mail.example.com",
	}))
	path := "/api/monitors/" + intToStr(created.ID) + "/project"
	if got := call(t, router, http.MethodPut, path, headers, map[string]any{"project_id": projectID(t, db, "Nuage")}); got.Code != http.StatusForbidden {
		t.Fatalf("link to an unreadable repo = %d: %s", got.Code, got.Body)
	}
	linked := decode[httpapi.MonitorResponse](t, call(t, router, http.MethodPut, path, headers, map[string]any{"project_id": projectID(t, db, "Courrier")}))
	if linked.Project == nil || linked.Project.Name != "Courrier" {
		t.Fatalf("linked = %+v", linked)
	}
	edited := call(t, router, http.MethodPatch, "/api/monitors/"+intToStr(created.ID), headers, map[string]any{
		"slug": "mail", "name": "Mail renamed", "type": "http", "target": "https://mail.example.com", "enabled": true,
	})
	if kept := decode[httpapi.MonitorResponse](t, edited); kept.Project == nil || kept.Project.Name != "Courrier" {
		t.Fatalf("an edit dropped the link: %s", edited.Body)
	}
	maintenanceShows(t, router, admin.Token, "Courrier", "Mail renamed")
	if cleared := decode[httpapi.MonitorResponse](t, call(t, router, http.MethodPut, path, headers, map[string]any{"project_id": nil})); cleared.Project != nil {
		t.Fatalf("unlink kept %+v", cleared.Project)
	}
}
