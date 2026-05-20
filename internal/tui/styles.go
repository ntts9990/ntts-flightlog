// Package tui — styles.go: Lipgloss style definitions for the TUI chrome.
//
// Color numbers map 1:1 from v1 awk (bin/ntts-flightlog, render_markdown_ansi,
// lines ~309-410). The values here are the authoritative record; ansi.go in the
// views package carries the raw-byte equivalents used for content rendering.
//
// Why two systems?
//   - Content rendering (flat/turns/decisions/blockers views) uses raw ANSI in
//     internal/tui/views/ansi.go to stay byte-identical to v1 awk output.
//   - TUI chrome (menu header, status bar, B4 report) uses Lipgloss because
//     these elements have no v1 equivalent and don't need byte-equality.
//
// When Lipgloss renders a Bold + Color style it combines attributes into a
// single sequence (e.g. ESC[1;38;5;109m) which differs from v1's separate
// ESC[1m ESC[38;5;109m. Therefore Lipgloss is NOT used for content rendering.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// v1 awk color palette (ANSI 256-colour indices, verbatim from awk BEGIN block):
//
//	title_color    = 81   (bright cyan)
//	section_color  = 117  (cyan-blue)
//	mode_color     = 220  (gold)
//	entry_color    = 109  (light blue-gray, LightSkyBlue3)
//	decision_color = 215  (amber/orange)
//	evidence_color = 114  (green)
//	blocker_color  = 203  (red)
//	anchor_color   = 117  (cyan, A.5 TIA)
//
//	8-turn cycle: [207, 39, 213, 99, 198, 165, 75, 141]

// Styles holds all Lipgloss styles for the TUI chrome.
// Instantiate once via DefaultStyles().
type Styles struct {
	// Menu chrome
	Menu   lipgloss.Style // inactive menu items (section color 117)
	Active lipgloss.Style // active view label (title color 81, underlined)

	// Content category styles (reference; actual rendering uses views/ansi.go)
	Title    lipgloss.Style // # headings (81)
	Section  lipgloss.Style // ## headings (117)
	Mode     lipgloss.Style // [mode] entries (220)
	Entry    lipgloss.Style // [entry] entries (109)
	Decision lipgloss.Style // [decision] entries (215)
	Evidence lipgloss.Style // [evidence] entries (114)
	Blocker  lipgloss.Style // [blocker] entries (203)
	Anchor   lipgloss.Style // ⚓ TIA anchor block (117, A.5)
	Dim      lipgloss.Style // detail body lines

	// Turn cycle styles (8 colours, index 0-7, mirrors turnColors in ansi.go)
	Turns [8]lipgloss.Style
}

// DefaultStyles returns the canonical Lipgloss style set for the flightlog TUI.
func DefaultStyles() Styles {
	bold := func(color string) lipgloss.Style {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
	}

	s := Styles{
		Menu:     bold("117"),
		Active:   bold("81").Underline(true),
		Title:    bold("81"),
		Section:  bold("117"),
		Mode:     bold("220"),
		Entry:    bold("109"),
		Decision: bold("215"),
		Evidence: bold("114"),
		Blocker:  bold("203"),
		Anchor:   bold("117"),
		Dim:      lipgloss.NewStyle().Faint(true),
	}

	// 8-turn cycle (mirrors views.turnColors order).
	turnPalette := [8]string{"207", "39", "213", "99", "198", "165", "75", "141"}
	for i, c := range turnPalette {
		s.Turns[i] = bold(c)
	}

	return s
}

// CategoryStyle returns the Lipgloss style for the given entry kind.
// Mirrors views.WriteEntry colour selection.
// kind values: "entry" | "decision" | "evidence" | "blocker" | "mode"
func (s Styles) CategoryStyle(kind string) lipgloss.Style {
	switch kind {
	case "mode":
		return s.Mode
	case "entry":
		return s.Entry
	case "evidence":
		return s.Evidence
	case "decision":
		return s.Decision
	case "blocker":
		return s.Blocker
	default:
		return s.Entry
	}
}

// TurnStyle returns the Lipgloss style for the given 1-based turn number.
// Mirrors views.TurnColorFor palette selection.
func (s Styles) TurnStyle(turnNum int) lipgloss.Style {
	idx := (turnNum - 1) % len(s.Turns)
	if idx < 0 {
		idx = 0
	}
	return s.Turns[idx]
}
