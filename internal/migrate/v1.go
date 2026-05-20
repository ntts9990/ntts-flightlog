// Package migrate implements v1 → v2 migration for ntts-flightlog.
//
// v1 stores data as a human-readable .ntts-flightlog/main.md file with
// per-turn turn-N.md files. v2 stores data in SQLite. This package parses
// v1 format, imports into SQLite, and provides a canonical md formatter for
// round-trip testing.
//
// "Lossless" is defined by 7 enumerated equality predicates (plan A5):
//  1. Entry count equality
//  2. Timestamp byte-equality (ISO 8601 verbatim)
//  3. Kind preserved
//  4. Title UTF-8 NFC byte-equality
//  5. Detail multi-line body byte-equality
//  6. OSC 8 URL payload byte-equality
//  7. Ordering preserved (rowid = source order; sequence_no monotonic)
package migrate

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// --- types ---

// Record is one ### heading parsed from a v1 main.md (or turn-N.md) file.
type Record struct {
	Seq       int    // 0-based position within file
	Timestamp string // ISO 8601 verbatim, e.g. "2026-05-20T03:24:31Z"
	KindRaw   string // raw bracket content, e.g. "turn-1-start", "entry"
	Kind      string // normalized: entry|decision|evidence|blocker|mode|turn-start|turn-end
	TurnNum   int    // N for turn-N-start/end; 0 for non-turn records
	Title     string // text after [kind] on heading line (verbatim, OSC 8 preserved)
	Detail    string // multi-line body; leading blank stripped, trailing blank stripped
}

// V1Data holds everything parsed from a v1 .ntts-flightlog/ directory.
type V1Data struct {
	SessionStartEpoch int64
	Mode              string
	TurnCount         int
	Records           []Record // all log records in source order
}

// EntryRow holds one row from the v2 entries table.
type EntryRow struct {
	ID        string
	SessionID string
	TurnID    string // empty string if NULL
	Kind      string
	Title     string
	Detail    string
	CreatedAt string
	AgentID   string // empty string if NULL
}

// TurnRow holds one row from the v2 turns table.
type TurnRow struct {
	ID         string
	SessionID  string
	SequenceNo int
	Title      string
	StartedAt  string
	EndedAt    string // empty string if NULL
	Status     string
	ElapsedMS  int64 // 0 if NULL
}

// --- parsing ---

var headingRE = regexp.MustCompile(`^### (\S+) \[([^\]]+)\] (.*)$`)
var turnKindRE = regexp.MustCompile(`^turn-(\d+)-(start|end)$`)

// ParseMainMD parses a v1 main.md reader and returns Records in file order.
// If the file contains a "## 작업 기록" section header, only records within
// that section are returned. Otherwise (e.g., for bare FormatEntries output),
// all matching ### headings are returned.
func ParseMainMD(r io.Reader) ([]Record, error) {
	scanner := bufio.NewScanner(r)

	var records []Record
	hasSectionHeader := false
	inLogSection := false
	var current *Record
	var detailLines []string

	// First pass: detect whether a section header exists.
	// We do a single-pass approach and handle both modes inline.
	for scanner.Scan() {
		line := scanner.Text()

		if line == "## 작업 기록" {
			hasSectionHeader = true
			inLogSection = true
			continue
		}
		if hasSectionHeader && strings.HasPrefix(line, "## ") && line != "## 작업 기록" {
			inLogSection = false
		}

		active := !hasSectionHeader || inLogSection

		if m := headingRE.FindStringSubmatch(line); m != nil && active {
			if current != nil {
				current.Detail = trimDetail(strings.Join(detailLines, "\n"))
				records = append(records, *current)
			}
			ts, kindRaw, title := m[1], m[2], m[3]
			kind, turnNum := normalizeKind(kindRaw)
			current = &Record{
				Seq:       len(records),
				Timestamp: ts,
				KindRaw:   kindRaw,
				Kind:      kind,
				TurnNum:   turnNum,
				Title:     title,
			}
			detailLines = nil
			continue
		}

		if current != nil && active {
			detailLines = append(detailLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	if current != nil {
		current.Detail = trimDetail(strings.Join(detailLines, "\n"))
		records = append(records, *current)
	}
	return records, nil
}

// normalizeKind maps raw bracket content to canonical kind + turn number.
func normalizeKind(raw string) (kind string, turnNum int) {
	if m := turnKindRE.FindStringSubmatch(raw); m != nil {
		n, _ := strconv.Atoi(m[1])
		if m[2] == "start" {
			return "turn-start", n
		}
		return "turn-end", n
	}
	switch raw {
	case "entry", "decision", "evidence", "blocker", "mode":
		return raw, 0
	default:
		return "entry", 0 // unknown kinds fall back to entry
	}
}

// trimDetail strips leading blank lines and trailing blank lines from a raw
// detail body (joined lines). Internal content is preserved verbatim.
func trimDetail(s string) string {
	lines := strings.Split(s, "\n")
	// Strip leading blank lines
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	// Strip trailing blank lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// ParseDir reads a v1 .ntts-flightlog/ directory and returns V1Data.
// pane-id is intentionally ignored (volatile; meaningless after session ends).
func ParseDir(dir string) (*V1Data, error) {
	data := &V1Data{}

	if b, err := os.ReadFile(filepath.Join(dir, "session-start-epoch")); err == nil {
		ep, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse session-start-epoch: %w", err)
		}
		data.SessionStartEpoch = ep
	}

	if b, err := os.ReadFile(filepath.Join(dir, "mode")); err == nil {
		data.Mode = strings.TrimSpace(string(b))
	}
	if data.Mode == "" {
		data.Mode = "solo"
	}

	if b, err := os.ReadFile(filepath.Join(dir, "turn-counter")); err == nil {
		tc, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			return nil, fmt.Errorf("parse turn-counter: %w", err)
		}
		data.TurnCount = tc
	}

	f, err := os.Open(filepath.Join(dir, "main.md"))
	if err != nil {
		return nil, fmt.Errorf("open main.md: %w", err)
	}
	defer f.Close()

	records, err := ParseMainMD(f)
	if err != nil {
		return nil, fmt.Errorf("parse main.md: %w", err)
	}
	data.Records = records
	return data, nil
}

// --- import ---

// ImportToDB inserts V1Data into an open v2 SQLite DB.
// Returns the new session ID. agent_id is NULL (best-effort: v1 had no agent tracking).
// NFC normalization is applied to all title fields before storage (predicate 4).
func ImportToDB(d *db.DB, data *V1Data) (string, error) {
	sessID := newID()
	startedAt := "1970-01-01T00:00:00Z"
	if data.SessionStartEpoch > 0 {
		startedAt = time.Unix(data.SessionStartEpoch, 0).UTC().Format(time.RFC3339)
	}

	if _, err := d.Exec(
		`INSERT INTO sessions (id, started_at, mode) VALUES (?, ?, ?)`,
		sessID, startedAt, data.Mode,
	); err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}

	// currentTurnID tracks the open turn as we scan records in order.
	var currentTurnID string
	turnIDs := map[int]string{} // turn number → DB id

	for i, rec := range data.Records {
		switch rec.Kind {
		case "turn-start":
			turnID := newID()
			turnIDs[rec.TurnNum] = turnID
			currentTurnID = turnID
			if _, err := d.Exec(
				`INSERT INTO turns (id, session_id, sequence_no, title, started_at, status)
				 VALUES (?, ?, ?, ?, ?, 'active')`,
				turnID, sessID, rec.TurnNum, nfc(rec.Title), rec.Timestamp,
			); err != nil {
				return "", fmt.Errorf("insert turn[%d]: %w", i, err)
			}

		case "turn-end":
			turnID, ok := turnIDs[rec.TurnNum]
			if !ok {
				continue // no matching turn-start; skip gracefully
			}
			elapsed := parseElapsedMS(rec.Detail)
			if _, err := d.Exec(
				`UPDATE turns SET ended_at=?, status='complete', elapsed_ms=? WHERE id=?`,
				rec.Timestamp, elapsed, turnID,
			); err != nil {
				return "", fmt.Errorf("update turn-end[%d]: %w", i, err)
			}
			currentTurnID = ""

		default:
			kind := rec.Kind
			if !isEntryKind(kind) {
				kind = "entry"
			}
			var turnIDArg interface{}
			if currentTurnID != "" {
				turnIDArg = currentTurnID
			}
			if _, err := d.Exec(
				`INSERT INTO entries (id, session_id, turn_id, kind, title, detail, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				newID(), sessID, turnIDArg, kind, nfc(rec.Title), rec.Detail, rec.Timestamp,
			); err != nil {
				return "", fmt.Errorf("insert entry[%d]: %w", i, err)
			}
		}
	}

	return sessID, nil
}

// --- export ---

// QueryEntries returns all entries for a session in insertion order (rowid).
// Insertion order equals source order per ImportToDB contract.
func QueryEntries(d *db.DB, sessionID string) ([]EntryRow, error) {
	rows, err := d.Query(
		`SELECT id, session_id, COALESCE(turn_id,''), kind, title, COALESCE(detail,''), created_at, COALESCE(agent_id,'')
		 FROM entries WHERE session_id=? ORDER BY rowid`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var out []EntryRow
	for rows.Next() {
		var e EntryRow
		if err := rows.Scan(&e.ID, &e.SessionID, &e.TurnID, &e.Kind, &e.Title, &e.Detail, &e.CreatedAt, &e.AgentID); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// QueryTurns returns all turns for a session in sequence_no order.
func QueryTurns(d *db.DB, sessionID string) ([]TurnRow, error) {
	rows, err := d.Query(
		`SELECT id, session_id, sequence_no, COALESCE(title,''), started_at, COALESCE(ended_at,''), status, COALESCE(elapsed_ms,0)
		 FROM turns WHERE session_id=? ORDER BY sequence_no`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query turns: %w", err)
	}
	defer rows.Close()

	var out []TurnRow
	for rows.Next() {
		var t TurnRow
		if err := rows.Scan(&t.ID, &t.SessionID, &t.SequenceNo, &t.Title, &t.StartedAt, &t.EndedAt, &t.Status, &t.ElapsedMS); err != nil {
			return nil, fmt.Errorf("scan turn: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// FormatEntries serializes EntryRows to the canonical ### main.md format.
// This is the canonical formatter shared with the A4 main.md mirror.
// Titles are output verbatim (already NFC-normalized in DB).
// Detail bodies are output verbatim (newlines preserved).
func FormatEntries(entries []EntryRow) string {
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString("### ")
		sb.WriteString(e.CreatedAt)
		sb.WriteString(" [")
		sb.WriteString(e.Kind)
		sb.WriteString("] ")
		sb.WriteString(e.Title)
		sb.WriteString("\n")
		if e.Detail != "" {
			sb.WriteString(e.Detail)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// --- helpers ---

// nfc applies Unicode NFC normalization to a string.
// Addresses macOS HFS+ NFD risk (plan A5 predicate 4).
func nfc(s string) string {
	return norm.NFC.String(s)
}

// isEntryKind returns true for the 5 valid entry kinds.
func isEntryKind(kind string) bool {
	switch kind {
	case "entry", "decision", "evidence", "blocker", "mode":
		return true
	}
	return false
}

// parseElapsedMS extracts elapsed milliseconds from a turn-end detail line.
// Format: "소요 시간: 4s." or "소요 시간: 1m 37s." etc.
// Returns 0 if not parseable.
func parseElapsedMS(detail string) int64 {
	lines := strings.Split(detail, "\n")
	for _, line := range lines {
		if strings.Contains(line, "소요 시간:") || strings.Contains(line, "elapsed:") {
			// Extract duration: look for patterns like "4s", "1m 37s", "unknown"
			parts := strings.Fields(line)
			for j, p := range parts {
				if strings.HasSuffix(p, "s.") || strings.HasSuffix(p, "s") {
					// Check if there's a "Nm" before it
					var minutes, seconds int
					if j > 0 && strings.HasSuffix(parts[j-1], "m") {
						m, _ := strconv.Atoi(strings.TrimSuffix(parts[j-1], "m"))
						minutes = m
					}
					secs := strings.TrimRight(p, "s.")
					s, _ := strconv.Atoi(secs)
					seconds = s
					return int64((minutes*60+seconds) * 1000)
				}
			}
		}
	}
	return 0
}

// newID generates a random 16-byte hex ID for DB primary keys using crypto/rand.
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read should never fail on any supported OS (plan targets:
		// darwin/linux/windows). If it does, panic is appropriate — silent
		// bad IDs would cause silent data corruption.
		panic(fmt.Sprintf("migrate.newID: crypto/rand.Read: %v", err))
	}
	return hex.EncodeToString(b)
}
