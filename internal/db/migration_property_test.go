// Package db_test — property/contract tests for SQLite schema migration invariants.
//
// Three migration files are covered:
//
//	0001_init.sql      — forward schema check + idempotent re-apply
//	0002_turn_anchors  — forward+backward+forward round-trip on turns table
//	0003_metric_views  — forward+backward+forward round-trip on metric views
//
// Each round-trip property is exercised with 100 randomly generated row counts
// (seeded for determinism), satisfying the plan D3 requirement of > 100 inputs.
//
// "Backward" definitions:
//
//	0001: re-apply SQL with IF NOT EXISTS — no-op, preserves rows (idempotent forward)
//	0002: ALTER TABLE turns DROP COLUMN for the 5 anchor cols, then re-ADD via SQL
//	0003: DROP VIEW for the 5 metric views, then re-CREATE via SQL
//
// modernc.org/sqlite v1.36.3 wraps SQLite 3.47+ which supports DROP COLUMN
// (added in SQLite 3.35.0, 2021-03-12).
package db_test

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// nMigrationGenerations is the number of random data shapes tested per
// round-trip property. Must be > 100 per plan D3.
const nMigrationGenerations = 100

// ── helpers ───────────────────────────────────────────────────────────────────

// migSQLDir returns the absolute path to internal/db/migrations/.
func migSQLDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile = .../internal/db/migration_property_test.go
	return filepath.Join(filepath.Dir(thisFile), "migrations")
}

// readMigSQL returns the content of a migration file by name.
func readMigSQL(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(migSQLDir(t), name))
	if err != nil {
		t.Fatalf("readMigSQL %s: %v", name, err)
	}
	return string(data)
}

// openMigMemDB opens an in-memory SQLite DB with all migrations applied and
// registers a Cleanup to close it. Use for tests that need the full schema.
func openMigMemDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("openMigMemDB: db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// migCountRows returns COUNT(*) from the given table or view.
func migCountRows(t *testing.T, d *db.DB, target string) int {
	t.Helper()
	var n int
	//nolint:gosec // target is always a compile-time literal in this file
	if err := d.QueryRow("SELECT COUNT(*) FROM " + target).Scan(&n); err != nil {
		t.Fatalf("migCountRows(%s): %v", target, err)
	}
	return n
}

// migExec executes a SQL statement, fatally failing the test on error.
func migExec(t *testing.T, d *db.DB, query string, args ...any) {
	t.Helper()
	if _, err := d.Exec(query, args...); err != nil {
		t.Fatalf("migExec %q: %v", query, err)
	}
}

// seedMigSession inserts one session row (prerequisite for turn FK).
func seedMigSession(t *testing.T, d *db.DB, id string) {
	t.Helper()
	migExec(t, d,
		`INSERT INTO sessions (id, started_at, mode) VALUES (?, '2026-01-01T00:00:00Z', 'solo')`,
		id,
	)
}

// seedMigTurn inserts one turn row linked to the given session.
func seedMigTurn(t *testing.T, d *db.DB, id, sessionID string, seqNo int) {
	t.Helper()
	migExec(t, d,
		`INSERT INTO turns (id, session_id, sequence_no, started_at, status, agent_id)
		 VALUES (?, ?, ?, '2026-01-01T00:00:00Z', 'complete', 'claude')`,
		id, sessionID, seqNo,
	)
}

// seedMigEntry inserts one entry row linked to the given session and turn.
func seedMigEntry(t *testing.T, d *db.DB, id, sessID, turnID, kind string) {
	t.Helper()
	migExec(t, d,
		`INSERT INTO entries (id, session_id, turn_id, kind, title, created_at, agent_id)
		 VALUES (?, ?, ?, ?, 'Test', '2026-01-01T00:00:00Z', 'claude')`,
		id, sessID, turnID, kind,
	)
}

// ── Property: 0001 Forward Schema ────────────────────────────────────────────

// TestMigration0001SchemaForwardProperty verifies that every db.Open creates
// exactly the tables, views, and indexes defined by 0001_init.sql.
// Checked across 100 independent in-memory DB openings for determinism.
func TestMigration0001SchemaForwardProperty(t *testing.T) {
	wantTables := []string{
		"sessions", "turns", "entries", "blockers",
		"decision_evidence_links", "_schema_migrations",
	}
	wantViews := []string{"decisions", "evidence"}
	wantIndexes := []string{
		"idx_turns_session", "idx_entries_session",
		"idx_entries_turn", "idx_entries_kind", "idx_blockers_turn",
	}

	for i := 0; i < nMigrationGenerations; i++ {
		d := openMigMemDB(t)

		for _, tbl := range wantTables {
			var n int
			if err := d.QueryRow(
				`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl,
			).Scan(&n); err != nil || n != 1 {
				t.Errorf("iter %d: table %q: count=%d err=%v", i, tbl, n, err)
			}
		}
		for _, vw := range wantViews {
			var n int
			if err := d.QueryRow(
				`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?`, vw,
			).Scan(&n); err != nil || n != 1 {
				t.Errorf("iter %d: view %q: count=%d err=%v", i, vw, n, err)
			}
		}
		for _, idx := range wantIndexes {
			var n int
			if err := d.QueryRow(
				`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx,
			).Scan(&n); err != nil || n != 1 {
				t.Errorf("iter %d: index %q: count=%d err=%v", i, idx, n, err)
			}
		}
	}
}

// TestMigration0001IdempotentForwardProperty verifies that re-applying the
// 0001 SQL (all CREATE TABLE/VIEW/INDEX IF NOT EXISTS) on a populated DB
// does not drop or modify existing rows — the migration is safe to replay.
// 100 random session counts tested.
func TestMigration0001IdempotentForwardProperty(t *testing.T) {
	sql0001 := readMigSQL(t, "0001_init.sql")
	rng := rand.New(rand.NewSource(1001))

	for i := 0; i < nMigrationGenerations; i++ {
		nSess := 1 + rng.Intn(15) // 1-15 sessions
		d := openMigMemDB(t)

		for j := 0; j < nSess; j++ {
			seedMigSession(t, d, fmt.Sprintf("sess-idp-%04d-%02d", i, j))
		}
		before := migCountRows(t, d, "sessions")

		// Re-apply 0001: all statements use IF NOT EXISTS → safe no-op.
		if _, err := d.Exec(sql0001); err != nil {
			t.Fatalf("iter %d: re-exec 0001_init.sql: %v", i, err)
		}

		after := migCountRows(t, d, "sessions")
		if after != before {
			t.Errorf("iter %d: 0001 idempotent re-apply changed session count %d → %d",
				i, before, after)
		}
	}
}

// ── Property: 0002 Round-trip ─────────────────────────────────────────────────

// anchorCols are the 5 columns added by 0002_turn_anchors.sql.
var anchorCols = []string{
	"intent", "constraints_json", "done_when", "drift_alerts", "anchor_last_shown_at",
}

// TestMigration0002RoundTripProperty verifies the 0002 migration round-trip:
//
//	forward (applied by db.Open) → seed N turns →
//	backward (DROP the 5 anchor columns) →
//	forward (re-ADD via 0002 SQL) →
//	COUNT(*) FROM turns = N
//
// 100 random turn counts tested; each iteration uses a dedicated DB to avoid
// cross-iteration interference.
func TestMigration0002RoundTripProperty(t *testing.T) {
	sql0002 := readMigSQL(t, "0002_turn_anchors.sql")
	rng := rand.New(rand.NewSource(2002))

	for i := 0; i < nMigrationGenerations; i++ {
		nTurns := 1 + rng.Intn(20) // 1-20 turns
		i, nTurns := i, nTurns    // capture for closure

		func() {
			d, err := db.Open(":memory:")
			if err != nil {
				t.Fatalf("iter %d: db.Open: %v", i, err)
			}
			defer func() { _ = d.Close() }()

			// Seed one session + nTurns turns.
			sessID := fmt.Sprintf("sess-rt2-%04d", i)
			seedMigSession(t, d, sessID)
			for j := 0; j < nTurns; j++ {
				seedMigTurn(t, d,
					fmt.Sprintf("turn-rt2-%04d-%02d", i, j),
					sessID, j+1,
				)
			}
			before := migCountRows(t, d, "turns")
			if before != nTurns {
				t.Fatalf("iter %d: seeded %d turns but count=%d", i, nTurns, before)
			}

			// Backward: drop the 5 anchor columns one by one.
			for _, col := range anchorCols {
				if _, err := d.Exec("ALTER TABLE turns DROP COLUMN " + col); err != nil {
					t.Fatalf("iter %d: backward DROP COLUMN %s: %v", i, col, err)
				}
			}

			// Forward: re-apply 0002 SQL to restore the anchor columns.
			if _, err := d.Exec(sql0002); err != nil {
				t.Fatalf("iter %d: forward re-apply 0002_turn_anchors.sql: %v", i, err)
			}

			// Invariant: row count must be unchanged.
			after := migCountRows(t, d, "turns")
			if after != before {
				t.Errorf("iter %d: 0002 round-trip changed turns count %d → %d",
					i, before, after)
			}
		}()
	}
}

// ── Property: 0003 Round-trip ─────────────────────────────────────────────────

// metricViewNames are the 5 views created by 0003_metric_views.sql.
var metricViewNames = []string{
	"metric_turn_duration",
	"metric_blocker_accumulation",
	"metric_agent_completion",
	"metric_agent_blocker_freq",
	"metric_evidence_bound_decisions",
}

// TestMigration0003RoundTripProperty verifies the 0003 migration round-trip:
//
//	forward (applied by db.Open) → seed data → record view counts →
//	backward (DROP the 5 metric views) →
//	forward (re-CREATE via 0003 SQL) →
//	view counts are identical to before
//
// Since views are non-materialised, the underlying base-table data is always
// preserved; the round-trip only risks breaking the view SQL.
// 100 random data shapes tested.
func TestMigration0003RoundTripProperty(t *testing.T) {
	sql0003 := readMigSQL(t, "0003_metric_views.sql")
	rng := rand.New(rand.NewSource(3003))

	for i := 0; i < nMigrationGenerations; i++ {
		nTurns := 1 + rng.Intn(10)    // 1-10 turns
		nDecisions := rng.Intn(6)     // 0-5 decision entries
		i, nTurns, nDecisions := i, nTurns, nDecisions

		func() {
			d, err := db.Open(":memory:")
			if err != nil {
				t.Fatalf("iter %d: db.Open: %v", i, err)
			}
			defer func() { _ = d.Close() }()

			// Seed one session + nTurns turns + nDecisions decision entries.
			sessID := fmt.Sprintf("sess-rt3-%04d", i)
			seedMigSession(t, d, sessID)
			for j := 0; j < nTurns; j++ {
				seedMigTurn(t, d,
					fmt.Sprintf("turn-rt3-%04d-%02d", i, j),
					sessID, j+1,
				)
			}
			turnID := fmt.Sprintf("turn-rt3-%04d-00", i)
			for j := 0; j < nDecisions; j++ {
				seedMigEntry(t, d,
					fmt.Sprintf("dec-rt3-%04d-%02d", i, j),
					sessID, turnID, "decision",
				)
			}

			// Record metric_turn_duration row count (= nTurns).
			beforeTurnRows := migCountRows(t, d, "metric_turn_duration")

			// Record metric_evidence_bound_decisions.total_count (= nDecisions).
			var beforeDecTotal int64
			if err := d.QueryRow(
				`SELECT total_count FROM metric_evidence_bound_decisions`,
			).Scan(&beforeDecTotal); err != nil {
				t.Fatalf("iter %d: query before dec total: %v", i, err)
			}

			// Backward: drop all 5 metric views.
			for _, vw := range metricViewNames {
				if _, err := d.Exec("DROP VIEW " + vw); err != nil {
					t.Fatalf("iter %d: backward DROP VIEW %s: %v", i, vw, err)
				}
			}

			// Forward: re-apply 0003 SQL (CREATE VIEW IF NOT EXISTS).
			if _, err := d.Exec(sql0003); err != nil {
				t.Fatalf("iter %d: forward re-apply 0003_metric_views.sql: %v", i, err)
			}

			// Invariant: metric_turn_duration row count unchanged.
			afterTurnRows := migCountRows(t, d, "metric_turn_duration")
			if afterTurnRows != beforeTurnRows {
				t.Errorf("iter %d: 0003 round-trip changed metric_turn_duration count %d → %d",
					i, beforeTurnRows, afterTurnRows)
			}

			// Invariant: metric_evidence_bound_decisions.total_count unchanged.
			var afterDecTotal int64
			if err := d.QueryRow(
				`SELECT total_count FROM metric_evidence_bound_decisions`,
			).Scan(&afterDecTotal); err != nil {
				t.Fatalf("iter %d: query after dec total: %v", i, err)
			}
			if afterDecTotal != beforeDecTotal {
				t.Errorf("iter %d: 0003 round-trip changed decision total_count %d → %d",
					i, beforeDecTotal, afterDecTotal)
			}
		}()
	}
}

// TestMigration0003AllViewsExistAfterForward verifies that all 5 metric views
// are present in sqlite_master after a fresh db.Open (forward migration).
func TestMigration0003AllViewsExistAfterForward(t *testing.T) {
	for i := 0; i < nMigrationGenerations; i++ {
		d := openMigMemDB(t)
		for _, vw := range metricViewNames {
			var n int
			if err := d.QueryRow(
				`SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?`, vw,
			).Scan(&n); err != nil || n != 1 {
				t.Errorf("iter %d: metric view %q: count=%d err=%v", i, vw, n, err)
			}
		}
	}
}
