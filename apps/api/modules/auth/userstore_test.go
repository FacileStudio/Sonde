package auth

import (
	"context"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/migrations"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const (
	testBootstrap = "sh scripts/postgres-up.sh && export TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5603/sonde_test?sslmode=disable"
	testIssuer    = "https://sso.test"
)

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}
	config := testdb.Config{Prefix: "sonde_userstore_test", Migrate: func(db *gorm.DB) error {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return migrate.Run(context.Background(), migrate.Config{DB: sqlDB, FS: migrations.FS})
	}}
	db, err := testdb.Open(url, config)
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if err := testdb.Truncate(db, config); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	return db
}

func claims(subject, email string, verified bool) porte.Claims {
	return porte.Claims{Provider: testIssuer, Subject: subject, Email: email, EmailVerified: verified, Locale: "de"}
}

func link(t *testing.T, db *gorm.DB, userID int64, subject string) {
	t.Helper()
	err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`, userID, testIssuer, subject).Error
	if err != nil {
		t.Fatalf("link identity: %v", err)
	}
}

func seed(t *testing.T, db *gorm.DB, email, name string) schemas.User {
	t.Helper()
	user := schemas.User{Email: email, Name: name, PasswordHash: "legacy-hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return user
}

// An account created before Registre owned sign-in has no federated identity.
// Its owner's first Facile sign-in must land on that same row, keep its data,
// and then resolve by subject.
func TestFirstFacileSignInLinksAnExistingAccountByVerifiedEmail(t *testing.T) {
	db := openTestDatabase(t)
	store := NewUserStore(db)
	ctx := context.Background()
	existing := seed(t, db, "camille@facile.studio", "Camille")

	id, err := store.UpsertFromOIDC(ctx, claims("sub-1", "Camille@Facile.Studio", true))
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	if id != existing.ID {
		t.Fatalf("first sign-in created account %d, want the existing %d", id, existing.ID)
	}
	link(t, db, id, "sub-1")

	again, err := store.UpsertFromOIDC(ctx, claims("sub-1", "camille@facile.studio", true))
	if err != nil || again != existing.ID {
		t.Fatalf("second sign-in = %d, %v; want %d", again, err, existing.ID)
	}

	var user schemas.User
	if err := db.First(&user, id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if user.Locale != "de" {
		t.Errorf("locale = %q, want the sign-in's de", user.Locale)
	}
	if user.PasswordHash != "legacy-hash" {
		t.Error("linking rewrote the account's stored data")
	}
}

func TestAnUnverifiedAddressNeverLinksNorRewritesAnAddress(t *testing.T) {
	db := openTestDatabase(t)
	store := NewUserStore(db)
	ctx := context.Background()
	noah := seed(t, db, "noah@facile.studio", "Noah")

	if _, err := store.UpsertFromOIDC(ctx, claims("sub-2", "noah@facile.studio", false)); err == nil {
		t.Fatal("an unverified address linked to an existing account")
	}

	owner, err := store.UpsertFromOIDC(ctx, claims("sub-owner", "mia@facile.studio", true))
	if err != nil {
		t.Fatalf("owner sign-in: %v", err)
	}
	link(t, db, owner, "sub-owner")
	if _, err := store.UpsertFromOIDC(ctx, claims("sub-owner", noah.Email, false)); err != nil {
		t.Fatalf("unverified refresh: %v", err)
	}
	var stored schemas.User
	if err := db.First(&stored, owner).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Email != "mia@facile.studio" {
		t.Errorf("an unverified claim moved the stored address to %q", stored.Email)
	}
}

func TestARecycledAddressIsNotHandedToANewSubject(t *testing.T) {
	db := openTestDatabase(t)
	store := NewUserStore(db)
	owner := seed(t, db, "lea@facile.studio", "Lea")
	link(t, db, owner.ID, "sub-owner")

	if _, err := store.UpsertFromOIDC(context.Background(), claims("sub-newcomer", "lea@facile.studio", true)); err == nil {
		t.Fatal("a second subject of the same issuer took over a linked account")
	}
}

func countRows(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func seedOwnedRows(t *testing.T, db *gorm.DB, owner int64, slug string) {
	t.Helper()
	err := db.Exec(`INSERT INTO monitors (owner_id, slug, name, type, target) VALUES (?, ?, 'm', 'http', 'https://x.example')`,
		owner, slug).Error
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}
	if err := db.Exec(`INSERT INTO webhooks (owner_id, name, url) VALUES (?, 'w', 'https://x.example/hook')`, owner).Error; err != nil {
		t.Fatalf("webhook: %v", err)
	}
}

// Registre erases a person and tells every app. The person's account and
// everything it owns go, other people's data stays, and a second delivery of
// the same notice is a no-op.
func TestEraseAccountDeletesThePersonsDataOnceAndIsIdempotent(t *testing.T) {
	db := openTestDatabase(t)
	store := NewUserStore(db)
	ctx := context.Background()

	gone := seed(t, db, "gone@facile.studio", "Gone")
	stays := seed(t, db, "stays@facile.studio", "Stays")
	link(t, db, gone.ID, "sub-gone")
	link(t, db, stays.ID, "sub-stays")
	seedOwnedRows(t, db, gone.ID, "mon-gone")
	seedOwnedRows(t, db, stays.ID, "mon-stays")
	if err := db.Exec(`INSERT INTO status_pages (slug, title) VALUES ('page', 'Page')`).Error; err != nil {
		t.Fatalf("status page: %v", err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		if err := store.EraseAccount(ctx, testIssuer, "sub-gone"); err != nil {
			t.Fatalf("erase attempt %d: %v", attempt, err)
		}
	}

	checks := []struct {
		name  string
		query string
		args  []any
		want  int64
	}{
		{"the erased account", `SELECT count(*) FROM users WHERE id = ?`, []any{gone.ID}, 0},
		{"its monitors", `SELECT count(*) FROM monitors WHERE owner_id = ?`, []any{gone.ID}, 0},
		{"its webhooks", `SELECT count(*) FROM webhooks WHERE owner_id = ?`, []any{gone.ID}, 0},
		{"its identity", `SELECT count(*) FROM porte_identities WHERE user_id = ?`, []any{gone.ID}, 0},
		{"the other account", `SELECT count(*) FROM users WHERE id = ?`, []any{stays.ID}, 1},
		{"the other monitors", `SELECT count(*) FROM monitors WHERE owner_id = ?`, []any{stays.ID}, 1},
		{"the other webhooks", `SELECT count(*) FROM webhooks WHERE owner_id = ?`, []any{stays.ID}, 1},
		{"the status pages", `SELECT count(*) FROM status_pages`, nil, 1},
	}
	for _, check := range checks {
		if got := countRows(t, db, check.query, check.args...); got != check.want {
			t.Errorf("%s: %d rows, want %d", check.name, got, check.want)
		}
	}

	if err := store.EraseAccount(ctx, testIssuer, "never-existed"); err != nil {
		t.Errorf("an unknown subject must be a no-op, got %v", err)
	}
}

func TestErasingTheOnlyAdministratorHandsTheRoleToTheOldestAccount(t *testing.T) {
	db := openTestDatabase(t)
	store := NewUserStore(db)
	ctx := context.Background()

	admin := seed(t, db, "admin@facile.studio", "Admin")
	older := seed(t, db, "older@facile.studio", "Older")
	newer := seed(t, db, "newer@facile.studio", "Newer")
	if err := db.Exec(`UPDATE users SET is_admin = true WHERE id = ?`, admin.ID).Error; err != nil {
		t.Fatalf("promote: %v", err)
	}
	link(t, db, admin.ID, "sub-admin")
	link(t, db, older.ID, "sub-older")

	if err := store.EraseAccount(ctx, testIssuer, "sub-admin"); err != nil {
		t.Fatalf("erase: %v", err)
	}
	if got := countRows(t, db, `SELECT count(*) FROM users WHERE is_admin AND id = ?`, older.ID); got != 1 {
		t.Errorf("the oldest remaining account is not administrator")
	}
	if got := countRows(t, db, `SELECT count(*) FROM users WHERE is_admin AND id = ?`, newer.ID); got != 0 {
		t.Errorf("a newer account was promoted too")
	}

	link(t, db, newer.ID, "sub-newer")
	if err := store.EraseAccount(ctx, testIssuer, "sub-newer"); err != nil {
		t.Fatalf("erase non-admin: %v", err)
	}
	if got := countRows(t, db, `SELECT count(*) FROM users WHERE is_admin`, nil...); got != 1 {
		t.Errorf("administrators = %d, want 1 after erasing a non-admin", got)
	}
}

func TestDeletionStateRoundTrips(t *testing.T) {
	db := openTestDatabase(t)
	state := NewDeletionState(db)
	ctx := context.Background()

	if loaded, err := state.Load(ctx); err != nil || loaded != nil {
		t.Fatalf("empty Load = %q, %v; want nil, nil", loaded, err)
	}
	for _, blob := range []string{`{"cursor":1}`, `{"cursor":2}`} {
		if err := state.Save(ctx, []byte(blob)); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	loaded, err := state.Load(ctx)
	if err != nil || string(loaded) != `{"cursor":2}` {
		t.Fatalf("Load = %q, %v; want the last save", loaded, err)
	}
}
