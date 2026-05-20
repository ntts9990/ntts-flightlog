package views

import (
	"strings"
	"testing"
)

// These tests verify that our ANSI rendering is byte-identical to v1 awk output.
//
// Expected strings are derived directly from the v1 awk renderer source
// (bin/ntts-flightlog, render_markdown_ansi, lines ~309-410). They constitute
// the specification: if these strings change, the B-Exit byte-equality gate fails.
//
// Notation: \033 = ESC (0x1b). v1 awk uses separate BOLD+COLOR sequences, not
// combined (e.g. \033[1m\033[38;5;109m, NOT \033[1;38;5;109m). This is why
// content rendering uses raw ANSI constants rather than Lipgloss, which would
// produce combined sequences.

const testTS = "2026-05-20T10:00:00Z"
const testTitle = "test title"
const testTurnsDir = "/tmp/turns"

// v1AwkEntry builds the expected v1 awk output for a given kind.
// This mirrors the awk printf lines:
//
//	printf "%s%s<glyph> %s  %s%s\n", BOLD, color, ts, kind, RESET
//	printf "%s%s  %s%s\n", BOLD, color, title, RESET
func v1AwkEntry(glyph, color, ts, kind, title string) string {
	return Bold + color + glyph + " " + ts + "  [" + kind + "]" + Reset + "\n" +
		Bold + color + "  " + title + Reset + "\n"
}

// --------------------------------------------------------------------------
// WriteEntry byte-equality tests (category-fixed colors from v1 awk)
// --------------------------------------------------------------------------

func TestWriteEntry_ByteEq_Entry(t *testing.T) {
	var sb strings.Builder
	WriteEntry(&sb, testTS, "entry", testTitle)
	got := sb.String()
	want := v1AwkEntry("◆", ColorEntry, testTS, "entry", testTitle)
	if got != want {
		t.Errorf("entry byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteEntry_ByteEq_Decision(t *testing.T) {
	var sb strings.Builder
	WriteEntry(&sb, testTS, "decision", testTitle)
	got := sb.String()
	want := v1AwkEntry("◆", ColorDecision, testTS, "decision", testTitle)
	if got != want {
		t.Errorf("decision byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteEntry_ByteEq_Evidence(t *testing.T) {
	var sb strings.Builder
	WriteEntry(&sb, testTS, "evidence", testTitle)
	got := sb.String()
	want := v1AwkEntry("✓", ColorEvidence, testTS, "evidence", testTitle)
	if got != want {
		t.Errorf("evidence byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteEntry_ByteEq_Blocker(t *testing.T) {
	var sb strings.Builder
	WriteEntry(&sb, testTS, "blocker", testTitle)
	got := sb.String()
	// v1 uses "!!" glyph for blocker (2 chars + space)
	want := Bold + ColorBlocker + "!! " + testTS + "  [blocker]" + Reset + "\n" +
		Bold + ColorBlocker + "  " + testTitle + Reset + "\n"
	if got != want {
		t.Errorf("blocker byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteEntry_ByteEq_Mode(t *testing.T) {
	var sb strings.Builder
	WriteEntry(&sb, testTS, "mode", testTitle)
	got := sb.String()
	// v1 uses "▣" glyph for mode
	want := Bold + ColorMode + "▣ " + testTS + "  [mode]" + Reset + "\n" +
		Bold + ColorMode + "  " + testTitle + Reset + "\n"
	if got != want {
		t.Errorf("mode byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// --------------------------------------------------------------------------
// Turn-colour cycle: all 8 colours (v1 awk turn_colors[0..7])
// --------------------------------------------------------------------------

func TestTurnColorFor_AllEight(t *testing.T) {
	// Expected ANSI codes for turns 1-8, mirroring v1 awk turn_colors[] exactly.
	want := []string{
		Esc + "[38;5;207m", // turn 1
		Esc + "[38;5;39m",  // turn 2
		Esc + "[38;5;213m", // turn 3
		Esc + "[38;5;99m",  // turn 4
		Esc + "[38;5;198m", // turn 5
		Esc + "[38;5;165m", // turn 6
		Esc + "[38;5;75m",  // turn 7
		Esc + "[38;5;141m", // turn 8
	}
	for i, w := range want {
		n := i + 1 // 1-based turn number
		got := TurnColorFor(n)
		if got != w {
			t.Errorf("TurnColorFor(%d) = %q, want %q", n, got, w)
		}
	}
}

func TestTurnColorFor_Wraps(t *testing.T) {
	// Turn 9 wraps to index 0 (same as turn 1).
	if TurnColorFor(9) != TurnColorFor(1) {
		t.Errorf("TurnColorFor(9) should equal TurnColorFor(1) (8-cycle wrap)")
	}
}

// --------------------------------------------------------------------------
// WriteTurnStart byte-equality (v1 awk turn-start block)
// --------------------------------------------------------------------------

func TestWriteTurnStart_ByteEq_Turn1(t *testing.T) {
	var sb strings.Builder
	WriteTurnStart(&sb, testTS, 1, testTitle, testTurnsDir)
	got := sb.String()

	color := TurnColorFor(1)
	url := "file://" + testTurnsDir + "/turn-1.md"
	st := Esc + "\\"
	osc := Esc + "]8;;" + url + st + testTitle + Esc + "]8;;" + st

	// v1 awk turn-start block (4 lines):
	// print color "■■…■" RESET
	// printf "%s%s▶ %s  %s%s\n", BOLD, color, ts, "[turn-N-start]", RESET
	// printf "%s%s  %s%s\n", BOLD, color, osc_link(url, title), RESET
	// print color "────…────" RESET
	want := color + "■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■" + Reset + "\n" +
		Bold + color + "▶ " + testTS + "  [turn-1-start]" + Reset + "\n" +
		Bold + color + "  " + osc + Reset + "\n" +
		color + "────────────────────────────────" + Reset + "\n"

	if got != want {
		t.Errorf("turn-start-1 byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteTurnStart_ByteEq_Turn3(t *testing.T) {
	// Turn 3 uses colour index 2 (palette[2] = 213).
	var sb strings.Builder
	WriteTurnStart(&sb, testTS, 3, testTitle, testTurnsDir)
	got := sb.String()

	color := TurnColorFor(3) // ESC[38;5;213m
	if !strings.Contains(got, color) {
		t.Errorf("turn-start-3: expected color %q in output %q", color, got)
	}
	if !strings.Contains(got, "[turn-3-start]") {
		t.Errorf("turn-start-3: missing [turn-3-start] tag in output %q", got)
	}
}

// --------------------------------------------------------------------------
// WriteTurnEnd byte-equality (v1 awk turn-end block)
// --------------------------------------------------------------------------

func TestWriteTurnEnd_ByteEq_Turn1(t *testing.T) {
	summary := "작업 완료"
	var sb strings.Builder
	WriteTurnEnd(&sb, testTS, 1, summary)
	got := sb.String()

	color := TurnColorFor(1)

	// v1 awk turn-end block (3 lines):
	// print color "────…────" RESET
	// printf "%s%s■ %s  %s%s\n", BOLD, color, ts, "[turn-N-end]", RESET
	// printf "%s%s  %s%s\n", BOLD, color, title, RESET
	want := color + "────────────────────────────────" + Reset + "\n" +
		Bold + color + "■ " + testTS + "  [turn-1-end]" + Reset + "\n" +
		Bold + color + "  " + summary + Reset + "\n"

	if got != want {
		t.Errorf("turn-end-1 byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// --------------------------------------------------------------------------
// WriteDetail byte-equality
// --------------------------------------------------------------------------

func TestWriteDetail_ByteEq(t *testing.T) {
	var sb strings.Builder
	WriteDetail(&sb, "detail line")
	got := sb.String()
	want := Dim + "detail line" + Reset + "\n"
	if got != want {
		t.Errorf("detail byte mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteDetail_SkipsBlankLines(t *testing.T) {
	var sb strings.Builder
	WriteDetail(&sb, "\n\n   \nreal line\n\n")
	got := sb.String()
	// Only the non-blank line should appear.
	if !strings.Contains(got, "real line") {
		t.Errorf("WriteDetail: expected 'real line' in output: %q", got)
	}
	// Blank lines should be stripped.
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("WriteDetail: expected 1 output line, got %d: %q", len(lines), got)
	}
}

// --------------------------------------------------------------------------
// ANSI constant values (hard-coded so drift is caught immediately)
// --------------------------------------------------------------------------

func TestANSIConstants_ExactBytes(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Esc", Esc, "\033"},
		{"Reset", Reset, "\033[0m"},
		{"Bold", Bold, "\033[1m"},
		{"Dim", Dim, "\033[2m"},
		{"ColorTitle", ColorTitle, "\033[38;5;81m"},
		{"ColorSection", ColorSection, "\033[38;5;117m"},
		{"ColorMode", ColorMode, "\033[38;5;220m"},
		{"ColorEntry", ColorEntry, "\033[38;5;109m"},
		{"ColorDecision", ColorDecision, "\033[38;5;215m"},
		{"ColorEvidence", ColorEvidence, "\033[38;5;114m"},
		{"ColorBlocker", ColorBlocker, "\033[38;5;203m"},
		{"ColorAnchor", ColorAnchor, "\033[38;5;117m"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("constant %s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// --------------------------------------------------------------------------
// Integration test: compare v2 noninteractive against v1 awk binary.
// Requires the v1 binary at bin/ntts-flightlog and a migrate-able fixture.
// Skipped automatically if v1 binary is absent (CI-safe).
// Build tag: go test -tags integration ./internal/tui/views/
// --------------------------------------------------------------------------

// (Integration test lives in ansi_integration_test.go — build tag "integration")
