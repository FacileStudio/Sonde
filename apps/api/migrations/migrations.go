// Package migrations embeds the ordered SQL that owns the schema.
//
// The schema ships as hand-written Postgres DDL applied by tronc/migrate at
// boot, not as GORM AutoMigrate: what runs in production must be exactly what
// the tests ran.
package migrations

import "embed"

// FS holds the migration files at its root, which is the layout tronc/migrate
// expects.
//
//go:embed *.sql
var FS embed.FS
