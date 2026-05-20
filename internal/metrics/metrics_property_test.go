// Package metrics_test — property/contract tests for the 5 metric invariants.
//
// Ten randomly shaped SQLite DBs are generated via seeded RNG (deterministic).
// For each shape, all 5 metric queries are verified against their invariants:
//
//  1. evidence_bound_decisions.ratio ∈ [0, 1]
//  2. agent_completion.rate ∈ [0, 1] per agent
//  3. blocker_accumulation.accumulated_seconds ≥ 0
//  4. agent_blocker_freq.freq ≥ 0
//  5. linked_count ≤ total_count (evidence_bound)
//
// Shape 0 is always an empty DB to cover the zero-data edge case.
// Shapes 1-9 are randomly populated with sessions, turns, entries, blockers,
// and decision-evidence links.
package metrics_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// nMetricShapes is the number of random DB shapes exercised per property run.
// Set to 100 to satisfy plan D3's ≥100 generated inputs requirement.
// Each shape is a distinct seeded in-memory DB; all 5 invariants are checked
// for every metric row in every shape.
const nMetricShapes = 100

// metricAgents is the pool of agent names used when seeding random shapes.
var metricAgents = []string{"claude", "codex", "gemini"}

// metricStatuses are the valid turn status values.
var metricStatuses = []string{"complete", "abort", "abandon", "active"}

// TestMetricsPropertyInvariants exercises all 5 metric invariants across
// 10 randomly shaped in-memory DBs. Seeds are deterministic so CI is stable.
func TestMetricsPropertyInvariants(t *testing.T) {
	for shape := 0; shape < nMetricShapes; shape++ {
		shape := shape // capture for subtest
		t.Run(fmt.Sprintf("shape%02d", shape), func(t *testing.T) {
			var d *db.DB
			if shape == 0 {
				// Shape 0: empty DB — verifies zero-data edge cases.
				d = openPropDB(t)
			} else {
				// Shapes 1-99: random data, each with its own seed.
				rng := rand.New(rand.NewSource(int64(20260520 + shape*997)))
				d = openShapeDB(t, rng)
			}

			checkAllMetricInvariants(t, d)
		})
	}
}

// TestMetricsPropertyEvidenceRatioBounds uses the standard B-Exit fixture to
// verify that ratio = linked/total is always in [0,1] on real fixture data.
func TestMetricsPropertyEvidenceRatioBounds(t *testing.T) {
	d := openFixtureDB(t)
	ebd, err := metrics.QueryEvidenceBoundDecisions(d)
	if err != nil {
		t.Fatalf("QueryEvidenceBoundDecisions: %v", err)
	}
	if ebd.Ratio < 0 || ebd.Ratio > 1 {
		t.Errorf("fixture evidence ratio=%.10f not in [0,1]", ebd.Ratio)
	}
	if ebd.LinkedCount > ebd.TotalCount {
		t.Errorf("fixture: linked_count %d > total_count %d", ebd.LinkedCount, ebd.TotalCount)
	}
}

// TestMetricsPropertyCompletionRateBounds uses the standard B-Exit fixture to
// verify that agent_completion rates are in [0,1] on real fixture data.
func TestMetricsPropertyCompletionRateBounds(t *testing.T) {
	d := openFixtureDB(t)
	rows, err := metrics.QueryAgentCompletion(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentCompletion: %v", err)
	}
	for _, r := range rows {
		if r.CompletionRate < 0 || r.CompletionRate > 1 {
			t.Errorf("agent %s completion_rate=%.10f not in [0,1]", r.AgentID, r.CompletionRate)
		}
		if r.CompleteCount > r.TotalCount {
			t.Errorf("agent %s complete_count %d > total_count %d",
				r.AgentID, r.CompleteCount, r.TotalCount)
		}
	}
}

// TestMetricsPropertyBlockerSecondsNonNegative uses the standard B-Exit fixture
// to verify accumulated_seconds ≥ 0 on real fixture data.
func TestMetricsPropertyBlockerSecondsNonNegative(t *testing.T) {
	d := openFixtureDB(t)
	rows, err := metrics.QueryBlockerAccumulations(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryBlockerAccumulations: %v", err)
	}
	for _, r := range rows {
		if r.AccumulatedSeconds < 0 {
			t.Errorf("blocker %s accumulated_seconds=%d < 0", r.BlockerID, r.AccumulatedSeconds)
		}
	}
}

// TestMetricsPropertyBlockerFreqNonNegative uses the standard B-Exit fixture
// to verify blocker_freq ≥ 0 on real fixture data.
func TestMetricsPropertyBlockerFreqNonNegative(t *testing.T) {
	d := openFixtureDB(t)
	rows, err := metrics.QueryAgentBlockerFreq(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentBlockerFreq: %v", err)
	}
	for _, r := range rows {
		if r.BlockerFreq < 0 {
			t.Errorf("agent %s blocker_freq=%.10f < 0", r.AgentID, r.BlockerFreq)
		}
	}
}

// TestMetricsPropertyEmptyDBInvariants verifies that an empty DB (zero rows in
// all tables) returns well-formed zero/empty results for all 5 metrics.
func TestMetricsPropertyEmptyDBInvariants(t *testing.T) {
	d := openPropDB(t)
	checkAllMetricInvariants(t, d)

	// Additional empty-DB specific assertions.
	ebd, err := metrics.QueryEvidenceBoundDecisions(d)
	if err != nil {
		t.Fatalf("QueryEvidenceBoundDecisions empty: %v", err)
	}
	assertFloat(t, "empty DB evidence ratio", ebd.Ratio, 0.0)
	if ebd.TotalCount != 0 {
		t.Errorf("empty DB: total_count=%d want 0", ebd.TotalCount)
	}

	ac, err := metrics.QueryAgentCompletion(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentCompletion empty: %v", err)
	}
	if len(ac) != 0 {
		t.Errorf("empty DB: agent_completion rows=%d want 0", len(ac))
	}
}

// ── shape generator ───────────────────────────────────────────────────────────

// openShapeDB creates an in-memory DB populated with random sessions, turns,
// entries, blockers, and decision-evidence links driven by rng.
// The schema is applied by db.Open (all 3 migrations).
func openShapeDB(t *testing.T, rng *rand.Rand) *db.DB {
	t.Helper()
	d := openPropDB(t)

	nSessions := rng.Intn(4) // 0-3 sessions (0 tests empty-ish case)
	for s := 0; s < nSessions; s++ {
		sessID := fmt.Sprintf("prop-sess-%02d", s)
		agent := metricAgents[rng.Intn(len(metricAgents))]
		propExec(t, d,
			`INSERT INTO sessions (id, started_at, mode, agent_id)
			 VALUES (?, '2026-01-01T00:00:00Z', 'solo', ?)`,
			sessID, agent,
		)

		nTurns := 1 + rng.Intn(5) // 1-5 turns per session
		for turn := 0; turn < nTurns; turn++ {
			turnID := fmt.Sprintf("prop-turn-%02d-%02d", s, turn)
			turnAgent := metricAgents[rng.Intn(len(metricAgents))]
			status := metricStatuses[rng.Intn(len(metricStatuses))]
			elapsedMS := int64(rng.Intn(3_600_000)) // 0-1 h, always ≥ 0

			propExec(t, d,
				`INSERT INTO turns
				   (id, session_id, sequence_no, started_at, ended_at, status, elapsed_ms, agent_id)
				 VALUES (?, ?, ?, '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', ?, ?, ?)`,
				turnID, sessID, turn+1, status, elapsedMS, turnAgent,
			)

			// Optionally add a decision entry (1-in-3 chance).
			if rng.Intn(3) == 0 {
				decID := fmt.Sprintf("prop-dec-%02d-%02d", s, turn)
				propExec(t, d,
					`INSERT INTO entries
					   (id, session_id, turn_id, kind, title, created_at, agent_id)
					 VALUES (?, ?, ?, 'decision', 'Test decision', '2026-01-01T00:00:00Z', ?)`,
					decID, sessID, turnID, turnAgent,
				)

				// Optionally link an evidence entry to this decision (1-in-2).
				if rng.Intn(2) == 0 {
					evID := fmt.Sprintf("prop-ev-%02d-%02d", s, turn)
					propExec(t, d,
						`INSERT INTO entries
						   (id, session_id, turn_id, kind, title, created_at, agent_id)
						 VALUES (?, ?, ?, 'evidence', 'Test evidence', '2026-01-01T00:00:00Z', ?)`,
						evID, sessID, turnID, turnAgent,
					)
					propExec(t, d,
						`INSERT INTO decision_evidence_links
						   (decision_entry_id, evidence_entry_id, created_at)
						 VALUES (?, ?, '2026-01-01T00:00:00Z')`,
						decID, evID,
					)
				}
			}

			// Optionally add a blocker (1-in-4 chance).
			if rng.Intn(4) == 0 {
				blID := fmt.Sprintf("prop-bl-%02d-%02d", s, turn)
				accSecs := int64(rng.Intn(7200)) // always ≥ 0
				propExec(t, d,
					`INSERT INTO blockers
					   (id, turn_id, title, opened_at, status, accumulated_seconds)
					 VALUES (?, ?, 'Test blocker', '2026-01-01T00:00:00Z', 'resolved', ?)`,
					blID, turnID, accSecs,
				)
			}
		}
	}
	return d
}

// ── invariant checker ─────────────────────────────────────────────────────────

// checkAllMetricInvariants runs all 5 metric queries and asserts each invariant.
// Called for every shape in TestMetricsPropertyInvariants.
func checkAllMetricInvariants(t *testing.T, d *db.DB) {
	t.Helper()

	// Invariant 1 & 5: evidence_bound_decisions.ratio ∈ [0,1]; linked ≤ total.
	ebd, err := metrics.QueryEvidenceBoundDecisions(d)
	if err != nil {
		t.Fatalf("QueryEvidenceBoundDecisions: %v", err)
	}
	if ebd.Ratio < 0 || ebd.Ratio > 1 {
		t.Errorf("evidence_bound ratio=%.10f not in [0,1]", ebd.Ratio)
	}
	if ebd.LinkedCount < 0 {
		t.Errorf("evidence_bound linked_count=%d < 0", ebd.LinkedCount)
	}
	if ebd.TotalCount < 0 {
		t.Errorf("evidence_bound total_count=%d < 0", ebd.TotalCount)
	}
	if ebd.LinkedCount > ebd.TotalCount {
		t.Errorf("evidence_bound linked_count %d > total_count %d",
			ebd.LinkedCount, ebd.TotalCount)
	}

	// Invariant 2: agent_completion.rate ∈ [0,1] per agent.
	ac, err := metrics.QueryAgentCompletion(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentCompletion: %v", err)
	}
	for _, row := range ac {
		if row.CompletionRate < 0 || row.CompletionRate > 1 {
			t.Errorf("agent %s completion_rate=%.10f not in [0,1]",
				row.AgentID, row.CompletionRate)
		}
		if row.CompleteCount < 0 {
			t.Errorf("agent %s complete_count=%d < 0", row.AgentID, row.CompleteCount)
		}
		if row.TotalCount < 1 {
			t.Errorf("agent %s total_count=%d < 1 (view excludes NULL agent_id)",
				row.AgentID, row.TotalCount)
		}
		if row.CompleteCount > row.TotalCount {
			t.Errorf("agent %s complete_count %d > total_count %d",
				row.AgentID, row.CompleteCount, row.TotalCount)
		}
	}

	// Invariant 3: blocker_accumulation.accumulated_seconds ≥ 0.
	ba, err := metrics.QueryBlockerAccumulations(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryBlockerAccumulations: %v", err)
	}
	for _, row := range ba {
		if row.AccumulatedSeconds < 0 {
			t.Errorf("blocker %s accumulated_seconds=%d < 0",
				row.BlockerID, row.AccumulatedSeconds)
		}
	}

	// Invariant 4: agent_blocker_freq.freq ≥ 0.
	af, err := metrics.QueryAgentBlockerFreq(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAgentBlockerFreq: %v", err)
	}
	for _, row := range af {
		if row.BlockerFreq < 0 {
			t.Errorf("agent %s blocker_freq=%.10f < 0",
				row.AgentID, row.BlockerFreq)
		}
		if row.BlockerCount < 0 {
			t.Errorf("agent %s blocker_count=%d < 0", row.AgentID, row.BlockerCount)
		}
		if row.TurnCount < 1 {
			t.Errorf("agent %s turn_count=%d < 1 (view excludes NULL agent_id)",
				row.AgentID, row.TurnCount)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// openPropDB opens a fresh in-memory DB with all migrations applied.
func openPropDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("openPropDB: db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// propExec executes a SQL statement, fatally failing the test on error.
func propExec(t *testing.T, d *db.DB, query string, args ...any) {
	t.Helper()
	if _, err := d.Exec(query, args...); err != nil {
		t.Fatalf("propExec %q %v: %v", query, args, err)
	}
}
