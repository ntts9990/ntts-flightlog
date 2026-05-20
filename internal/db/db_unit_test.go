package db_test

// db_unit_test.go: unit tests for db.Version(), Open() idempotency, and
// migration tracking. Covers statements that BenchmarkColdOpen also covers
// but which don't count toward -cover without -bench flag (e.g. Version()).

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// TestVersion verifies Version() returns a non-empty SQLite semver string.
// BenchmarkColdOpen already calls Version() but bench results don't count
// toward -cover without -bench; this regular test closes that gap.
func TestVersion(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	ver, err := d.Version()
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if ver == "" {
		t.Error("Version returned empty string")
	}
	if !strings.Contains(ver, ".") {
		t.Errorf("Version %q looks wrong; expected semver like 3.x.y", ver)
	}
	t.Logf("SQLite version: %s", ver)
}

// TestMigrateIdempotent opens the same file-based DB twice and verifies the
// second Open succeeds. The migration runner must exercise the
// "already applied → continue" branch for each of the 3 migration files.
func TestMigrateIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idempotent.db")

	// First open: applies all migrations.
	d1, err := db.Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := d1.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	// Second open: all migrations are already recorded → each takes continue path.
	d2, err := db.Open(path)
	if err != nil {
		t.Fatalf("second Open (idempotent): %v", err)
	}
	defer func() { _ = d2.Close() }()

	ver, err := d2.Version()
	if err != nil {
		t.Fatalf("Version after second Open: %v", err)
	}
	if ver == "" {
		t.Error("Version empty after second Open")
	}

	// Schema must be intact: exactly 7 migrations recorded.
	var count int
	if err := d2.QueryRow("SELECT COUNT(*) FROM _schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query _schema_migrations: %v", err)
	}
	if count != 7 {
		t.Errorf("_schema_migrations count: got %d, want 7", count)
	}
}

// TestOpenAndClose verifies a round-trip Open+Close on a fresh file DB.
func TestOpenAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openclose.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestOpenInMemoryMultiple verifies multiple independent in-memory DBs can be
// opened and each receives the full migration suite.
func TestOpenInMemoryMultiple(t *testing.T) {
	t.Parallel()
	for i := range 5 {
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatalf("[%d] Open: %v", i, err)
		}
		var cnt int
		if err := d.QueryRow("SELECT COUNT(*) FROM _schema_migrations").Scan(&cnt); err != nil {
			t.Errorf("[%d] query _schema_migrations: %v", i, err)
		}
		if cnt != 7 {
			t.Errorf("[%d] migration count = %d, want 7", i, cnt)
		}
		_ = d.Close()
	}
}

// TestOpenBadPath verifies Open returns an error when the parent directory of
// the DB file does not exist. This covers the applyPRAGMAs error branch inside
// Open (the path where applyPRAGMAs fails and the sql.DB is closed before
// returning the error).
func TestOpenBadPath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "nonexistent_subdir", "test.db")
	_, err := db.Open(bad)
	if err == nil {
		t.Fatal("Open with nonexistent parent dir: expected error, got nil")
	}
}

// TestVersionOnClosedDB verifies Version() returns an error when the underlying
// sql.DB has been closed, covering the error-return branch.
func TestVersionOnClosedDB(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_ = d.Close()

	_, err = d.Version()
	if err == nil {
		t.Error("Version on closed DB: expected error, got nil")
	}
}

// TestMigrateTracksFilenames verifies the migration runner records each SQL
// filename in _schema_migrations after the first Open.
func TestMigrateTracksFilenames(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	rows, err := d.Query("SELECT filename FROM _schema_migrations ORDER BY filename")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			t.Fatalf("scan: %v", err)
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	want := []string{"0001_init.sql", "0002_turn_anchors.sql", "0003_metric_views.sql", "0004_blocker_resolution.sql", "0005_live_blocker_accumulation.sql", "0006_decision_status.sql", "0007_turn_outcome.sql"}
	if len(files) != len(want) {
		t.Fatalf("migration files: got %v, want %v", files, want)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("migration[%d]: got %q, want %q", i, f, want[i])
		}
	}
}
