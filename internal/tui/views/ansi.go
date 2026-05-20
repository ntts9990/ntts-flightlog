package views

import (
	"fmt"
	"strings"
)

// ANSI escape sequences — identical to worklog/view.go (v1 awk port).
// These constants are the ground truth for byte-equality tests (B2 exit criterion).
// B2 will introduce Lipgloss styles in internal/tui/styles.go; these raw constants
// remain the reference implementation.
const (
	Esc   = "\033"
	Reset = Esc + "[0m"
	Bold  = Esc + "[1m"
	Dim   = Esc + "[2m"

	ColorTitle    = Esc + "[38;5;81m"
	ColorSection  = Esc + "[38;5;117m"
	ColorMode     = Esc + "[38;5;220m"
	ColorEntry    = Esc + "[38;5;109m"
	ColorDecision = Esc + "[38;5;215m"
	ColorEvidence = Esc + "[38;5;114m"
	ColorBlocker  = Esc + "[38;5;203m"
	ColorAnchor   = Esc + "[38;5;117m" // cyan — TIA anchor block (A.5)
)

// turnColors is the 8-color cycle for turn headers (mirrors v1 awk turn_colors[]).
var turnColors = []string{
	Esc + "[38;5;207m",
	Esc + "[38;5;39m",
	Esc + "[38;5;213m",
	Esc + "[38;5;99m",
	Esc + "[38;5;198m",
	Esc + "[38;5;165m",
	Esc + "[38;5;75m",
	Esc + "[38;5;141m",
}

// TurnColorFor returns the ANSI color string for the 1-based turn number n.
// Mirrors v1 awk turn_color_for(n): idx = (n-1) % 8.
func TurnColorFor(n int) string {
	idx := (n - 1) % len(turnColors)
	if idx < 0 {
		idx = 0
	}
	return turnColors[idx]
}

// osc8Link wraps text in an OSC 8 hyperlink (file:// URL).
// Mirrors v1 awk osc_link(url, text).
func osc8Link(url, text string) string {
	st := Esc + "\\"
	return Esc + "]8;;" + url + st + text + Esc + "]8;;" + st
}

// WriteTurnStart writes a turn-start header block to sb.
// Mirrors v1 awk output for lines matching /\[turn-N-start\]/.
func WriteTurnStart(sb *strings.Builder, ts string, seqNo int, title, turnsDir string) {
	color := TurnColorFor(seqNo)
	kind := fmt.Sprintf("turn-%d-start", seqNo)
	url := fmt.Sprintf("file://%s/turn-%d.md", turnsDir, seqNo)
	sb.WriteString(color + "■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■" + Reset + "\n")
	fmt.Fprintf(sb, "%s%s▶ %s  [%s]%s\n", Bold, color, ts, kind, Reset)
	fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, color, osc8Link(url, title), Reset)
	sb.WriteString(color + "────────────────────────────────" + Reset + "\n")
}

// WriteTurnEnd writes a turn-end footer block to sb.
// Mirrors v1 awk output for lines matching /\[turn-N-end\]/.
func WriteTurnEnd(sb *strings.Builder, ts string, seqNo int, summary string) {
	color := TurnColorFor(seqNo)
	kind := fmt.Sprintf("turn-%d-end", seqNo)
	sb.WriteString(color + "────────────────────────────────" + Reset + "\n")
	fmt.Fprintf(sb, "%s%s■ %s  [%s]%s\n", Bold, color, ts, kind, Reset)
	fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, color, summary, Reset)
}

// WriteEntry writes a single entry header (kind + title) to sb.
// Mirrors v1 awk per-kind rendering in render_markdown_ansi.
func WriteEntry(sb *strings.Builder, ts, kind, title string) {
	switch kind {
	case "mode":
		fmt.Fprintf(sb, "%s%s▣ %s  [%s]%s\n", Bold, ColorMode, ts, kind, Reset)
		fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorMode, title, Reset)
	case "entry":
		fmt.Fprintf(sb, "%s%s◆ %s  [%s]%s\n", Bold, ColorEntry, ts, kind, Reset)
		fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorEntry, title, Reset)
	case "evidence":
		fmt.Fprintf(sb, "%s%s✓ %s  [%s]%s\n", Bold, ColorEvidence, ts, kind, Reset)
		fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorEvidence, title, Reset)
	case "decision":
		fmt.Fprintf(sb, "%s%s◆ %s  [%s]%s\n", Bold, ColorDecision, ts, kind, Reset)
		fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorDecision, title, Reset)
	case "blocker":
		fmt.Fprintf(sb, "%s%s!! %s  [%s]%s\n", Bold, ColorBlocker, ts, kind, Reset)
		fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorBlocker, title, Reset)
	default:
		fmt.Fprintf(sb, "%s◆ %s  [%s]%s\n", Bold, ts, kind, Reset)
		fmt.Fprintf(sb, "  %s\n", title)
	}
}

// WriteDetail writes a detail/body block to sb in dim style.
func WriteDetail(sb *strings.Builder, detail string) {
	if detail == "" {
		return
	}
	for _, line := range strings.Split(detail, "\n") {
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(sb, "%s%s%s\n", Dim, line, Reset)
		}
	}
}

// WriteTurnAnchor writes a TIA anchor block for a turn with non-NULL intent.
// Mirrors A.5.5 worklog/view.go anchor rendering.
func WriteTurnAnchor(sb *strings.Builder, t Turn) {
	if !t.Intent.Valid || t.Intent.String == "" {
		return
	}
	fmt.Fprintf(sb, "%s%s⚓ 의도: %s%s\n", Bold, ColorAnchor, t.Intent.String, Reset)
	if t.ConstraintsJSON.Valid && t.ConstraintsJSON.String != "" {
		fmt.Fprintf(sb, "%s%s📐 제약: %s%s\n", Bold, ColorAnchor, t.ConstraintsJSON.String, Reset)
	}
	if t.DoneWhen.Valid && t.DoneWhen.String != "" {
		fmt.Fprintf(sb, "%s%s✅ 완료조건: %s%s%s\n", Bold, ColorAnchor, t.DoneWhen.String, Reset, "")
	}
}
