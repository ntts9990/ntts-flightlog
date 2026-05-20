package metrics_test

// metrics_extra_test.go: additional coverage for window filter branches,
// FormatJSON nil-slice coercion, and agent-filter paths.

import (
	"strings"
	"testing"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// TestWindowFilterDay exercises the Filter{Window:"day"} → windowExpr path.
// Fixture data is dated in the past so day-window returns 0 rows; the test
// asserts no error rather than row count.
func TestWindowFilterDay(t *testing.T) {
	d := openFixtureDB(t)
	f := metrics.Filter{Window: "day"}

	rows, err := metrics.QueryTurnDurations(d, f)
	if err != nil {
		t.Fatalf("QueryTurnDurations(day): %v", err)
	}
	t.Logf("day-filtered turn durations: %d (fixture data is in the past)", len(rows))

	ba, err := metrics.QueryBlockerAccumulations(d, f)
	if err != nil {
		t.Fatalf("QueryBlockerAccumulations(day): %v", err)
	}
	t.Logf("day-filtered blocker accumulations: %d", len(ba))

	ac, err := metrics.QueryAgentCompletion(d, f)
	if err != nil {
		t.Fatalf("QueryAgentCompletion(day): %v", err)
	}
	t.Logf("day-filtered agent completion: %d", len(ac))
}

// TestWindowFilterWeek exercises the Filter{Window:"week"} → windowExpr path.
func TestWindowFilterWeek(t *testing.T) {
	d := openFixtureDB(t)
	f := metrics.Filter{Window: "week"}

	_, err := metrics.QueryTurnDurations(d, f)
	if err != nil {
		t.Fatalf("QueryTurnDurations(week): %v", err)
	}
	_, err = metrics.QueryBlockerAccumulations(d, f)
	if err != nil {
		t.Fatalf("QueryBlockerAccumulations(week): %v", err)
	}
	_, err = metrics.QueryAgentCompletion(d, f)
	if err != nil {
		t.Fatalf("QueryAgentCompletion(week): %v", err)
	}
}

// TestQueryAllWindowDay exercises QueryAll with a day window, covering the
// filter-pass-through inside QueryAll.
func TestQueryAllWindowDay(t *testing.T) {
	d := openFixtureDB(t)
	snap, err := metrics.QueryAll(d, metrics.Filter{Window: "day"})
	if err != nil {
		t.Fatalf("QueryAll(day): %v", err)
	}
	// Just verify no panic and snapshot is structurally valid.
	if snap == nil {
		t.Fatal("QueryAll returned nil snapshot")
	}
}

// TestQueryAgentBlockerFreqFilter covers the agent-filter branch inside
// QueryAgentBlockerFreq (different from the already-tested claude-filter in
// metrics_test.go which goes through QueryAgentCompletion).
func TestQueryAgentBlockerFreqFilter(t *testing.T) {
	d := openFixtureDB(t)

	// claude branch.
	rows, err := metrics.QueryAgentBlockerFreq(d, metrics.Filter{Agent: "claude"})
	if err != nil {
		t.Fatalf("QueryAgentBlockerFreq(claude): %v", err)
	}
	if len(rows) != 1 || rows[0].AgentID != "claude" {
		t.Errorf("expected 1 claude row, got %v", rows)
	}

	// codex branch.
	rows, err = metrics.QueryAgentBlockerFreq(d, metrics.Filter{Agent: "codex"})
	if err != nil {
		t.Fatalf("QueryAgentBlockerFreq(codex): %v", err)
	}
	if len(rows) != 1 || rows[0].AgentID != "codex" {
		t.Errorf("expected 1 codex row, got %v", rows)
	}
}

// TestFormatJSONWithAgent covers the agent field serialisation path in FormatJSON.
func TestFormatJSONWithAgent(t *testing.T) {
	d := openFixtureDB(t)
	snap, err := metrics.QueryAll(d, metrics.Filter{Agent: "claude"})
	if err != nil {
		t.Fatalf("QueryAll: %v", err)
	}

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	raw, err := metrics.FormatJSON(snap, "all", "claude", fixedTime)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}
	if !strings.Contains(string(raw), `"agent": "claude"`) {
		t.Errorf("FormatJSON missing agent field:\n%s", raw)
	}
}

// TestFormatJSONNilSlices covers the nil→[] coercion branches inside FormatJSON.
// These branches fire when QueryAll returns nil slices (empty DB → no rows).
func TestFormatJSONNilSlices(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	snap, err := metrics.QueryAll(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAll on empty DB: %v", err)
	}

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	raw, err := metrics.FormatJSON(snap, "all", "", fixedTime)
	if err != nil {
		t.Fatalf("FormatJSON with nil slices: %v", err)
	}

	// nil slices must serialize as [] not null (schema stability requirement).
	if strings.Contains(string(raw), ": null") {
		t.Errorf("FormatJSON produced null (expected []): %s", raw)
	}
	if !strings.Contains(string(raw), `"turn_duration": []`) {
		t.Errorf("turn_duration not [] in output:\n%s", raw)
	}
}

// TestFormatJSONDeterminism verifies FormatJSON is deterministic across two calls
// with the same inputs (covers the determinism invariant from schema_test.go
// but at the unit level without fixture DB dependency).
func TestFormatJSONDeterminism(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	snap, err := metrics.QueryAll(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAll: %v", err)
	}

	fixedTime := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	r1, err := metrics.FormatJSON(snap, "week", "codex", fixedTime)
	if err != nil {
		t.Fatalf("FormatJSON call 1: %v", err)
	}
	r2, err := metrics.FormatJSON(snap, "week", "codex", fixedTime)
	if err != nil {
		t.Fatalf("FormatJSON call 2: %v", err)
	}
	if string(r1) != string(r2) {
		t.Error("FormatJSON is not deterministic across two calls with the same inputs")
	}
}
