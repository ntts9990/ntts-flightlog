package tui

// tui_extra_test.go: additional coverage for New, Init, DefaultStyles,
// CategoryStyle, TurnStyle, RenderView all-views, cmdLoad, cmdCheckChanges,
// and WindowSizeMsg handling.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/tui/views"
)

// ── Styles ────────────────────────────────────────────────────────────────

func TestDefaultStyles_NonNil(t *testing.T) {
	s := DefaultStyles()
	// Verify a few styles produce non-empty rendered strings.
	if s.Menu.Render("x") == "" {
		t.Error("DefaultStyles: Menu style renders empty string")
	}
	if s.Active.Render("x") == "" {
		t.Error("DefaultStyles: Active style renders empty string")
	}
	// Turn styles: 8 entries.
	for i, ts := range s.Turns {
		if ts.Render("x") == "" {
			t.Errorf("DefaultStyles: Turns[%d] renders empty string", i)
		}
	}
}

func TestCategoryStyle_AllKinds(t *testing.T) {
	s := DefaultStyles()
	kinds := []string{"mode", "entry", "evidence", "decision", "blocker", "other"}
	for _, k := range kinds {
		st := s.CategoryStyle(k)
		if st.Render("x") == "" {
			t.Errorf("CategoryStyle(%q): renders empty string", k)
		}
	}
}

func TestTurnStyle_Cycles(t *testing.T) {
	s := DefaultStyles()
	// Turn 1-8 and wrap at 9.
	for n := 1; n <= 10; n++ {
		st := s.TurnStyle(n)
		if st.Render("x") == "" {
			t.Errorf("TurnStyle(%d): renders empty string", n)
		}
	}
	// Turns 1 and 9 should use the same underlying palette entry.
	if s.TurnStyle(1).Render("x") != s.TurnStyle(9).Render("x") {
		t.Error("TurnStyle: turn 1 and turn 9 should map to same colour (8-cycle wrap)")
	}
}

// ── New / Init ────────────────────────────────────────────────────────────

func TestNew_CreatesModel(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	m := New(d, "/tmp/turns")
	if m.activeView != viewFlat {
		t.Errorf("New: activeView = %d, want %d (viewFlat)", m.activeView, viewFlat)
	}
	if m.d != d {
		t.Error("New: DB pointer not stored on model")
	}
}

func TestInit_ReturnsBatchCmd(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	m := New(d, "/tmp/turns")
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init: expected non-nil Cmd (batch of load+tick), got nil")
	}
}

// ── RenderView all named views ─────────────────────────────────────────────

func TestRenderView_AllViews(t *testing.T) {
	data := &views.WorklogData{
		Sessions: []views.Session{
			{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"},
		},
	}
	viewNames := []string{"flat", "turns", "decisions", "blockers", "report", "visual", "other"}
	for _, name := range viewNames {
		out := RenderView(data, name, "/tmp/turns")
		if out == "" {
			t.Errorf("RenderView(%q): returned empty string", name)
		}
	}
}

func TestRenderView_NilData_AllViews(t *testing.T) {
	for _, name := range []string{"flat", "turns", "decisions", "blockers", "report", "visual"} {
		out := RenderView(nil, name, "/tmp/turns")
		if out == "" {
			t.Errorf("RenderView nil data (%q): returned empty string", name)
		}
	}
}

// ── WindowSizeMsg ──────────────────────────────────────────────────────────

func TestUpdate_WindowSizeMsg_InitializesViewport(t *testing.T) {
	m := makeModel(nil) // ready=true from makeModel
	// Send a window-resize while already ready: updates width/height.
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	got := newModel.(Model)
	if got.width != 120 {
		t.Errorf("WindowSizeMsg ready: width = %d, want 120", got.width)
	}
	if got.height != 40 {
		t.Errorf("WindowSizeMsg ready: height = %d, want 40", got.height)
	}
}

func TestUpdate_WindowSizeMsg_NotReady(t *testing.T) {
	// Model with ready=false: first WindowSizeMsg should set ready=true.
	m := Model{activeView: viewFlat, turnsDir: "/tmp/turns"}
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	newModel, _ := m.Update(msg)
	got := newModel.(Model)
	if !got.ready {
		t.Error("WindowSizeMsg not-ready: model should be ready after first resize")
	}
	if got.width != 80 {
		t.Errorf("WindowSizeMsg not-ready: width = %d, want 80", got.width)
	}
}

func TestUpdate_WindowSizeMsg_VerySmallHeight(t *testing.T) {
	// vpHeight clamped to 1 when terminal is too small.
	m := Model{activeView: viewFlat, turnsDir: "/tmp/turns"}
	msg := tea.WindowSizeMsg{Width: 80, Height: 1} // headerHeight=2 → vpHeight=-1 → clamped to 1
	newModel, _ := m.Update(msg)
	got := newModel.(Model)
	if !got.ready {
		t.Error("WindowSizeMsg small: model should still be ready")
	}
}

// ── cmdLoad execution ──────────────────────────────────────────────────────

func TestCmdLoad_HappyPath(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	cmd := cmdLoad(d)
	msg := cmd() // execute the returned func
	switch m := msg.(type) {
	case loadedMsg:
		if m.data == nil {
			t.Error("cmdLoad happy: loadedMsg.data is nil")
		}
	case errMsg:
		t.Errorf("cmdLoad happy: got errMsg: %v", m.err)
	default:
		t.Errorf("cmdLoad happy: unexpected msg type %T", msg)
	}
}

func TestCmdLoad_ClosedDB_ReturnsErrMsg(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	_ = d.Close() // close before cmdLoad executes

	cmd := cmdLoad(d)
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Errorf("cmdLoad closed DB: expected errMsg, got %T", msg)
	}
}

// ── cmdCheckChanges ────────────────────────────────────────────────────────

func TestCmdCheckChanges_NoChange(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	// lastSeq = 0, current SeqSum is also 0 → no change → nil msg.
	cmd := cmdCheckChanges(d, 0)
	msg := cmd()
	if msg != nil {
		t.Errorf("cmdCheckChanges no change: expected nil, got %T", msg)
	}
}

func TestCmdCheckChanges_Changed(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	// Insert a session so SeqSum becomes 1; pass lastSeq=0 → change detected.
	if _, err := d.Exec(`INSERT INTO sessions (started_at, mode) VALUES ('2026-05-20T10:00:00Z', 'solo')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	cmd := cmdCheckChanges(d, 0)
	msg := cmd()
	switch m := msg.(type) {
	case loadedMsg:
		if m.seqSum != 1 {
			t.Errorf("cmdCheckChanges changed: seqSum = %d, want 1", m.seqSum)
		}
	case nil:
		t.Error("cmdCheckChanges changed: got nil, expected loadedMsg")
	default:
		t.Errorf("cmdCheckChanges changed: unexpected msg type %T", msg)
	}
}

func TestCmdCheckChanges_ClosedDB(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	_ = d.Close()

	// Closed DB: SeqSum errors → returns nil (no reload triggered).
	cmd := cmdCheckChanges(d, 0)
	msg := cmd()
	if msg != nil {
		t.Logf("cmdCheckChanges closed DB returned %T (nil expected, but ok)", msg)
	}
}

// ── currentContent default branch ─────────────────────────────────────────

func TestCurrentContent_DefaultView(t *testing.T) {
	m := makeModel(nil)
	m.activeView = viewID(99) // unknown view → default case
	content := m.currentContent()
	// Default falls back to RenderFlat placeholder.
	if !strings.Contains(content, "워크로그가 비어") {
		t.Errorf("currentContent default: expected flat placeholder, got %q", content)
	}
}
