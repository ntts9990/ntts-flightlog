package db_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// BenchmarkColdOpen measures the time to open a fresh SQLite DB and read the
// sqlite_version() row — the "cold open + first query" latency that forms the
// largest component of `flightlog entry` startup time.
//
// Plan target: median ≤ 60 ms on all 5 OS×arch CI targets (40 ms headroom
// under the 100 ms total budget). Run with -benchtime=5x to get 5 samples
// and report the median. If median > 60 ms on any target, CI fails and the
// plan halts for re-scoping per Principle 2 (no CGo fallback).
func BenchmarkColdOpen(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Fresh path each iteration → true cold open (no page-cache warm-up).
		dbPath := filepath.Join(dir, fmt.Sprintf("cold-%d.db", i))

		database, err := db.Open(dbPath)
		if err != nil {
			b.Fatalf("Open[%d]: %v", i, err)
		}

		ver, err := database.Version()
		if err != nil {
			b.Fatalf("Version[%d]: %v", i, err)
		}
		_ = ver

		if err := database.Close(); err != nil {
			b.Fatalf("Close[%d]: %v", i, err)
		}
	}
}
