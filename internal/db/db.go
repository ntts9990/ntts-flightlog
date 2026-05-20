// Package db provides the SQLite data layer for ntts-flightlog v2.
//
// Driver: modernc.org/sqlite (CGo-free). Per plan Principle 2, no CGo fallback
// is permitted; if cold-start benchmarks fail the budget, the plan halts for
// re-scoping rather than pivoting to mattn/go-sqlite3.
//
// Every connection open applies three PRAGMAs (plan A2, locked at iter 2):
//
//	PRAGMA journal_mode=WAL       — enables concurrent readers + single writer
//	PRAGMA busy_timeout=5000      — wait up to 5 s before SQLITE_BUSY
//	PRAGMA synchronous=NORMAL     — good durability/perf balance with WAL
//
// A single database/sql connection is used per process (MaxOpenConns=1).
// If profiling later reveals contention, escalate to a pool of max 3 — but not
// arbitrarily (per plan note).
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // registers "sqlite" driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps *sql.DB with ntts-flightlog connection settings pre-applied.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at the given path, applies the
// required PRAGMAs, and runs any pending schema migrations. The returned *DB
// is ready for use.
//
// Callers must call Close() when done.
func Open(dbPath string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("db.Open: sql.Open(%q): %w", dbPath, err)
	}

	// Single connection per process (plan A2 spec).
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// Apply required PRAGMAs immediately after open.
	if err := applyPRAGMAs(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db.Open: applyPRAGMAs: %w", err)
	}

	d := &DB{sqlDB}

	// Run any pending migrations.
	if err := d.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db.Open: migrate: %w", err)
	}

	return d, nil
}

// applyPRAGMAs executes the three required PRAGMAs on the connection pool.
// With MaxOpenConns=1 this covers the single shared connection.
func applyPRAGMAs(db *sql.DB) error {
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("exec %q: %w", pragma, err)
		}
	}
	return nil
}

// migrate bootstraps the _schema_migrations tracking table and applies all
// embedded *.sql files under migrations/ in lexicographic order, skipping any
// already recorded. This is the minimal home-grown migration runner (plan A2);
// no golang-migrate dependency is introduced.
func (d *DB) migrate() error {
	// Bootstrap the tracking table before querying it.
	const createMeta = `CREATE TABLE IF NOT EXISTS _schema_migrations (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		filename   TEXT    NOT NULL UNIQUE,
		applied_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`
	if _, err := d.Exec(createMeta); err != nil {
		return fmt.Errorf("create _schema_migrations: %w", err)
	}

	// Enumerate migration files (embed.FS always uses forward slashes).
	dirEntries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range dirEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var n int
		if err := d.QueryRow(
			"SELECT COUNT(*) FROM _schema_migrations WHERE filename = ?", name,
		).Scan(&n); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if n > 0 {
			continue // already applied
		}

		data, err := migrationsFS.ReadFile(path.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := d.Exec(string(data)); err != nil {
			return fmt.Errorf("exec migration %s: %w", name, err)
		}

		if _, err := d.Exec(
			"INSERT INTO _schema_migrations (filename) VALUES (?)", name,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}

	return nil
}

// Version returns the SQLite library version string (e.g. "3.47.0").
// Used in BenchmarkColdOpen to verify the connection is live.
func (d *DB) Version() (string, error) {
	var ver string
	if err := d.QueryRow("SELECT sqlite_version()").Scan(&ver); err != nil {
		return "", fmt.Errorf("sqlite_version: %w", err)
	}
	return ver, nil
}
