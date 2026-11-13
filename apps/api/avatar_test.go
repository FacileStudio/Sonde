package main

import (
	"context"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

// TestThePictureClaimIsStoredOnlyWhenHTTPS pins what a sign-in writes: the
// https URL Registre sends, and nothing for a placeholder, so a browser is
// never pointed at a data: URL and the client draws the initial instead.
func TestThePictureClaimIsStoredOnlyWhenHTTPS(t *testing.T) {
	db := avatarDB(t)
	users := userStore(t, db)
	ctx := context.Background()

	signedIn := claims("pic", "pic@facile.studio", "Pic", true)
	signedIn.Picture = "https://sso.facile.studio/avatars/pic"
	picID, err := users.UpsertFromOIDC(ctx, signedIn)
	if err != nil {
		t.Fatalf("sign-in: %v", err)
	}
	if got := avatarOf(t, db, picID); got != signedIn.Picture {
		t.Fatalf("sign-in stored %q, want %q", got, signedIn.Picture)
	}
	linkIdentity(t, db, picID, "pic")

	signedIn.Picture = "data:image/svg+xml;base64,AAAA"
	if _, err := users.UpsertFromOIDC(ctx, signedIn); err != nil {
		t.Fatalf("second sign-in: %v", err)
	}
	if got := avatarOf(t, db, picID); got != "" {
		t.Fatalf("a non-https claim was stored: %q", got)
	}
}

// TestBackfillFillsOnlyEmptyAvatars pins the boot backfill for accounts that
// signed in before Sonde stored the claim: it derives Registre's stable URL
// from the identity row, never overwrites a URL a sign-in wrote, and a second
// run changes nothing.
func TestBackfillFillsOnlyEmptyAvatars(t *testing.T) {
	db := avatarDB(t)
	users := userStore(t, db)
	ctx := context.Background()

	legacy := schemas.User{Email: "legacy@facile.studio", Name: "Legacy"}
	kept := schemas.User{Email: "kept@facile.studio", Name: "Kept"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("legacy account: %v", err)
	}
	if err := db.Create(&kept).Error; err != nil {
		t.Fatalf("kept account: %v", err)
	}
	legacyID, keptID := legacy.ID, kept.ID
	linkIdentity(t, db, legacyID, "legacy-sub")
	linkIdentity(t, db, keptID, "kept-sub")
	if err := db.Exec(`UPDATE users SET avatar_url = 'https://elsewhere.example/kept' WHERE id = ?`, keptID).Error; err != nil {
		t.Fatalf("mark: %v", err)
	}

	filled, err := users.BackfillPictures(ctx, "https://sso.facile.studio")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if filled != 1 || avatarOf(t, db, legacyID) != "https://sso.facile.studio/avatars/legacy-sub" {
		t.Fatalf("backfill filled %d rows, legacy avatar %q", filled, avatarOf(t, db, legacyID))
	}
	if got := avatarOf(t, db, keptID); got != "https://elsewhere.example/kept" {
		t.Fatalf("backfill overwrote a stored URL: %q", got)
	}
	if again, _ := users.BackfillPictures(ctx, "https://sso.facile.studio"); again != 0 {
		t.Fatalf("a second backfill filled %d rows", again)
	}
}

// avatarDB opens an emptied test database, or skips without one.
func avatarDB(t *testing.T) *gorm.DB {
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
	if err := testdb.Truncate(db, testdb.Config{Prefix: "sonde_auth_test"}); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// linkIdentity records the Registre identity porte would have stored.
func linkIdentity(t *testing.T, db *gorm.DB, userID int64, subject string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, "https://sso.facile.studio", subject).Error; err != nil {
		t.Fatalf("link the identity: %v", err)
	}
}

// avatarOf reads the stored picture URL of one account.
func avatarOf(t *testing.T, db *gorm.DB, id int64) string {
	t.Helper()
	var user schemas.User
	if err := db.First(&user, id).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	return user.AvatarURL
}
