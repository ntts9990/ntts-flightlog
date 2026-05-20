package worklog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ANSI escape sequences (mirror v1 awk color values exactly).
const (
	esc   = "\033"
	reset = esc + "[0m"
	bold  = esc + "[1m"
	dim   = esc + "[2m"

	colorTitle    = esc + "[38;5;81m"
	colorSection  = esc + "[38;5;117m"
	colorMode     = esc + "[38;5;220m"
	colorEntry    = esc + "[38;5;109m"
	colorDecision = esc + "[38;5;215m"
	colorEvidence = esc + "[38;5;114m"
	colorBlocker  = esc + "[38;5;203m"
	colorAnchor   = esc + "[38;5;117m" // cyan — TIA anchor block (A.5)
)

// turnColors mirrors v1's 8-color cycle for turn headers.
var turnColors = []string{
	esc + "[38;5;207m",
	esc + "[38;5;39m",
	esc + "[38;5;213m",
	esc + "[38;5;99m",
	esc + "[38;5;198m",
	esc + "[38;5;165m",
	esc + "[38;5;75m",
	esc + "[38;5;141m",
}

func turnColorFor(n int) string {
	idx := (n - 1) % len(turnColors)
	if idx < 0 {
		idx = 0
	}
	return turnColors[idx]
}

// osc8Link wraps text in an OSC 8 hyperlink.
func osc8Link(url, text string) string {
	st := esc + "\\"
	return esc + "]8;;" + url + st + text + esc + "]8;;" + st
}

// entryHeaderRe matches `### TS [kind] title` lines.
var entryHeaderRe = regexp.MustCompile(`^\#\#\# (\S+) \[([^\]]+)\] (.+)$`)

// turnStartRe matches `[turn-N-start]` kind strings.
var turnStartRe = regexp.MustCompile(`^turn-(\d+)-start$`)

// turnEndRe matches `[turn-N-end]` kind strings.
var turnEndRe = regexp.MustCompile(`^turn-(\d+)-end$`)

// RenderFlat renders main.md with ANSI color (v1 `render_markdown_ansi` port).
func RenderFlat(c *Config, w io.Writer) error {
	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(w, "(main.md 파일이 없습니다. flightlog start로 시작하세요.)")
			return nil
		}
		return err
	}
	absTurns, _ := filepath.Abs(c.TurnsDir)
	renderMarkdownANSI(string(data), absTurns, w)
	return nil
}

// RenderTurns renders each per-turn markdown file sequentially.
func RenderTurns(c *Config, w io.Writer) error {
	entries, err := filepath.Glob(filepath.Join(c.TurnsDir, "turn-*.md"))
	if err != nil || len(entries) == 0 {
		fmt.Fprintln(w, "(turn 파일이 아직 없습니다. turn-start로 첫 턴을 시작하세요.)")
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return turnSortKey(entries[i]) < turnSortKey(entries[j])
	})
	absTurns, _ := filepath.Abs(c.TurnsDir)
	for _, f := range entries {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		renderMarkdownANSI(string(data), absTurns, w)
		fmt.Fprintln(w)
	}
	return nil
}

// turnSortKey extracts the numeric turn number for natural sort.
func turnSortKey(path string) int {
	base := filepath.Base(path)
	base = strings.TrimPrefix(base, "turn-")
	base = strings.TrimSuffix(base, ".md")
	n, _ := strconv.Atoi(base)
	return n
}

// FilterByKind renders only entries matching kind (decisions or blockers).
func FilterByKind(c *Config, kind string, w io.Writer) error {
	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	filterEntriesByKind(string(data), kind, w)
	return nil
}

// renderMarkdownANSI is the Go port of v1's awk render_markdown_ansi function.
func renderMarkdownANSI(content, absTurns string, w io.Writer) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# "):
			fmt.Fprintln(w, bold+colorTitle+line+reset)
		case strings.HasPrefix(line, "## "):
			fmt.Fprintln(w)
			fmt.Fprintln(w, bold+colorSection+line+reset)
		case strings.HasPrefix(line, "업데이트:"), strings.HasPrefix(line, "시작:"):
			fmt.Fprintln(w, dim+line+reset)
		// A.5 TIA anchor lines — render in cyan.
		case strings.HasPrefix(line, "⚓ 의도:"),
			strings.HasPrefix(line, "📐 제약:"),
			strings.HasPrefix(line, "✅ 완료조건:"),
			strings.HasPrefix(line, "─── ⚓"):
			fmt.Fprintln(w, bold+colorAnchor+line+reset)
		case entryHeaderRe.MatchString(line):
			m := entryHeaderRe.FindStringSubmatch(line)
			ts, kind, title := m[1], m[2], m[3]
			renderEntryHeader(w, ts, kind, title, absTurns)
		case len(line) > 0 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t"):
			fmt.Fprintln(w, dim+line+reset)
		default:
			fmt.Fprintln(w, line)
		}
	}
}

// renderEntryHeader renders a `### TS [kind] title` line with appropriate color/symbol.
func renderEntryHeader(w io.Writer, ts, kind, title, absTurns string) {
	if m := turnStartRe.FindStringSubmatch(kind); m != nil {
		n, _ := strconv.Atoi(m[1])
		color := turnColorFor(n)
		url := "file://" + absTurns + "/turn-" + m[1] + ".md"
		fmt.Fprintln(w, color+"■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■"+reset)
		fmt.Fprintf(w, "%s%s▶ %s  [%s]%s\n", bold, color, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, color, osc8Link(url, title), reset)
		fmt.Fprintln(w, color+"────────────────────────────────"+reset)
		return
	}
	if m := turnEndRe.FindStringSubmatch(kind); m != nil {
		n, _ := strconv.Atoi(m[1])
		color := turnColorFor(n)
		fmt.Fprintln(w, color+"────────────────────────────────"+reset)
		fmt.Fprintf(w, "%s%s■ %s  [%s]%s\n", bold, color, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, color, title, reset)
		return
	}
	switch kind {
	case "mode":
		fmt.Fprintf(w, "%s%s▣ %s  [%s]%s\n", bold, colorMode, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorMode, title, reset)
	case "entry":
		fmt.Fprintf(w, "%s%s◆ %s  [%s]%s\n", bold, colorEntry, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorEntry, title, reset)
	case "evidence":
		fmt.Fprintf(w, "%s%s✓ %s  [%s]%s\n", bold, colorEvidence, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorEvidence, title, reset)
	case "decision":
		fmt.Fprintf(w, "%s%s◆ %s  [%s]%s\n", bold, colorDecision, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorDecision, title, reset)
	case "blocker":
		fmt.Fprintf(w, "%s%s!! %s  [%s]%s\n", bold, colorBlocker, ts, kind, reset)
		fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorBlocker, title, reset)
	default:
		fmt.Fprintf(w, "%s◆ %s  [%s]%s\n", bold, ts, kind, reset)
		fmt.Fprintf(w, "  %s\n", title)
	}
}

// filterEntriesByKind is the Go port of v1's awk filter_entries_by_kind.
func filterEntriesByKind(content, kind string, w io.Writer) {
	lines := strings.Split(content, "\n")
	printing := false
	pat := "[" + kind + "]"
	for _, line := range lines {
		if entryHeaderRe.MatchString(line) {
			if strings.Contains(line, pat) {
				m := entryHeaderRe.FindStringSubmatch(line)
				ts, title := m[1], m[3]
				if kind == "blocker" {
					fmt.Fprintf(w, "%s%s!! %s%s\n", bold, colorBlocker, ts, reset)
					fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorBlocker, title, reset)
				} else {
					fmt.Fprintf(w, "%s%s◆ %s%s\n", bold, colorDecision, ts, reset)
					fmt.Fprintf(w, "%s%s  %s%s\n", bold, colorDecision, title, reset)
				}
				printing = true
			} else {
				printing = false
			}
			continue
		}
		if printing && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "### ") {
			fmt.Fprintln(w, dim+line+reset)
		}
	}
}
