package db_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// TestConcurrentReadWrite spawns 2 goroutines against a single WAL-mode DB:
//   - writer: 100 INSERT INTO entries
//   - reader: 100 SELECT from the decisions view
//
// Asserts zero SQLITE_BUSY errors and total elapsed < 1 s.
// Must pass on all 5 OS×arch CI targets (plan A2, A-Exit criteria).
func TestConcurrentReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "concurrent.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// Seed prerequisites: one session + one turn (FK constraints).
	if _, err := database.Exec(
		`INSERT INTO sessions (id, started_at, mode) VALUES ('sess-c1', '2026-01-01T00:00:00Z', 'solo')`,
	); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO turns (id, session_id, sequence_no, started_at) VALUES ('turn-c1', 'sess-c1', 1, '2026-01-01T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed turn: %v", err)
	}

	const ops = 100
	start := time.Now()
	var wg sync.WaitGroup
	errCh := make(chan error, ops*2)

	// Goroutine 1: 100 INSERT INTO entries.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < ops; i++ {
			_, err := database.Exec(
				`INSERT INTO entries (id, session_id, turn_id, kind, title, created_at)
				 VALUES (?, 'sess-c1', 'turn-c1', 'decision', ?, '2026-01-01T00:00:00Z')`,
				fmt.Sprintf("e-w-%04d", i),
				fmt.Sprintf("concurrent write entry %d", i),
			)
			if err != nil {
				errCh <- fmt.Errorf("writer[%d]: %w", i, err)
			}
		}
	}()

	// Goroutine 2: 100 SELECT from the decisions view (report-style query).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < ops; i++ {
			rows, err := database.Query(`SELECT id, title, created_at FROM decisions ORDER BY created_at LIMIT 50`)
			if err != nil {
				errCh <- fmt.Errorf("reader[%d]: %w", i, err)
				continue
			}
			_ = rows.Close()
		}
	}()

	wg.Wait()
	close(errCh)

	elapsed := time.Since(start)

	// Check for SQLITE_BUSY or any other errors.
	var busyCount, otherCount int
	for e := range errCh {
		if strings.Contains(e.Error(), "SQLITE_BUSY") || strings.Contains(e.Error(), "database is locked") {
			busyCount++
		} else {
			otherCount++
		}
		t.Errorf("concurrent error: %v", e)
	}
	if busyCount > 0 {
		t.Errorf("got %d SQLITE_BUSY errors (WAL+busy_timeout=5000 should prevent these)", busyCount)
	}
	if otherCount > 0 {
		t.Errorf("got %d non-BUSY errors", otherCount)
	}

	if elapsed >= time.Second {
		t.Errorf("total elapsed %v ≥ 1s budget", elapsed)
	}
	t.Logf("concurrent test: %d writes + %d reads in %v (zero SQLITE_BUSY)", ops, ops, elapsed)
}
