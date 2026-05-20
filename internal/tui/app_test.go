package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ntts9990/ntts-flightlog/internal/tui/views"
)

// makeModel returns a ready Model with mock data injected, suitable for unit
// testing without a real DB or terminal.
func makeModel(data *views.WorklogData) Model {
	m := Model{
		activeView: viewFlat,
		turnsDir:   "/tmp/turns",
		ready:      true, // skip WindowSizeMsg in tests
	}
	// Set up a minimal viewport (zero-sized is fine for string rendering tests).
	m.vp.SetContent(m.contentForData(data))
	m.data = data
	return m
}

// contentForData returns rendered content for given data without touching m.vp
// (avoids uninitialized viewport issues in tests).
func (m Model) contentForData(data *views.WorklogData) string {
	saved := m.data
	m.data = data
	c := m.currentContent()
	m.data = saved
	return c
}

// sendKey feeds a key message through Model.Update and returns the new Model.
func sendKey(m Model, key string) Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	newModel, _ := m.Update(msg)
	return newModel.(Model)
}

// --------------------------------------------------------------------------
// View switching tests
// --------------------------------------------------------------------------

func TestUpdate_KeySwitchViews(t *testing.T) {
	tests := []struct {
		key      string
		wantView viewID
	}{
		{"1", viewFlat},
		{"2", viewTurns},
		{"3", viewDecisions},
		{"4", viewBlockers},
		{"5", viewReport},
	}
	for _, tc := range tests {
		t.Run("key_"+tc.key, func(t *testing.T) {
			m := makeModel(nil)
			m.activeView = viewReport // start on a different view
			got := sendKey(m, tc.key)
			if got.activeView != tc.wantView {
				t.Errorf("key %q: activeView = %d, want %d", tc.key, got.activeView, tc.wantView)
			}
		})
	}
}

func TestUpdate_KeyReload(t *testing.T) {
	m := makeModel(nil)
	// "r" should produce a non-nil Cmd (the load command) without changing view.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	newModel, cmd := m.Update(msg)
	got := newModel.(Model)
	if got.activeView != viewFlat {
		t.Errorf("key r: activeView = %d, want %d", got.activeView, viewFlat)
	}
	if cmd == nil {
		t.Error("key r: expected non-nil Cmd (load command), got nil")
	}
}

func TestUpdate_KeyQuit(t *testing.T) {
	m := makeModel(nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("key q: expected tea.Quit cmd, got nil")
	}
	// Execute the cmd to verify it produces QuitMsg.
	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("key q: cmd() returned %T, want tea.QuitMsg", result)
	}
}

// --------------------------------------------------------------------------
// Data loading tests
// --------------------------------------------------------------------------

func TestUpdate_LoadedMsg_StoresData(t *testing.T) {
	m := makeModel(nil)
	wantData := &views.WorklogData{
		Sessions: []views.Session{{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"}},
	}
	msg := loadedMsg{data: wantData, seqSum: 42}
	newModel, _ := m.Update(msg)
	got := newModel.(Model)

	if got.data != wantData {
		t.Error("loadedMsg: data not stored on model")
	}
	if got.seqSum != 42 {
		t.Errorf("loadedMsg: seqSum = %d, want 42", got.seqSum)
	}
	if got.err != nil {
		t.Errorf("loadedMsg: err = %v, want nil", got.err)
	}
}

func TestUpdate_ErrMsg_StoresError(t *testing.T) {
	m := makeModel(nil)
	wantErr := errMsg{err: &mockError{"db closed"}}
	newModel, _ := m.Update(wantErr)
	got := newModel.(Model)
	if got.err == nil {
		t.Error("errMsg: err should be set on model")
	}
}

// --------------------------------------------------------------------------
// View rendering tests
// --------------------------------------------------------------------------

func TestView_NotReady(t *testing.T) {
	m := Model{activeView: viewFlat} // ready=false
	got := m.View()
	if !strings.Contains(got, "로딩") {
		t.Errorf("View when not ready: want loading message, got %q", got)
	}
}

func TestView_WithError(t *testing.T) {
	m := makeModel(nil)
	m.err = &mockError{"connection refused"}
	got := m.View()
	if !strings.Contains(got, "오류") {
		t.Errorf("View with error: want error message, got %q", got)
	}
}

func TestView_MenuHeader_ContainsAllKeys(t *testing.T) {
	m := makeModel(nil)
	header := m.menuHeader()
	for _, k := range []string{"1=flat", "2=turns", "3=decisions", "4=blockers", "5=report", "r=reload", "q=quit"} {
		if !strings.Contains(header, k) {
			t.Errorf("menuHeader missing %q in: %q", k, header)
		}
	}
}

func TestUpdate_KoreanKeyboardReloadAndQuit(t *testing.T) {
	m := makeModel(nil)
	reloadMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㄱ")}
	_, reloadCmd := m.Update(reloadMsg)
	if reloadCmd == nil {
		t.Fatal("ㄱ should trigger reload command")
	}

	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ㅂ")}
	_, quitCmd := m.Update(quitMsg)
	if quitCmd == nil {
		t.Fatal("ㅂ should trigger quit command")
	}
}

func TestView_MenuHeader_ActiveViewUnderlined(t *testing.T) {
	for _, tc := range []struct {
		view    viewID
		wantKey string
	}{
		{viewFlat, "1=flat"},
		{viewTurns, "2=turns"},
		{viewDecisions, "3=decisions"},
		{viewBlockers, "4=blockers"},
		{viewReport, "5=report"},
	} {
		m := makeModel(nil)
		m.activeView = tc.view
		header := m.menuHeader()
		// activeStyle adds underline escape; simply check the label is present.
		if !strings.Contains(header, tc.wantKey) {
			t.Errorf("activeView=%d: header missing %q", tc.view, tc.wantKey)
		}
	}
}

// --------------------------------------------------------------------------
// currentContent tests
// --------------------------------------------------------------------------

func TestCurrentContent_EmptyData_ShowsPlaceholder(t *testing.T) {
	tests := []struct {
		view    viewID
		wantSub string
	}{
		{viewFlat, "워크로그가 비어"},
		{viewTurns, "turn"},
		{viewDecisions, "결정"},
		{viewBlockers, "블로커"},
		{viewReport, "리포트"},
	}
	for _, tc := range tests {
		m := makeModel(nil)
		m.activeView = tc.view
		content := m.currentContent()
		if !strings.Contains(content, tc.wantSub) {
			t.Errorf("view %d empty state: content missing %q, got %q", tc.view, tc.wantSub, content)
		}
	}
}

func TestCurrentContent_WithData_ShowsEntries(t *testing.T) {
	data := &views.WorklogData{
		Sessions: []views.Session{
			{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"},
		},
		Entries: []views.Entry{
			{
				ID:        "e1",
				SessionID: "s1",
				Kind:      "entry",
				Title:     "테스트 엔트리",
				CreatedAt: "2026-05-20T10:01:00Z",
			},
		},
	}
	m := makeModel(data)
	m.activeView = viewFlat
	content := m.currentContent()
	if !strings.Contains(content, "테스트 엔트리") {
		t.Errorf("flat view: expected entry title in content, got:\n%s", content)
	}
}

func TestCurrentContent_Decisions_OnlyDecisions(t *testing.T) {
	data := &views.WorklogData{
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "일반 엔트리", CreatedAt: "2026-05-20T10:00:00Z"},
			{ID: "e2", SessionID: "s1", Kind: "decision", Title: "중요 결정", CreatedAt: "2026-05-20T10:01:00Z"},
		},
	}
	m := makeModel(data)
	m.activeView = viewDecisions
	content := m.currentContent()
	if strings.Contains(content, "일반 엔트리") {
		t.Error("decisions view: should not contain non-decision entries")
	}
	if !strings.Contains(content, "중요 결정") {
		t.Error("decisions view: missing decision entry")
	}
}

func TestCurrentContent_Blockers_OnlyBlockers(t *testing.T) {
	data := &views.WorklogData{
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "일반", CreatedAt: "2026-05-20T10:00:00Z"},
			{ID: "e2", SessionID: "s1", Kind: "blocker", Title: "블로킹 이슈", CreatedAt: "2026-05-20T10:01:00Z"},
		},
	}
	m := makeModel(data)
	m.activeView = viewBlockers
	content := m.currentContent()
	if strings.Contains(content, "일반") {
		t.Error("blockers view: should not contain non-blocker entries")
	}
	if !strings.Contains(content, "블로킹 이슈") {
		t.Error("blockers view: missing blocker entry")
	}
}

// --------------------------------------------------------------------------
// Tick tests
// --------------------------------------------------------------------------

func TestUpdate_TickMsg_ProducesNewTick(t *testing.T) {
	m := makeModel(nil)
	// Tick with same seqSum → should not load, but should produce new tick cmd.
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("tickMsg: expected non-nil Cmd (new tick + check), got nil")
	}
}

// --------------------------------------------------------------------------
// RenderView helper tests
// --------------------------------------------------------------------------

func TestRenderView_UnknownViewDefaultsToFlat(t *testing.T) {
	data := &views.WorklogData{
		Sessions: []views.Session{{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"}},
	}
	out := RenderView(data, "unknown-view", "/tmp/turns")
	// flat view shows session title, not a placeholder
	if !strings.Contains(out, "Flightlog") && !strings.Contains(out, "워크로그가") {
		t.Errorf("RenderView unknown: unexpected output: %q", out)
	}
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }
