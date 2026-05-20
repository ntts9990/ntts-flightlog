// Package views provides data types, DB query functions, and ANSI rendering
// helpers for the Bubble Tea TUI views.
package views

import (
	"database/sql"
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
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
	ID         string
	SessionID  string
	SequenceNo int
	Title      sql.NullString
	StartedAt  string
	EndedAt    sql.NullString
	Status     string
	ElapsedMs  sql.NullInt64
	AgentID    sql.NullString
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
}

// WorklogData holds all data needed to render the TUI views.
type WorklogData struct {
	Sessions []Session
	Turns    []Turn
	Entries  []Entry
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
	return &WorklogData{Sessions: sessions, Turns: turns, Entries: entries}, nil
}

func loadSessions(d *db.DB) ([]Session, error) {
	rows, err := d.Query(`
		SELECT id, started_at, ended_at, mode, agent_id, title
		FROM sessions ORDER BY started_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
		       status, elapsed_ms, agent_id,
		       intent, constraints_json, done_when
		FROM turns ORDER BY started_at, sequence_no`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ts []Turn
	for rows.Next() {
		var t Turn
		if err := rows.Scan(
			&t.ID, &t.SessionID, &t.SequenceNo, &t.Title,
			&t.StartedAt, &t.EndedAt, &t.Status, &t.ElapsedMs, &t.AgentID,
			&t.Intent, &t.ConstraintsJSON, &t.DoneWhen,
		); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, rows.Err()
}

func loadEntries(d *db.DB) ([]Entry, error) {
	rows, err := d.Query(`
		SELECT id, session_id, turn_id, kind, title, detail, created_at, agent_id
		FROM entries ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var es []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.TurnID, &e.Kind,
			&e.Title, &e.Detail, &e.CreatedAt, &e.AgentID,
		); err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return es, rows.Err()
}

// SeqSum returns a change-detection counter: sum of row counts across key tables.
// When this value increases the caller should reload data from DB.
func SeqSum(d *db.DB) (int64, error) {
	var sum int64
	err := d.QueryRow(`
		SELECT (SELECT COUNT(*) FROM entries) +
		       (SELECT COUNT(*) FROM turns) +
		       (SELECT COUNT(*) FROM sessions)
	`).Scan(&sum)
	return sum, err
}
