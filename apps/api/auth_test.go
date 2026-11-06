package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	auth "github.com/FacileStudio/Sonde/apps/api/modules/auth"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/testdb"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const testIssuer = "https://sso.facile.studio"

func liveRouter(t *testing.T, configure func(*env.Config)) (chi.Router, *gorm.DB) {
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

	appEnv := env.Config{}
	if configure != nil {
		configure(&appEnv)
	}
	router, err := buildRouter(routerDeps{DB: db, SQL: rawSQL(t, db), Env: appEnv, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("buildRouter: %v", err)
	}
	return router, db
}

func rawSQL(t *testing.T, db *gorm.DB) *sql.DB {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	return sqlDB
}

func call(t *testing.T, router chi.Router, method, path string, headers map[string]string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload := bytes.NewReader(nil)
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		payload = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, payload)
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

type signedSession struct {
	Token string
	User  struct {
		ID      int64
		Email   string
		IsAdmin bool
	}
}

// signIn does what a completed Facile sign-in does: the user store resolves the
// account from the claims and porte's session store issues the cookie token.
// No password route exists to do it any other way.
func signIn(t *testing.T, db *gorm.DB, email string) signedSession {
	t.Helper()
	ctx := context.Background()
	subject := "sub-" + email
	userID, err := auth.NewUserStore(db).UpsertFromOIDC(ctx, claims(subject, email, "Someone", true))
	if err != nil {
		t.Fatalf("sign-in of %s: %v", email, err)
	}
	err = db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
		userID, testIssuer, subject).Error
	if err != nil {
		t.Fatalf("link identity: %v", err)
	}
	token, err := porte.NewToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	now := time.Now()
	_, err = portepg.New(rawSQL(t, db)).Sessions().Create(ctx, porte.Session{
		TokenHash: porte.HashToken(token), UserID: userID,
		CreatedAt: now, LastUsedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	var user schemas.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	out := signedSession{Token: token}
	out.User.ID, out.User.Email, out.User.IsAdmin = user.ID, user.Email, user.IsAdmin
	return out
}

func cookieHeader(token string) map[string]string {
	return map[string]string{"Cookie": porte.SessionCookieName + "=" + token}
}

// The browser transport is the one that matters: the cookie authenticates,
// a read succeeds without any extra header, and the same cookie on a mutation
// is refused until X-Facile-CSRF rides along. Bearer requests are exempt by
// design, which is exactly why a bearer-only test would pass either way and
// prove nothing.
func TestCookieMutationRequiresTheCSRFHeaderWhileReadsSucceed(t *testing.T) {
	router, db := liveRouter(t, nil)
	created := signIn(t, db, "someone@facile.studio")

	me := call(t, router, http.MethodGet, "/api/users/me", cookieHeader(created.Token), nil)
	if me.Code != http.StatusOK {
		t.Fatalf("cookie read = %d: %s", me.Code, me.Body)
	}

	refused := call(t, router, http.MethodPatch, "/api/users/me", cookieHeader(created.Token), map[string]any{
		"name": "Renamed",
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("cookie mutation without CSRF = %d, want 403: %s", refused.Code, refused.Body)
	}

	allowed := call(t, router, http.MethodPatch, "/api/users/me", csrfHeaders(created.Token), map[string]any{"name": "Renamed"})
	if allowed.Code != http.StatusOK {
		t.Fatalf("cookie mutation with CSRF = %d: %s", allowed.Code, allowed.Body)
	}
}

func TestLogoutIsIdempotentAndKillsTheSession(t *testing.T) {
	router, db := liveRouter(t, nil)
	created := signIn(t, db, "someone@facile.studio")

	first := call(t, router, http.MethodPost, "/api/auth/logout", csrfHeaders(created.Token), nil)
	if first.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", first.Code, first.Body)
	}
	second := call(t, router, http.MethodPost, "/api/auth/logout", csrfHeaders(created.Token), nil)
	if second.Code != http.StatusOK {
		t.Fatalf("a stale logout must still succeed (v0.2.8 semantics): %d %s", second.Code, second.Body)
	}

	if after := call(t, router, http.MethodGet, "/api/users/me", cookieHeader(created.Token), nil); after.Code != http.StatusUnauthorized {
		t.Fatalf("the session survived the logout: %d", after.Code)
	}
}

func TestTheFirstAccountIsAdminAndTheSecondIsNot(t *testing.T) {
	_, db := liveRouter(t, nil)

	if first := signIn(t, db, "first@facile.studio"); !first.User.IsAdmin {
		t.Fatal("the first account did not become admin")
	}
	if second := signIn(t, db, "second@facile.studio"); second.User.IsAdmin {
		t.Fatal("the second account became admin")
	}
}

func configureIssuer(cfg *env.Config) {
	cfg.Porte.Issuer = fakeIssuer()
	cfg.Porte.ClientID = "sonde"
	cfg.Porte.ClientSecret = "secret"
	cfg.Porte.RedirectURL = "http://localhost:4000/api/auth/oidc/callback"
	cfg.Porte.SuccessURL = "http://localhost:4000/"
}

// Sign-in goes through Registre only. No password route exists to probe for an
// account, with an issuer configured or without one, and /auth/config tells the
// client so.
func TestNoPasswordRouteExistsAndTheConfigSaysSSOOnly(t *testing.T) {
	for name, configure := range map[string]func(*env.Config){"with an issuer": configureIssuer, "without": nil} {
		t.Run(name, func(t *testing.T) {
			router, _ := liveRouter(t, configure)
			for _, path := range []string{"/api/auth/register", "/api/auth/login", "/api/auth/forgot-password"} {
				body := map[string]string{"email": "someone@facile.studio", "password": "a-long-enough-password"}
				code := call(t, router, http.MethodPost, path, nil, body).Code
				if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
					t.Errorf("POST %s = %d, want 404", path, code)
				}
			}
		})
	}

	router, _ := liveRouter(t, configureIssuer)
	config := call(t, router, http.MethodGet, "/api/auth/config", nil, nil)
	if config.Code != http.StatusOK {
		t.Fatalf("/auth/config = %d: %s", config.Code, config.Body)
	}
	var reported struct {
		SSOOnly           bool  `json:"sso_only"`
		OIDCEnabled       bool  `json:"oidc_enabled"`
		AllowRegistration *bool `json:"allow_registration"`
	}
	if err := json.Unmarshal(config.Body.Bytes(), &reported); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if !reported.SSOOnly || !reported.OIDCEnabled || reported.AllowRegistration != nil {
		t.Fatalf("/auth/config lied about the world: %s", config.Body)
	}
}

// fakeIssuer serves the smallest discovery document oidc.New accepts, so the
// boot path runs without reaching for the real provider.
func fakeIssuer() string {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 "http://" + r.Host,
			"authorization_endpoint": "http://" + r.Host + "/auth",
			"token_endpoint":         "http://" + r.Host + "/token",
			"jwks_uri":               "http://" + r.Host + "/keys",
		})
	}))
	return server.URL
}

func claims(subject, email, name string, verified bool) porte.Claims {
	return porte.Claims{
		Provider:      testIssuer,
		Subject:       subject,
		Email:         email,
		Name:          name,
		EmailVerified: verified,
		Locale:        "de",
	}
}

func applySchema(db *gorm.DB) error {
	files, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return migrate.Run(context.Background(), migrate.Config{DB: sqlDB, FS: files, Logger: slog.New(slog.DiscardHandler)})
}
