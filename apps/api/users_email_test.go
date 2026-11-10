package main

import (
	"net/http"
	"testing"
)

func TestNobodyCanChangeTheirEmail(t *testing.T) {
	router, db := fleetRouter(t)
	user := signIn(t, db, "someone@facile.studio")

	refused := call(t, router, http.MethodPatch, "/api/users/me", csrfHeaders(user.Token), map[string]any{"email": "other@facile.studio"})
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("email change = %d, want 400", refused.Code)
	}
	var email string
	db.Table("users").Select("email").Where("id = ?", user.User.ID).Scan(&email)
	if email != "someone@facile.studio" {
		t.Fatalf("email is now %q", email)
	}
	renamed := call(t, router, http.MethodPatch, "/api/users/me", csrfHeaders(user.Token), map[string]any{"name": "Renamed"})
	if renamed.Code != http.StatusOK {
		t.Fatalf("name change = %d: %s", renamed.Code, renamed.Body)
	}
}

func TestAnAdminCannotClaimAnAllowListedEmail(t *testing.T) {
	router, db := fleetRouter(t)
	stranger := signIn(t, db, "stranger@facile.studio")
	if !stranger.User.IsAdmin {
		t.Fatal("setup: the first account should be admin")
	}

	call(t, router, http.MethodPatch, "/api/users/me", csrfHeaders(stranger.Token), map[string]any{"email": "admin@facile.studio"})

	if got := fleetGet(t, router, "/api/fleet/actions", stranger.Token).Code; got != http.StatusForbidden {
		t.Fatalf("admin after claiming an allow-listed email = %d, want 403", got)
	}
}
