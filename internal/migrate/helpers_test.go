package migrate

// helpers_test.go: white-box tests for unexported helper functions.
// Package is `migrate` (not `migrate_test`) to access unexported symbols.

import (
	"testing"
)

// TestIsEntryKind exhaustively exercises all branches of isEntryKind.
func TestIsEntryKind(t *testing.T) {
	valid := []string{"entry", "decision", "evidence", "blocker", "mode"}
	for _, kind := range valid {
		if !isEntryKind(kind) {
			t.Errorf("isEntryKind(%q) = false, want true", kind)
		}
	}
	invalid := []string{"", "unknown", "turn-start", "turn-end", "other", "ENTRY"}
	for _, kind := range invalid {
		if isEntryKind(kind) {
			t.Errorf("isEntryKind(%q) = true, want false", kind)
		}
	}
}

// TestParseElapsedMS exercises all branches of the v1-format elapsed-ms parser.
func TestParseElapsedMS(t *testing.T) {
	tests := []struct {
		detail string
		want   int64
	}{
		{"소요 시간: 4s.", 4000},
		{"소요 시간: 1m 37s.", 97000},
		{"elapsed: 30s", 30000},
		{"소요 시간: 2m 0s.", 120000},
		{"소요 시간: 0s.", 0},
		{"소요 시간: unknown", 0},
		{"", 0},
		{"no match here", 0},
		// multi-line detail: parser scans all lines
		{"first line\n소요 시간: 5s.\nthird line", 5000},
	}
	for _, tt := range tests {
		got := parseElapsedMS(tt.detail)
		if got != tt.want {
			t.Errorf("parseElapsedMS(%q) = %d, want %d", tt.detail, got, tt.want)
		}
	}
}

// TestNormalizeKind verifies normalizeKind maps raw bracket content correctly.
func TestNormalizeKind(t *testing.T) {
	tests := []struct {
		raw      string
		wantKind string
		wantTurn int
	}{
		{"entry", "entry", 0},
		{"decision", "decision", 0},
		{"evidence", "evidence", 0},
		{"blocker", "blocker", 0},
		{"mode", "mode", 0},
		{"turn-1-start", "turn-start", 1},
		{"turn-3-end", "turn-end", 3},
		{"turn-10-start", "turn-start", 10},
		// unknown kinds fall back to "entry" per plan A5
		{"unknown-kind", "entry", 0},
		{"", "entry", 0},
		{"custom", "entry", 0},
	}
	for _, tt := range tests {
		gotKind, gotTurn := normalizeKind(tt.raw)
		if gotKind != tt.wantKind || gotTurn != tt.wantTurn {
			t.Errorf("normalizeKind(%q) = (%q, %d), want (%q, %d)",
				tt.raw, gotKind, gotTurn, tt.wantKind, tt.wantTurn)
		}
	}
}

// TestTrimDetail verifies that trimDetail strips leading/trailing blank lines
// while preserving internal blank lines and content verbatim.
func TestTrimDetail(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"content", "content"},
		{"line1\n\nline2", "line1\n\nline2"},
		{"just one line", "just one line"},
	}
	for _, tt := range tests {
		got := trimDetail(tt.input)
		if got != tt.want {
			t.Errorf("trimDetail(%q):\n  got  %q\n  want %q", tt.input, got, tt.want)
		}
	}
}

// TestNewID verifies newID returns a 32-character lowercase hex string.
func TestNewID(t *testing.T) {
	id := newID()
	if len(id) != 32 {
		t.Errorf("newID() length = %d, want 32", len(id))
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("newID() contains non-hex character %q in %q", c, id)
			break
		}
	}
	// Two calls must return distinct IDs.
	id2 := newID()
	if id == id2 {
		t.Error("newID() returned the same value twice")
	}
}
