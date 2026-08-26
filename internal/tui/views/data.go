// Package views provides data types, DB query functions, and ANSI rendering
// helpers for the Bubble Tea TUI views.
package views

import (
	"database/sql"
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// Session represents a work session row from the sessions table.
type Session struct {
	ID        string
	StartedAt string
	EndedAt   sql.NullString
	Mode      string
	AgentID   sql.NullString
	Title     sql.NullString
}

// Turn represents a work turn row from the turns table (including A.5 anchor cols).
type Turn struct {
	ID           string
	SessionID    string
	SequenceNo   int
	Title        sql.NullString
	StartedAt    string
	EndedAt      sql.NullString
	Status       string
	ElapsedMs    sql.NullInt64
	AgentID      sql.NullString
	Outcome      sql.NullString
	Lane         sql.NullString
	ParentTurnID sql.NullString
	// A.5 TIA anchor columns (0002_turn_anchors.sql)
	Intent          sql.NullString
	ConstraintsJSON sql.NullString
	DoneWhen        sql.NullString
}

// Entry represents a worklog entry row from the entries table.
type Entry struct {
	ID        string
	SessionID string
	TurnID    sql.NullString
	Kind      string
	Title     string
	Detail    sql.NullString
	CreatedAt string
	AgentID   sql.NullString
	Lane      sql.NullString
}

// Blocker represents a blocker state row tied to a blocker entry.
type Blocker struct {
	ID                 string
	TurnID             sql.NullString
	EntryID            sql.NullString
	Title              string
	OpenedAt           string
	ClosedAt           sql.NullString
	Status             string
	AccumulatedSeconds int64
	ResolutionNote     sql.NullString
}

// DecisionEvidenceLink represents an explicit decision→evidence relationship.
type DecisionEvidenceLink struct {
	DecisionEntryID string
	EvidenceEntryID string
}

// DecisionState represents ADR-lite lifecycle state for a decision entry.
type DecisionState struct {
	DecisionEntryID     string
	Status              string
	SupersededByEntryID sql.NullString
	SupersededAt        sql.NullString
	Rationale           sql.NullString
}

// WorklogData holds all data needed to render the TUI views.
type WorklogData struct {
	Sessions              []Session
	Turns                 []Turn
	Entries               []Entry
	Blockers              []Blocker
	DecisionEvidenceLinks []DecisionEvidenceLink
	DecisionStates        []DecisionState
	Attention             []metrics.AttentionItem
}

// LoadAll queries all worklog data from SQLite in display order.
func LoadAll(d *db.DB) (*WorklogData, error) {
	sessions, err := loadSessions(d)
	if err != nil {
		return nil, fmt.Errorf("tui: load sessions: %w", err)
	}
	turns, err := loadTurns(d)
	if err != nil {
		return nil, fmt.Errorf("tui: load turns: %w", err)
	}
	entries, err := loadEntries(d)
	if err != nil {
		return nil, fmt.Errorf("tui: load entries: %w", err)
	}
	blockers, err := loadBlockers(d)
	if err != nil {
		return nil, fmt.Errorf("tui: load blockers: %w", err)
	}
	links, err := loadDecisionEvidenceLinks(d)
	if err != nil {
		return nil, fmt.Errorf("tui: load decision evidence links: %w", err)
	}
	decisionStates, err := loadDecisionStates(d)
	if err != nil {
		return nil, fmt.Errorf("tui: load decision states: %w", err)
	}
	attention, err := metrics.QueryAttention(d, metrics.Filter{Window: "all"}, metrics.AttentionOptions{})
	if err != nil {
		return nil, fmt.Errorf("tui: load attention: %w", err)
	}
	return &WorklogData{
		Sessions:              sessions,
		Turns:                 turns,
		Entries:               entries,
		Blockers:              blockers,
		DecisionEvidenceLinks: links,
		DecisionStates:        decisionStates,
		Attention:             attention.Items,
	}, nil
}

func loadSessions(d *db.DB) ([]Session, error) {
	rows, err := d.Query(`
		SELECT id, started_at, ended_at, mode, agent_id, title
		FROM sessions ORDER BY started_at`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ss []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.StartedAt, &s.EndedAt, &s.Mode, &s.AgentID, &s.Title); err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, rows.Err()
}

func loadTurns(d *db.DB) ([]Turn, error) {
	rows, err := d.Query(`
		SELECT id, session_id, sequence_no, title, started_at, ended_at,
		       status, elapsed_ms, agent_id, outcome, lane, parent_turn_id,
		       intent, constraints_json, done_when
		FROM turns ORDER BY started_at, sequence_no`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ts []Turn
	for rows.Next() {
		var t Turn
		if err := rows.Scan(
			&t.ID, &t.SessionID, &t.SequenceNo, &t.Title,
			&t.StartedAt, &t.EndedAt, &t.Status, &t.ElapsedMs, &t.AgentID,
			&t.Outcome, &t.Lane, &t.ParentTurnID, &t.Intent, &t.ConstraintsJSON, &t.DoneWhen,
		); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, rows.Err()
}

func loadEntries(d *db.DB) ([]Entry, error) {
	rows, err := d.Query(`
		SELECT id, session_id, turn_id, kind, title, detail, created_at, agent_id, lane
		FROM entries ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var es []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.TurnID, &e.Kind,
			&e.Title, &e.Detail, &e.CreatedAt, &e.AgentID, &e.Lane,
		); err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return es, rows.Err()
}

func loadBlockers(d *db.DB) ([]Blocker, error) {
	rows, err := d.Query(`
		SELECT id, turn_id, entry_id, title, opened_at, closed_at,
		       status, accumulated_seconds, resolution_note
		FROM blockers ORDER BY status, opened_at`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var bs []Blocker
	for rows.Next() {
		var b Blocker
		if err := rows.Scan(
			&b.ID, &b.TurnID, &b.EntryID, &b.Title, &b.OpenedAt,
			&b.ClosedAt, &b.Status, &b.AccumulatedSeconds, &b.ResolutionNote,
		); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, rows.Err()
}

func loadDecisionEvidenceLinks(d *db.DB) ([]DecisionEvidenceLink, error) {
	rows, err := d.Query(`
		SELECT decision_entry_id, evidence_entry_id
		FROM decision_evidence_links ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var links []DecisionEvidenceLink
	for rows.Next() {
		var link DecisionEvidenceLink
		if err := rows.Scan(&link.DecisionEntryID, &link.EvidenceEntryID); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func loadDecisionStates(d *db.DB) ([]DecisionState, error) {
	rows, err := d.Query(`
		SELECT decision_entry_id, status, superseded_by_entry_id, superseded_at, rationale
		FROM decision_status ORDER BY decision_entry_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var states []DecisionState
	for rows.Next() {
		var state DecisionState
		if err := rows.Scan(
			&state.DecisionEntryID, &state.Status, &state.SupersededByEntryID,
			&state.SupersededAt, &state.Rationale,
		); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

// SeqSum returns a change-detection counter: sum of row counts across key tables.
// When this value increases the caller should reload data from DB.
func SeqSum(d *db.DB) (int64, error) {
	var sum int64
	err := d.QueryRow(`
		SELECT (SELECT COUNT(*) FROM entries) +
		       (SELECT COUNT(*) FROM turns) +
		       (SELECT COUNT(*) FROM sessions) +
		       (SELECT COUNT(*) FROM blockers) +
		       (SELECT COUNT(*) FROM decision_evidence_links) +
		       (SELECT COUNT(*) FROM decision_status)
	`).Scan(&sum)
	return sum, err
}
