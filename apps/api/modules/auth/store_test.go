package auth

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/FacileStudio/Sonde/apps/api/migrations"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
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

func sqlOf(t *testing.T, db *gorm.DB) *sql.DB {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	return sqlDB
}

func newStore(t *testing.T, db *gorm.DB) *UserStore {
	t.Helper()
	store, err := NewUserStore(context.Background(), sqlOf(t, db))
	if err != nil {
		t.Fatalf("user store: %v", err)
	}
	return store
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

// Every users column has a default or is one the kit writes, so an account the
// kit creates holds no NULL and NewStore reports no column as unwritten.
func TestTheUserStoreLeavesNoColumnUnwritten(t *testing.T) {
	if got := newStore(t, openTestDatabase(t)).Unwritten(); len(got) != 0 {
		t.Fatalf("Unwritten() = %v, want none", got)
	}
}

// NewUserStore reads the users table when it is built, through kit.NewStore. A
// store built as a literal checks nothing, so it would come up over a table
// without the admin flag and fail on the first sign-in instead of at boot.
func TestTheUserStoreReadsTheUsersTableWhenBuilt(t *testing.T) {
	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}
	db, err := testdb.Open(url, testdb.Config{Prefix: "sonde_userstore_bare", Migrate: func(db *gorm.DB) error {
		if err := db.Exec(`DROP TABLE IF EXISTS users`).Error; err != nil {
			return err
		}
		return db.Exec(`CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '', avatar_url TEXT NOT NULL DEFAULT '', locale TEXT NOT NULL DEFAULT 'en')`).Error
	}})
	if err != nil {
		t.Fatalf("test database: %v", err)
	}
	if _, err := NewUserStore(context.Background(), sqlOf(t, db)); err == nil || !strings.Contains(err.Error(), "is_admin") {
		t.Fatalf("NewUserStore over a users table without is_admin: err = %v, want one naming the column", err)
	}
}

// Registre erases a person and tells every app. The person's account and
// everything it owns go, other people's data stays, and a second delivery of
// the same notice is a no-op.
func TestEraseAccountDeletesThePersonsDataOnceAndIsIdempotent(t *testing.T) {
	db := openTestDatabase(t)
	store := newStore(t, db)
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
	store := newStore(t, db)
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
