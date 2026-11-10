package events

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/migrations"
	antenne "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/testdb"
	"gorm.io/gorm"
)

const testBootstrap = "sh scripts/postgres-up.sh && export TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5603/sonde_test?sslmode=disable"

type fakeEmitter struct {
	mu        sync.Mutex
	dialErr   error
	emitErr   error
	dials     int
	connected bool
	channels  []string
}

func (e *fakeEmitter) Connect(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dials++
	e.connected = e.dialErr == nil
	return e.dialErr
}

func (e *fakeEmitter) EmitNow(channel string, _ any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.emitErr != nil {
		return e.emitErr
	}
	e.channels = append(e.channels, channel)
	return nil
}

func (e *fakeEmitter) IsConnected() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.connected
}

func (e *fakeEmitter) Disconnect() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connected = false
}

func (e *fakeEmitter) sent() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.channels...)
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}
	config := testdb.Config{Prefix: "sonde_events_test", Migrate: func(db *gorm.DB) error {
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
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	return sqlDB
}

func testPublisher(t *testing.T, emitter *fakeEmitter) *Publisher {
	t.Helper()
	config := &antenne.Config{Instance: "http://bus.invalid", App: DefaultApp, Secret: "shared"}
	return newPublisher(config, emitter, testDB(t), slog.New(slog.DiscardHandler))
}

func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 5s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
