package metrics_test

// schema_test.go validates that the JSON output of metrics.FormatJSON conforms
// to the frozen schema in testdata/golden/report_schema.json.
//
// FormatJSON lives in the metrics package (not cli) to avoid an import cycle
// (cli → metrics; metrics_test → cli would be circular). This keeps the
// schema validation test self-contained within the metrics package.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// ── schema mirror types ───────────────────────────────────────────────────────
// Mirror the frozen schema. Decode failures = structural incompatibility.

type schemaTop struct {
	Window      string         `json:"window"`
	Agent       string         `json:"agent"`
	GeneratedAt string         `json:"generated_at"`
	Metrics     schemaMetrics  `json:"metrics"`
}

type schemaMetrics struct {
	TurnDuration        []schemaTurnDuration        `json:"turn_duration"`
	BlockerAccumulation []schemaBlockerAccumulation `json:"blocker_accumulation"`
	AgentCompletion     []schemaAgentCompletion     `json:"agent_completion"`
	AgentBlockerFreq    []schemaAgentBlockerFreq    `json:"agent_blocker_freq"`
	EvidenceBound       schemaEvidenceBound         `json:"evidence_bound_decisions"`
}

type schemaTurnDuration struct {
	TurnID    string `json:"turn_id"`
	AgentID   string `json:"agent_id"`
	ElapsedMS *int64 `json:"elapsed_ms"`
}

type schemaBlockerAccumulation struct {
	BlockerID          string `json:"blocker_id"`
	OpenedAt           string `json:"opened_at"`
	ClosedAt           string `json:"closed_at"`
	AccumulatedSeconds int64  `json:"accumulated_seconds"`
}

type schemaAgentCompletion struct {
	AgentID        string  `json:"agent_id"`
	CompletionRate float64 `json:"completion_rate"`
	CompleteCount  int64   `json:"complete_count"`
	TotalCount     int64   `json:"total_count"`
}

type schemaAgentBlockerFreq struct {
	AgentID      string  `json:"agent_id"`
	BlockerFreq  float64 `json:"blocker_freq"`
	BlockerCount int64   `json:"blocker_count"`
	TurnCount    int64   `json:"turn_count"`
}

type schemaEvidenceBound struct {
	Ratio       float64 `json:"ratio"`
	LinkedCount int64   `json:"linked_count"`
	TotalCount  int64   `json:"total_count"`
}

// TestReportJSONSchema validates the JSON report output:
//  1. Is valid JSON that decodes into the schema mirror types
//  2. Contains all required top-level fields with correct types
//  3. Satisfies schema invariants (rates in [0,1], counts ≥ 0)
//  4. Matches expected values from the B-Exit fixture
//  5. The report_schema.json file itself is valid JSON with a $schema key
func TestReportJSONSchema(t *testing.T) {
	d := openFixtureDB(t)

	snap, err := metrics.QueryAll(d, metrics.Filter{})
	if err != nil {
		t.Fatalf("QueryAll: %v", err)
	}

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	raw, err := metrics.FormatJSON(snap, "all", "", fixedTime)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	// ── 1. Structural decode ─────────────────────────────────────────────────
	var out schemaTop
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("JSON unmarshal failed:\n%s\nerror: %v", raw, err)
	}

	// ── 2. Top-level required fields ─────────────────────────────────────────
	if out.Window == "" {
		t.Error("window field is empty")
	}
	if out.GeneratedAt == "" {
		t.Error("generated_at field is empty")
	}
	if out.GeneratedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("generated_at: got %q, want %q", out.GeneratedAt, "2026-01-01T00:00:00Z")
	}

	m := out.Metrics

	// ── 3. turn_duration ─────────────────────────────────────────────────────
	if len(m.TurnDuration) != 30 {
		t.Errorf("turn_duration count: got %d, want 30", len(m.TurnDuration))
	}
	for i, r := range m.TurnDuration {
		if r.TurnID == "" {
			t.Errorf("turn_duration[%d]: empty turn_id", i)
		}
	}

	// ── 4. blocker_accumulation ───────────────────────────────────────────────
	if len(m.BlockerAccumulation) != 3 {
		t.Errorf("blocker_accumulation count: got %d, want 3", len(m.BlockerAccumulation))
	}
	for _, r := range m.BlockerAccumulation {
		if r.AccumulatedSeconds < 0 {
			t.Errorf("blocker %s: negative accumulated_seconds %d", r.BlockerID, r.AccumulatedSeconds)
		}
	}

	// ── 5. agent_completion — rates in [0,1], total_count ≥ 1 ────────────────
	if len(m.AgentCompletion) != 2 {
		t.Errorf("agent_completion count: got %d, want 2", len(m.AgentCompletion))
	}
	for _, r := range m.AgentCompletion {
		if r.CompletionRate < 0 || r.CompletionRate > 1 {
			t.Errorf("agent %s: completion_rate %.6f out of [0,1]", r.AgentID, r.CompletionRate)
		}
		if r.TotalCount < 1 {
			t.Errorf("agent %s: total_count %d < 1", r.AgentID, r.TotalCount)
		}
	}

	// ── 6. agent_blocker_freq — freq ≥ 0 ─────────────────────────────────────
	if len(m.AgentBlockerFreq) != 2 {
		t.Errorf("agent_blocker_freq count: got %d, want 2", len(m.AgentBlockerFreq))
	}
	for _, r := range m.AgentBlockerFreq {
		if r.BlockerFreq < 0 {
			t.Errorf("agent %s: blocker_freq %.6f < 0", r.AgentID, r.BlockerFreq)
		}
	}

	// ── 7. evidence_bound_decisions — invariants ──────────────────────────────
	ebd := m.EvidenceBound
	if ebd.Ratio < 0 || ebd.Ratio > 1 {
		t.Errorf("evidence_bound ratio %.6f out of [0,1]", ebd.Ratio)
	}
	if ebd.LinkedCount > ebd.TotalCount {
		t.Errorf("evidence_bound: linked_count %d > total_count %d", ebd.LinkedCount, ebd.TotalCount)
	}

	// ── 8. Fixture-specific value assertions ──────────────────────────────────
	assertFloat(t, "schema evidence ratio", ebd.Ratio, 0.5)
	if ebd.LinkedCount != 4 {
		t.Errorf("evidence linked_count: got %d, want 4", ebd.LinkedCount)
	}
	if ebd.TotalCount != 8 {
		t.Errorf("evidence total_count: got %d, want 8", ebd.TotalCount)
	}

	// ── 9. report_schema.json is valid JSON with $schema key ─────────────────
	schemaPath := filepath.Join(fixtureDir(t), "golden", "report_schema.json")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read report_schema.json: %v", err)
	}
	var schemaDoc map[string]any
	if err := json.Unmarshal(schemaBytes, &schemaDoc); err != nil {
		t.Fatalf("report_schema.json is not valid JSON: %v", err)
	}
	if schemaDoc["$schema"] == nil {
		t.Error("report_schema.json missing $schema key")
	}

	// ── 10. Round-trip determinism ────────────────────────────────────────────
	raw2, err := metrics.FormatJSON(snap, "all", "", fixedTime)
	if err != nil {
		t.Fatalf("second FormatJSON: %v", err)
	}
	if string(raw) != string(raw2) {
		t.Error("FormatJSON output is not deterministic across two calls with same inputs")
	}
}
