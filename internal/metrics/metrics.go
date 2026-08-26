// Package metrics provides the 5 core analytic metric query functions for
// ntts-flightlog v2 Phase B3. Each function queries one of the SQL views
// defined in internal/db/migrations/0003_metric_views.sql.
//
// 5 views / metrics:
//
//  1. metric_turn_duration        — Turn 소요시간 분포
//  2. metric_blocker_accumulation — Blocker 누적시간
//  3. metric_agent_completion     — Agent별 Turn 완료율
//  4. metric_agent_blocker_freq   — Agent별 Blocker 빈도
//  5. metric_evidence_bound_decisions — Evidence가 붙은 Decision 비율
//
// FormatJSON lives here (not in internal/cli) so that internal/metrics tests
// can call it without creating an import cycle (cli → metrics → cli).
package metrics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// ── Filter ───────────────────────────────────────────────────────────────────

// Filter constrains report queries by time window and/or agent name.
// Zero values mean "no constraint" (query all data).
type Filter struct {
	Window string // "day" | "week" | "all" (empty or "all" = no filter)
	Agent  string // empty = all agents
}

// windowExpr returns a SQLite datetime expression for the window lower bound,
// or empty string if the window is "all" or unset (no time filter).
func (f Filter) windowExpr() string {
	switch f.Window {
	case "day":
		return "datetime('now', '-1 day')"
	case "week":
		return "datetime('now', '-7 days')"
	default:
		return ""
	}
}

// ── Metric 1: TurnDuration ───────────────────────────────────────────────────

// TurnDuration holds one row from the metric_turn_duration view.
type TurnDuration struct {
	TurnID    string `json:"turn_id"`
	AgentID   string `json:"agent_id"`  // empty string when NULL in DB
	ElapsedMS *int64 `json:"elapsed_ms"` // nil when turn has not ended yet
}

// QueryTurnDurations returns all rows from metric_turn_duration.
// Optionally filtered by window (turns.started_at) and/or agent name via a
// JOIN to the underlying turns table.
func QueryTurnDurations(d *db.DB, f Filter) ([]TurnDuration, error) {
	// Join to turns so we can filter by started_at without embedding it in the view.
	q := `SELECT m.turn_id, COALESCE(m.agent_id,''), m.elapsed_ms
	      FROM metric_turn_duration m
	      JOIN turns t ON t.id = m.turn_id
	      WHERE 1=1`
	var args []any
	if ws := f.windowExpr(); ws != "" {
		q += " AND t.started_at >= " + ws
	}
	if f.Agent != "" {
		q += " AND m.agent_id = ?"
		args = append(args, f.Agent)
	}
	q += " ORDER BY t.started_at, t.sequence_no"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("metric_turn_duration: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []TurnDuration
	for rows.Next() {
		var td TurnDuration
		if err := rows.Scan(&td.TurnID, &td.AgentID, &td.ElapsedMS); err != nil {
			return nil, fmt.Errorf("metric_turn_duration scan: %w", err)
		}
		out = append(out, td)
	}
	return out, rows.Err()
}

// ── Metric 2: BlockerAccumulation ───────────────────────────────────────────

// BlockerAccumulation holds one row from the metric_blocker_accumulation view.
type BlockerAccumulation struct {
	BlockerID          string `json:"blocker_id"`
	OpenedAt           string `json:"opened_at"`
	ClosedAt           string `json:"closed_at"`           // empty string when still open
	AccumulatedSeconds int64  `json:"accumulated_seconds"`
}

// QueryBlockerAccumulations returns all rows from metric_blocker_accumulation,
// optionally filtered by window (blockers.opened_at).
func QueryBlockerAccumulations(d *db.DB, f Filter) ([]BlockerAccumulation, error) {
	q := `SELECT blocker_id, opened_at, COALESCE(closed_at,''), accumulated_seconds
	      FROM metric_blocker_accumulation
	      WHERE 1=1`
	var args []any
	if ws := f.windowExpr(); ws != "" {
		q += " AND opened_at >= " + ws
	}
	q += " ORDER BY opened_at"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("metric_blocker_accumulation: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []BlockerAccumulation
	for rows.Next() {
		var ba BlockerAccumulation
		if err := rows.Scan(&ba.BlockerID, &ba.OpenedAt, &ba.ClosedAt, &ba.AccumulatedSeconds); err != nil {
			return nil, fmt.Errorf("metric_blocker_accumulation scan: %w", err)
		}
		out = append(out, ba)
	}
	return out, rows.Err()
}

// ── Metric 3: AgentCompletion ────────────────────────────────────────────────

// AgentCompletion holds one row from the metric_agent_completion view.
type AgentCompletion struct {
	AgentID        string  `json:"agent_id"`
	CompletionRate float64 `json:"completion_rate"`
	CompleteCount  int64   `json:"complete_count"`
	TotalCount     int64   `json:"total_count"`
}

// QueryAgentCompletion returns all rows from metric_agent_completion,
// optionally filtered by agent name.
func QueryAgentCompletion(d *db.DB, f Filter) ([]AgentCompletion, error) {
	q := `SELECT agent_id, completion_rate, complete_count, total_count
	      FROM metric_agent_completion
	      WHERE 1=1`
	var args []any
	if f.Agent != "" {
		q += " AND agent_id = ?"
		args = append(args, f.Agent)
	}
	q += " ORDER BY agent_id"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("metric_agent_completion: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []AgentCompletion
	for rows.Next() {
		var ac AgentCompletion
		if err := rows.Scan(&ac.AgentID, &ac.CompletionRate, &ac.CompleteCount, &ac.TotalCount); err != nil {
			return nil, fmt.Errorf("metric_agent_completion scan: %w", err)
		}
		out = append(out, ac)
	}
	return out, rows.Err()
}

// ── Metric 4: AgentBlockerFreq ───────────────────────────────────────────────

// AgentBlockerFreq holds one row from the metric_agent_blocker_freq view.
type AgentBlockerFreq struct {
	AgentID      string  `json:"agent_id"`
	BlockerFreq  float64 `json:"blocker_freq"`
	BlockerCount int64   `json:"blocker_count"`
	TurnCount    int64   `json:"turn_count"`
}

// QueryAgentBlockerFreq returns all rows from metric_agent_blocker_freq,
// optionally filtered by agent name.
func QueryAgentBlockerFreq(d *db.DB, f Filter) ([]AgentBlockerFreq, error) {
	q := `SELECT agent_id, blocker_freq, blocker_count, turn_count
	      FROM metric_agent_blocker_freq
	      WHERE 1=1`
	var args []any
	if f.Agent != "" {
		q += " AND agent_id = ?"
		args = append(args, f.Agent)
	}
	q += " ORDER BY agent_id"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("metric_agent_blocker_freq: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []AgentBlockerFreq
	for rows.Next() {
		var af AgentBlockerFreq
		if err := rows.Scan(&af.AgentID, &af.BlockerFreq, &af.BlockerCount, &af.TurnCount); err != nil {
			return nil, fmt.Errorf("metric_agent_blocker_freq scan: %w", err)
		}
		out = append(out, af)
	}
	return out, rows.Err()
}

// ── Metric 5: EvidenceBoundDecisions ────────────────────────────────────────

// EvidenceBoundDecisions holds the single aggregate row from
// metric_evidence_bound_decisions.
type EvidenceBoundDecisions struct {
	Ratio       float64 `json:"ratio"`
	LinkedCount int64   `json:"linked_count"`
	TotalCount  int64   `json:"total_count"`
}

// QueryEvidenceBoundDecisions returns the single aggregate evidence-bound
// decision ratio. The view always returns exactly one row.
func QueryEvidenceBoundDecisions(d *db.DB) (*EvidenceBoundDecisions, error) {
	row := d.QueryRow(`SELECT ratio, linked_count, total_count
	                   FROM metric_evidence_bound_decisions`)
	var ebd EvidenceBoundDecisions
	if err := row.Scan(&ebd.Ratio, &ebd.LinkedCount, &ebd.TotalCount); err != nil {
		return nil, fmt.Errorf("metric_evidence_bound_decisions: %w", err)
	}
	return &ebd, nil
}

// ── Snapshot ─────────────────────────────────────────────────────────────────

// Snapshot holds all 5 metrics computed from the same DB state.
// Used by the report command (B4) to render text or JSON output.
type Snapshot struct {
	TurnDurations        []TurnDuration        `json:"turn_duration"`
	BlockerAccumulations []BlockerAccumulation  `json:"blocker_accumulation"`
	AgentCompletion      []AgentCompletion      `json:"agent_completion"`
	AgentBlockerFreq     []AgentBlockerFreq     `json:"agent_blocker_freq"`
	EvidenceBound        EvidenceBoundDecisions `json:"evidence_bound_decisions"`
}

// QueryAll queries all 5 metrics with the given filter and returns a Snapshot.
func QueryAll(d *db.DB, f Filter) (*Snapshot, error) {
	td, err := QueryTurnDurations(d, f)
	if err != nil {
		return nil, err
	}
	ba, err := QueryBlockerAccumulations(d, f)
	if err != nil {
		return nil, err
	}
	ac, err := QueryAgentCompletion(d, f)
	if err != nil {
		return nil, err
	}
	af, err := QueryAgentBlockerFreq(d, f)
	if err != nil {
		return nil, err
	}
	ebd, err := QueryEvidenceBoundDecisions(d)
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		TurnDurations:        td,
		BlockerAccumulations: ba,
		AgentCompletion:      ac,
		AgentBlockerFreq:     af,
		EvidenceBound:        *ebd,
	}, nil
}

// ── JSON serialisation ────────────────────────────────────────────────────────

// ReportJSON is the schema-stable top-level structure for
// `flightlog report --format json`. Structure is frozen in
// testdata/golden/report_schema.json and validated by schema_test.go.
type ReportJSON struct {
	Window      string         `json:"window"`
	Agent       string         `json:"agent"`
	GeneratedAt string         `json:"generated_at"`
	Metrics     ReportMetrics  `json:"metrics"`
}

// ReportMetrics wraps the 5 metric slices/structs for JSON output.
type ReportMetrics struct {
	TurnDuration        []TurnDuration        `json:"turn_duration"`
	BlockerAccumulation []BlockerAccumulation `json:"blocker_accumulation"`
	AgentCompletion     []AgentCompletion     `json:"agent_completion"`
	AgentBlockerFreq    []AgentBlockerFreq    `json:"agent_blocker_freq"`
	EvidenceBound       EvidenceBoundDecisions `json:"evidence_bound_decisions"`
}

// FormatJSON serialises the snapshot to indented JSON suitable for
// `flightlog report --format json`.
//
// generatedAt is injected so tests can use a fixed timestamp; pass
// time.Time{} (zero value) to use time.Now().UTC() at call time.
func FormatJSON(snap *Snapshot, window, agent string, generatedAt time.Time) ([]byte, error) {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	// Ensure nil slices marshal as [] not null (schema stability).
	tds := snap.TurnDurations
	if tds == nil {
		tds = []TurnDuration{}
	}
	bas := snap.BlockerAccumulations
	if bas == nil {
		bas = []BlockerAccumulation{}
	}
	acs := snap.AgentCompletion
	if acs == nil {
		acs = []AgentCompletion{}
	}
	afs := snap.AgentBlockerFreq
	if afs == nil {
		afs = []AgentBlockerFreq{}
	}

	out := ReportJSON{
		Window:      window,
		Agent:       agent,
		GeneratedAt: generatedAt.Format("2006-01-02T15:04:05Z"),
		Metrics: ReportMetrics{
			TurnDuration:        tds,
			BlockerAccumulation: bas,
			AgentCompletion:     acs,
			AgentBlockerFreq:    afs,
			EvidenceBound:       snap.EvidenceBound,
		},
	}
	return json.MarshalIndent(out, "", "  ")
}
