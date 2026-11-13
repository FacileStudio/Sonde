package main

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/env"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/oidc"
	jwtpkg "github.com/FacileStudio/porte/oidc/jwt"
	"gorm.io/gorm"
)

// fakeRegistre serves the discovery document oidc.New needs and a deletions
// list naming every erased subject on every call, as a re-polled overlap does.
func fakeRegistre(t *testing.T, erased ...string) string {
	t.Helper()
	deletedAt := time.Now().Add(-time.Hour).UTC()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/directory/deletions" {
			list := []directory.Deleted{}
			for _, subject := range erased {
				list = append(list, directory.Deleted{Subject: subject, DeletedAt: deletedAt})
			}
			json.NewEncoder(w).Encode(map[string]any{"deleted": list})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 "http://" + r.Host,
			"authorization_endpoint": "http://" + r.Host + "/auth",
			"token_endpoint":         "http://" + r.Host + "/token",
			"jwks_uri":               "http://" + r.Host + "/keys",
		})
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func createAccount(t *testing.T, db *gorm.DB, issuer, subject string) {
	t.Helper()
	ctx := context.Background()
	identity := claims(subject, subject+"@facile.studio", "Someone", true)
	identity.Provider = issuer
	userID, err := userStore(t, db).UpsertFromOIDC(ctx, identity)
	if err != nil {
		t.Fatalf("create %s: %v", subject, err)
	}
	err = db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`, userID, issuer, subject).Error
	if err != nil {
		t.Fatalf("link %s: %v", subject, err)
	}
}

func accounts(t *testing.T, db *gorm.DB, subject string) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE subject = ?`, subject).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", subject, err)
	}
	return count
}

// An account Registre erased stays erased: the sign-in refuses to create it
// again, and one that comes back anyway (a race past the guard) is erased
// again by the next poll, with no restart and nobody else touched.
func TestAnErasedSubjectIsRefusedAndErasedAgainWithoutRestart(t *testing.T) {
	db := openAuthDatabase(t)
	issuer := fakeRegistre(t, "sub-gone")
	appEnv := env.Config{}
	configureIssuer(&appEnv)
	appEnv.Porte.Issuer = issuer
	logger := slog.New(slog.DiscardHandler)
	users := userStore(t, db)
	deletions := newDeletions(db, rawSQL(t, db), users, appEnv, logger)
	id, err := buildIdentity(routerDeps{DB: db, SQL: rawSQL(t, db), Users: users, Deletions: deletions, Env: appEnv, Logger: logger})
	if err != nil {
		t.Fatalf("buildIdentity: %v", err)
	}
	ctx := context.Background()
	createAccount(t, db, issuer, "sub-gone")
	createAccount(t, db, issuer, "sub-stays")
	before := userCount(t, db)

	if err := deletions.Poll(ctx); err != nil {
		t.Fatalf("first poll: %v", err)
	}
	if got := accounts(t, db, "sub-gone"); got != 0 {
		t.Fatalf("erased subject still holds %d identities", got)
	}

	verified := true
	_, err = id.kit.ProvisionBearer(ctx, jwtpkg.Claims{
		Subject: "sub-gone", Email: "sub-gone@facile.studio", Name: "Someone",
		Profile: jwtpkg.Profile{EmailVerified: &verified},
	})
	if !stderrors.Is(err, porte.ErrSubjectErased) || !stderrors.Is(err, oidc.ErrProvisionRefused) {
		t.Fatalf("provisioning an erased subject = %v, want ErrSubjectErased and ErrProvisionRefused", err)
	}
	if got := accounts(t, db, "sub-gone"); got != 0 {
		t.Fatalf("the refused provisioning left %d identities", got)
	}

	createAccount(t, db, issuer, "sub-gone")
	if err := deletions.Poll(ctx); err != nil {
		t.Fatalf("second poll: %v", err)
	}
	if got := accounts(t, db, "sub-gone"); got != 0 {
		t.Fatalf("re-created account survived the next poll with %d identities", got)
	}
	if got := accounts(t, db, "sub-stays"); got != 1 {
		t.Fatalf("bystander holds %d identities, want 1", got)
	}
	if after := userCount(t, db); after != before-1 {
		t.Fatalf("users went from %d to %d, want exactly one fewer", before, after)
	}
}

func userCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Raw(`SELECT count(*) FROM users`).Scan(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	return count
}

// Without an issuer there is no receiver, and the sign-in gets no guard
// rather than an interface around a nil pointer that would panic.
func TestNoReceiverGivesNoGuard(t *testing.T) {
	if guard := erasedGuard(nil); guard != nil {
		t.Fatalf("erasedGuard(nil) = %#v, want nil", guard)
	}
}
