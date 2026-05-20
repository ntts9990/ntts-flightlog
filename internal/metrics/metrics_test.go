package metrics_test

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// fixtureDir returns the absolute path to the testdata directory at the
// project root, regardless of where the test binary runs from.
func fixtureDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// thisFile = .../internal/metrics/metrics_test.go → ../../testdata
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata")
}

// openFixtureDB creates an in-memory SQLite DB, applies all migrations (which
// creates the metric views), then seeds it with the B-Exit fixture SQL.
func openFixtureDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	fixturePath := filepath.Join(fixtureDir(t), "metric_fixtures", "fixture_10s_3t_47e.sql")
	sql, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if _, err := d.Exec(string(sql)); err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return d
}

const eps = 1e-9 // tolerance for float64 comparisons

func assertFloat(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Errorf("%s: got %.15f, want %.15f", label, got, want)
	}
}

// ── Test 1: metric_turn_duration ─────────────────────────────────────────────

func TestQueryTurnDurations(t *testing.T) {
	d := openFixtureDB(t)
	f := metrics.Filter{} // no filter → all data

	rows, err := metrics.QueryTurnDurations(d, f)
	if err != nil {
		t.Fatalf("QueryTurnDurations: %v", err)
	}

	// Fixture: 10 sessions × 3 turns = 30 turns, all with elapsed_ms set.
	const wantCount = 30
	if len(rows) != wantCount {
		t.Errorf("turn_duration count: got %d, want %d", len(rows), wantCount)
	}

	// All rows must have non-empty turn_id and non-nil elapsed_ms.
	for i, r := range rows {
		if r.TurnID == "" {
			t.Errorf("row[%d]: empty turn_id", i)
		}
		if r.ElapsedMS == nil {
			t.Errorf("row[%d] turn_id=%s: nil elapsed_ms", i, r.TurnID)
		}
	}

	// Count by agent.
	agentCounts := map[string]int{}
	for _, r := range rows {
		agentCounts[r.AgentID]++
	}
	if agentCounts["claude"] != 24 {
		t.Errorf("claude turn count: got %d, want 24", agentCounts["claude"])
	}
	if agentCounts["codex"] != 6 {
		t.Errorf("codex turn count: got %d, want 6", agentCounts["codex"])
	}
}

// ── Test 2: metric_blocker_accumulation ──────────────────────────────────────

func TestQueryBlockerAccumulations(t *testing.T) {
	d := openFixtureDB(t)

	rows, err := metrics.QueryBlockerAccumulations(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryBlockerAccumulations: %v", err)
	}

	// Fixture: 3 blockers (2 resolved, 1 open).
	if len(rows) != 3 {
		t.Fatalf("blocker_accumulation count: got %d, want 3", len(rows))
	}

	// Build map for easy lookup.
	m := map[string]metrics.BlockerAccumulation{}
	for _, r := range rows {
		m[r.BlockerID] = r
	}

	bl1, ok := m["BL1"]
	if !ok {
		t.Fatal("BL1 not found")
	}
	if bl1.AccumulatedSeconds != 3600 {
		t.Errorf("BL1 accumulated_seconds: got %d, want 3600", bl1.AccumulatedSeconds)
	}
	if bl1.ClosedAt == "" {
		t.Error("BL1 closed_at should be non-empty (resolved)")
	}

	bl2, ok := m["BL2"]
	if !ok {
		t.Fatal("BL2 not found")
	}
	if bl2.AccumulatedSeconds != 1800 {
		t.Errorf("BL2 accumulated_seconds: got %d, want 1800", bl2.AccumulatedSeconds)
	}

	bl3, ok := m["BL3"]
	if !ok {
		t.Fatal("BL3 not found")
	}
	if bl3.AccumulatedSeconds != 0 {
		t.Errorf("BL3 accumulated_seconds: got %d, want 0", bl3.AccumulatedSeconds)
	}
	if bl3.ClosedAt != "" {
		t.Errorf("BL3 closed_at: got %q, want empty (open blocker)", bl3.ClosedAt)
	}
}

// ── Test 3: metric_agent_completion ──────────────────────────────────────────

func TestQueryAgentCompletion(t *testing.T) {
	d := openFixtureDB(t)

	rows, err := metrics.QueryAgentCompletion(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentCompletion: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("agent_completion row count: got %d, want 2", len(rows))
	}

	m := map[string]metrics.AgentCompletion{}
	for _, r := range rows {
		m[r.AgentID] = r
	}

	// claude: 23 complete + 1 abort = 24 turns → rate = 23/24
	claude := m["claude"]
	if claude.CompleteCount != 23 {
		t.Errorf("claude complete_count: got %d, want 23", claude.CompleteCount)
	}
	if claude.TotalCount != 24 {
		t.Errorf("claude total_count: got %d, want 24", claude.TotalCount)
	}
	assertFloat(t, "claude completion_rate", claude.CompletionRate, 23.0/24.0)

	// codex: 5 complete + 1 abandon = 6 turns → rate = 5/6
	codex := m["codex"]
	if codex.CompleteCount != 5 {
		t.Errorf("codex complete_count: got %d, want 5", codex.CompleteCount)
	}
	if codex.TotalCount != 6 {
		t.Errorf("codex total_count: got %d, want 6", codex.TotalCount)
	}
	assertFloat(t, "codex completion_rate", codex.CompletionRate, 5.0/6.0)
}

// ── Test 4: metric_agent_blocker_freq ────────────────────────────────────────

func TestQueryAgentBlockerFreq(t *testing.T) {
	d := openFixtureDB(t)

	rows, err := metrics.QueryAgentBlockerFreq(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentBlockerFreq: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("agent_blocker_freq row count: got %d, want 2", len(rows))
	}

	m := map[string]metrics.AgentBlockerFreq{}
	for _, r := range rows {
		m[r.AgentID] = r
	}

	// claude: 3 blockers across 24 turns → freq = 3/24 = 0.125
	claude := m["claude"]
	if claude.BlockerCount != 3 {
		t.Errorf("claude blocker_count: got %d, want 3", claude.BlockerCount)
	}
	if claude.TurnCount != 24 {
		t.Errorf("claude turn_count: got %d, want 24", claude.TurnCount)
	}
	assertFloat(t, "claude blocker_freq", claude.BlockerFreq, 3.0/24.0)

	// codex: 0 blockers across 6 turns → freq = 0.0
	codex := m["codex"]
	if codex.BlockerCount != 0 {
		t.Errorf("codex blocker_count: got %d, want 0", codex.BlockerCount)
	}
	if codex.TurnCount != 6 {
		t.Errorf("codex turn_count: got %d, want 6", codex.TurnCount)
	}
	assertFloat(t, "codex blocker_freq", codex.BlockerFreq, 0.0)
}

// ── Test 5: metric_evidence_bound_decisions ───────────────────────────────────

func TestQueryEvidenceBoundDecisions(t *testing.T) {
	d := openFixtureDB(t)

	ebd, err := metrics.QueryEvidenceBoundDecisions(d)
	if err != nil {
		t.Fatalf("QueryEvidenceBoundDecisions: %v", err)
	}

	// 8 decisions total; 4 have linked evidence → ratio = 4/8 = 0.5
	if ebd.TotalCount != 8 {
		t.Errorf("total_count: got %d, want 8", ebd.TotalCount)
	}
	if ebd.LinkedCount != 4 {
		t.Errorf("linked_count: got %d, want 4", ebd.LinkedCount)
	}
	assertFloat(t, "evidence_bound ratio", ebd.Ratio, 0.5)
}

// ── Test 6: QueryAll (Snapshot) ───────────────────────────────────────────────

func TestQueryAll(t *testing.T) {
	d := openFixtureDB(t)

	snap, err := metrics.QueryAll(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAll: %v", err)
	}

	if len(snap.TurnDurations) != 30 {
		t.Errorf("TurnDurations count: got %d, want 30", len(snap.TurnDurations))
	}
	if len(snap.BlockerAccumulations) != 3 {
		t.Errorf("BlockerAccumulations count: got %d, want 3", len(snap.BlockerAccumulations))
	}
	if len(snap.AgentCompletion) != 2 {
		t.Errorf("AgentCompletion count: got %d, want 2", len(snap.AgentCompletion))
	}
	if len(snap.AgentBlockerFreq) != 2 {
		t.Errorf("AgentBlockerFreq count: got %d, want 2", len(snap.AgentBlockerFreq))
	}
	assertFloat(t, "EvidenceBound.Ratio", snap.EvidenceBound.Ratio, 0.5)
}

// ── Test 7: Filter.Agent ─────────────────────────────────────────────────────

func TestQueryAgentFilter(t *testing.T) {
	d := openFixtureDB(t)

	// Filter to claude only.
	fClaude := metrics.Filter{Agent: "claude"}

	td, err := metrics.QueryTurnDurations(d, fClaude)
	if err != nil {
		t.Fatalf("QueryTurnDurations(claude): %v", err)
	}
	if len(td) != 24 {
		t.Errorf("claude turn_duration count: got %d, want 24", len(td))
	}

	ac, err := metrics.QueryAgentCompletion(d, fClaude)
	if err != nil {
		t.Fatalf("QueryAgentCompletion(claude): %v", err)
	}
	if len(ac) != 1 || ac[0].AgentID != "claude" {
		t.Errorf("AgentCompletion filter: got %v", ac)
	}
}

// ── Test 8: empty DB returns zero/empty results ───────────────────────────────

func TestEmptyDB(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	td, err := metrics.QueryTurnDurations(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryTurnDurations on empty DB: %v", err)
	}
	if len(td) != 0 {
		t.Errorf("expected 0 turn durations on empty DB, got %d", len(td))
	}

	ebd, err := metrics.QueryEvidenceBoundDecisions(d)
	if err != nil {
		t.Fatalf("QueryEvidenceBoundDecisions on empty DB: %v", err)
	}
	// No decisions → ratio = 0.0, counts = 0.
	if ebd.TotalCount != 0 {
		t.Errorf("empty DB: total_count %d, want 0", ebd.TotalCount)
	}
	assertFloat(t, "empty DB evidence ratio", ebd.Ratio, 0.0)
}
