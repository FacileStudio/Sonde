package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	auth "github.com/FacileStudio/Sonde/apps/api/modules/auth"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/testdb"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func liveRouter(t *testing.T, configure func(*env.Config)) chi.Router {
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

	appEnv := env.Config{AllowRegistration: true}
	if configure != nil {
		configure(&appEnv)
	}
	router, err := buildRouter(routerDeps{DB: db, SQL: rawSQL(t, db), Env: appEnv, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("buildRouter: %v", err)
	}
	return router
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
	Token string `json:"token"`
	User  struct {
		ID      int64  `json:"id"`
		Email   string `json:"email"`
		IsAdmin bool   `json:"is_admin"`
	} `json:"user"`
}

func registerAndSignIn(t *testing.T, router chi.Router, email string) signedSession {
	t.Helper()
	registered := call(t, router, http.MethodPost, "/api/auth/register", nil, map[string]string{
		"email": email, "name": "Someone", "password": "a-long-enough-password",
	})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register = %d: %s", registered.Code, registered.Body)
	}
	var created signedSession
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	return created
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
	router := liveRouter(t, nil)
	created := registerAndSignIn(t, router, "someone@facile.studio")

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

	allowed := call(t, router, http.MethodPatch, "/api/users/me",
		map[string]string{
			"Cookie":        porte.SessionCookieName + "=" + created.Token,
			"X-Facile-CSRF": "1",
		},
		map[string]any{"name": "Renamed"})
	if allowed.Code != http.StatusOK {
		t.Fatalf("cookie mutation with CSRF = %d: %s", allowed.Code, allowed.Body)
	}
}

func TestLogoutIsIdempotentAndKillsTheSession(t *testing.T) {
	router := liveRouter(t, nil)
	created := registerAndSignIn(t, router, "someone@facile.studio")

	headers := func(csrf bool) map[string]string {
		out := cookieHeader(created.Token)
		if csrf {
			out["X-Facile-CSRF"] = "1"
		}
		return out
	}

	first := call(t, router, http.MethodPost, "/api/auth/logout", headers(true), nil)
	if first.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", first.Code, first.Body)
	}
	second := call(t, router, http.MethodPost, "/api/auth/logout", headers(true), nil)
	if second.Code != http.StatusOK {
		t.Fatalf("a stale logout must still succeed (v0.2.8 semantics): %d %s", second.Code, second.Body)
	}

	if after := call(t, router, http.MethodGet, "/api/users/me", cookieHeader(created.Token), nil); after.Code != http.StatusUnauthorized {
		t.Fatalf("the session survived the logout: %d", after.Code)
	}
}

func TestTheFirstAccountIsAdminAndTheSecondIsNot(t *testing.T) {
	router := liveRouter(t, nil)

	first := registerAndSignIn(t, router, "first@facile.studio")
	if !first.User.IsAdmin {
		t.Fatal("the first account did not become admin")
	}

	registered := call(t, router, http.MethodPost, "/api/auth/register", nil, map[string]string{
		"email": "second@facile.studio", "name": "Someone", "password": "a-long-enough-password",
	})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register second = %d: %s", registered.Code, registered.Body)
	}
	var created signedSession
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if created.User.IsAdmin {
		t.Fatal("the second account became admin")
	}
}

// SSO_ONLY=true removes the credential routes instead of rejecting them — there
// must be no endpoint left to probe for an account — while /auth/config keeps
// telling the client what the world looks like.
func TestSSOOnlyRemovesThePasswordRoutesAndReportsItself(t *testing.T) {
	router := liveRouter(t, func(cfg *env.Config) {
		cfg.Porte.SSOOnly = true
		cfg.Porte.Issuer = fakeIssuer()
		cfg.Porte.ClientID = "sonde"
		cfg.Porte.ClientSecret = "secret"
		cfg.Porte.RedirectURL = "https://sonde.facile.studio/api/auth/oidc/callback"
		cfg.Porte.SuccessURL = "https://sonde.facile.studio/"
	})

	if code := call(t, router, http.MethodPost, "/api/auth/login", nil, map[string]string{
		"email": "someone@facile.studio", "password": "a-long-enough-password",
	}).Code; code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		t.Fatalf("login survived SSO_ONLY: %d", code)
	}

	config := call(t, router, http.MethodGet, "/api/auth/config", nil, nil)
	if config.Code != http.StatusOK {
		t.Fatalf("/auth/config = %d: %s", config.Code, config.Body)
	}
	var reported struct {
		SSOOnly     bool `json:"sso_only"`
		OIDCEnabled bool `json:"oidc_enabled"`
	}
	if err := json.Unmarshal(config.Body.Bytes(), &reported); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if !reported.SSOOnly || !reported.OIDCEnabled {
		t.Fatalf("/auth/config lied about the world: %+v", reported)
	}
}

// Without an issuer the kit serves /auth/config with oidc_enabled=false — that
// is what tells the login page to draw the password form.
func TestAuthConfigWithoutAnIssuerReportsPasswordsOnly(t *testing.T) {
	router := liveRouter(t, nil)

	config := call(t, router, http.MethodGet, "/api/auth/config", nil, nil)
	var reported struct {
		SSOOnly           bool `json:"sso_only"`
		OIDCEnabled       bool `json:"oidc_enabled"`
		AllowRegistration bool `json:"allow_registration"`
	}
	if err := json.Unmarshal(config.Body.Bytes(), &reported); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if reported.OIDCEnabled || reported.SSOOnly || !reported.AllowRegistration {
		t.Fatalf("/auth/config lied about the world: %+v", reported)
	}
}

// The OIDC half of the upsert: first sign-in becomes admin, a second sign-in
// finds the same row by (provider, subject), and an address the provider does
// not vouch for may never adopt an account that already exists. Registre
// reports email_verified truthfully, so the refusal rarely fires against the
// suite provider, but it is what stands between a callback and every
// pre-existing address on a provider that stays quiet. Authentik hardcoded
// email_verified=false, which is the history this rule was written against.
func TestUpsertFromOIDCOwnsTheAccountRules(t *testing.T) {
	url, ok := testdb.URL()
	if !ok {
		testdb.Announce("sh scripts/postgres-up.sh && export " + testdb.EnvVar + "=postgres://postgres:postgres@127.0.0.1:5603/sonde_test?sslmode=disable")
		t.Skip(testdb.SkipReason("sh scripts/postgres-up.sh"))
	}
	db, err := testdb.Open(url, testdb.Config{Prefix: "sonde_auth_test", Migrate: applySchema})
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}

	users := auth.NewUserStore(db)
	ctx := context.Background()

	userID, err := users.UpsertFromOIDC(ctx, claims("abc", "someone@facile.studio", "Someone", true))
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	var user schemas.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if !user.IsAdmin {
		t.Fatal("the first account to sign in did not become admin")
	}
	if user.PasswordHash != "" {
		t.Fatal("an SSO account was given a password hash")
	}

	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, "https://sso.facile.studio", "abc").Error; err != nil {
		t.Fatalf("link the identity: %v", err)
	}

	again, err := users.UpsertFromOIDC(ctx, claims("abc", "renamed@facile.studio", "Renamed", true))
	if err != nil {
		t.Fatalf("second sign-in: %v", err)
	}
	if again != userID {
		t.Fatalf("a renamed user got a second account: %d then %d", userID, again)
	}

	if _, err := users.UpsertFromOIDC(ctx, claims("other", "renamed@facile.studio", "Someone Else", false)); err == nil {
		t.Fatal("an unverified address adopted an existing account")
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
		Provider:      "https://sso.facile.studio",
		Subject:       subject,
		Email:         email,
		Name:          name,
		EmailVerified: verified,
	}
}

func applySchema(db *gorm.DB) error {
	for _, file := range []string{"migrations/00001_sonde.sql", "migrations/00002_porte.sql", "migrations/00004_user_avatar.sql"} {
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		up := strings.Split(string(raw), "-- +goose Down")[0]
		up = strings.ReplaceAll(up, "CREATE TABLE ", "CREATE TABLE IF NOT EXISTS ")
		up = strings.ReplaceAll(up, "CREATE INDEX ", "CREATE INDEX IF NOT EXISTS ")
		for _, statement := range strings.Split(up, ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if err := db.Exec(statement).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
