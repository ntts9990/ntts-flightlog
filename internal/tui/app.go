// Package tui implements the Bubble Tea interactive viewer for ntts-flightlog v2.
//
// Architecture:
//   - Model holds a bubbles/viewport for scrollable ANSI content.
//   - 5 views: flat (1), turns (2), decisions (3), blockers (4), report (5).
//   - Menu header pins to the top; viewport fills remaining terminal height.
//   - Data source: SQLite via internal/db — NOT main.md file mtime polling (plan B1).
//   - Change detection: 2-second tick polls SeqSum (entry+turn+session row counts).
//
// Run() starts the Bubble Tea program with alt-screen.
// RenderView() renders a single view to a string for noninteractive / test use.
package tui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/tui/views"
)

// viewID identifies one of the 5 TUI views.
type viewID int

const (
	viewFlat      viewID = iota + 1 // 1 = flat worklog
	viewTurns                        // 2 = grouped by turn
	viewDecisions                    // 3 = decisions only
	viewBlockers                     // 4 = blockers only
	viewReport                       // 5 = metrics report (B4 placeholder)
)

// headerHeight is the number of lines consumed by the pinned menu (line + blank).
const headerHeight = 2

// menu/active styles for the pinned header bar.
var (
	menuStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117"))

	activeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81")).
			Underline(true)
)

// --------------------------------------------------------------------------
// Messages
// --------------------------------------------------------------------------

// tickMsg fires every 2 s to trigger change detection.
type tickMsg time.Time

// loadedMsg carries freshly loaded worklog data from the DB.
type loadedMsg struct {
	data   *views.WorklogData
	seqSum int64
}

// errMsg carries an error from a background DB operation.
type errMsg struct{ err error }

// --------------------------------------------------------------------------
// Model
// --------------------------------------------------------------------------

// Model is the Bubble Tea application model for the flightlog TUI.
type Model struct {
	activeView viewID
	data       *views.WorklogData
	vp         viewport.Model
	ready      bool
	d          *db.DB
	seqSum     int64
	width      int
	height     int
	turnsDir   string // absolute path, used for OSC 8 file:// links
	err        error
}

// New creates a new TUI Model. The DB must already be open.
func New(d *db.DB, turnsDir string) Model {
	absTurns, _ := filepath.Abs(turnsDir)
	return Model{
		activeView: viewFlat,
		d:          d,
		turnsDir:   absTurns,
	}
}

// --------------------------------------------------------------------------
// Bubble Tea interface
// --------------------------------------------------------------------------

// Init fires the initial data load and starts the 2-second tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(cmdLoad(m.d), cmdTick())
}

// Update processes all incoming messages and returns the updated model + next Cmd.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	// ---- Key events --------------------------------------------------------
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m = m.switchView(viewFlat)
		case "2":
			m = m.switchView(viewTurns)
		case "3":
			m = m.switchView(viewDecisions)
		case "4":
			m = m.switchView(viewBlockers)
		case "5":
			m = m.switchView(viewReport)
		case "r":
			cmds = append(cmds, cmdLoad(m.d))
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	// ---- Window resize -----------------------------------------------------
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := msg.Height - headerHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.vp = viewport.New(msg.Width, vpHeight)
			m.vp.SetContent(m.currentContent())
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = vpHeight
		}

	// ---- Data loaded -------------------------------------------------------
	case loadedMsg:
		m.data = msg.data
		m.seqSum = msg.seqSum
		m.err = nil
		if m.ready {
			m.vp.SetContent(m.currentContent())
		}

	// ---- DB error ----------------------------------------------------------
	case errMsg:
		m.err = msg.err

	// ---- Periodic tick: check for DB changes --------------------------------
	case tickMsg:
		cmds = append(cmds, cmdCheckChanges(m.d, m.seqSum), cmdTick())
	}

	// Propagate all events to the viewport for scroll handling.
	if m.ready {
		m.vp, vpCmd = m.vp.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the full screen: pinned menu header + scrollable content.
func (m Model) View() string {
	if !m.ready {
		return "로딩 중...\n"
	}
	if m.err != nil {
		return fmt.Sprintf("%s오류: %v%s\n", views.Bold+views.ColorBlocker, m.err, views.Reset)
	}
	return m.menuHeader() + "\n" + m.vp.View()
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// switchView changes the active view and resets the viewport content.
func (m Model) switchView(v viewID) Model {
	m.activeView = v
	if m.ready {
		m.vp.SetContent(m.currentContent())
		m.vp.GotoTop()
	}
	return m
}

// menuHeader returns the single pinned menu line.
// The active view label is underlined; others use the section color.
func (m Model) menuHeader() string {
	type label struct {
		key   string
		name  string
		view  viewID
	}
	items := []label{
		{"1", "flat", viewFlat},
		{"2", "turns", viewTurns},
		{"3", "decisions", viewDecisions},
		{"4", "blockers", viewBlockers},
		{"5", "report", viewReport},
	}

	header := ""
	sep := menuStyle.Render(" / ")
	for i, l := range items {
		if i > 0 {
			header += sep
		}
		text := l.key + "=" + l.name
		if m.activeView == l.view {
			header += activeStyle.Render(text)
		} else {
			header += menuStyle.Render(text)
		}
	}
	header += sep + menuStyle.Render("r=reload") + sep + menuStyle.Render("q=quit")
	return header
}

// currentContent returns the rendered ANSI string for the active view.
func (m Model) currentContent() string {
	switch m.activeView {
	case viewFlat:
		return views.RenderFlat(m.data, m.turnsDir)
	case viewTurns:
		return views.RenderTurns(m.data, m.turnsDir)
	case viewDecisions:
		return views.RenderDecisions(m.data)
	case viewBlockers:
		return views.RenderBlockers(m.data)
	case viewReport:
		return views.RenderReport(m.data)
	default:
		return views.RenderFlat(m.data, m.turnsDir)
	}
}

// --------------------------------------------------------------------------
// Commands
// --------------------------------------------------------------------------

// cmdLoad returns a Cmd that queries all worklog data from the DB.
func cmdLoad(d *db.DB) tea.Cmd {
	return func() tea.Msg {
		data, err := views.LoadAll(d)
		if err != nil {
			return errMsg{err: err}
		}
		seq, err := views.SeqSum(d)
		if err != nil {
			return errMsg{err: err}
		}
		return loadedMsg{data: data, seqSum: seq}
	}
}

// cmdCheckChanges returns a Cmd that triggers a full reload only when SeqSum changed.
func cmdCheckChanges(d *db.DB, lastSeq int64) tea.Cmd {
	return func() tea.Msg {
		seq, err := views.SeqSum(d)
		if err != nil || seq == lastSeq {
			return nil // no change; next tick will retry
		}
		return cmdLoad(d)() // data changed — reload inline
	}
}

// cmdTick returns a Cmd that fires a tickMsg after 2 seconds.
func cmdTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --------------------------------------------------------------------------
// Public API (used by internal/cli/view.go)
// --------------------------------------------------------------------------

// Run starts the Bubble Tea TUI with the alternate screen buffer.
func Run(d *db.DB, turnsDir string) error {
	m := New(d, turnsDir)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// RenderView renders a single named view to a string without starting the TUI.
// Used by --noninteractive mode and the B2 byte-equality test.
// viewName: "flat" | "turns" | "decisions" | "blockers" | "report"
func RenderView(data *views.WorklogData, viewName, turnsDir string) string {
	absTurns, _ := filepath.Abs(turnsDir)
	switch viewName {
	case "turns":
		return views.RenderTurns(data, absTurns)
	case "decisions":
		return views.RenderDecisions(data)
	case "blockers":
		return views.RenderBlockers(data)
	case "report":
		return views.RenderReport(data)
	default: // "flat" or anything else
		return views.RenderFlat(data, absTurns)
	}
}
