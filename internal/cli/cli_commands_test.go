package cli

// cli_commands_test.go: exercises cobra command RunE handlers directly,
// bypassing flag parsing. Each newXxxCmd() applies flag defaults via StringVar,
// so RunE receives correct zero-value defaults without cobra's Parse step.
//
// Strategy: chain commands in session order (start → turn-start → … → stop)
// so that the DB state required by each command is already present.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// execRunE calls cmd.RunE(cmd, args) and fatals on error.
// Use expectErr=true for commands that must return a non-nil error.
func execRunE(t *testing.T, cmd *cobra.Command, expectErr bool, args ...string) error {
	t.Helper()
	err := cmd.RunE(cmd, args)
	if expectErr && err == nil {
		t.Errorf("execRunE: expected error for %q, got nil", cmd.Use)
	} else if !expectErr && err != nil {
		t.Errorf("execRunE %q: %v", cmd.Use, err)
	}
	return err
}

// setupEnv sets WORKLOG_DIR to a fresh temp dir and clears TMUX so
// startPane takes the non-interactive path. Returns the worklog dir.
func setupEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WORKLOG_DIR", dir)
	t.Setenv("TMUX", "")
	t.Setenv("WORKLOG_VIEWER_SCRIPT", "") // suppress viewer script
	return dir
}

// TestMigrateCmd runs the migrate command on a fresh DB (no-op since Open
// already migrates, but it exercises the RunE handler).
func TestMigrateCmd(t *testing.T) {
	setupEnv(t)
	execRunE(t, newMigrateCmd(), false)
}

// TestPathCmd prints the worklog path to stdout.
func TestPathCmd(t *testing.T) {
	setupEnv(t)
	execRunE(t, newPathCmd(), false)
}

// TestStatusCmd updates the status label (requires ≥1 arg).
func TestStatusCmd(t *testing.T) {
	setupEnv(t)
	execRunE(t, newStatusCmd(), false, "활성", "작업 중", "다음 단계")
}

func TestStatusCmd_CreatesSessionWithoutStart(t *testing.T) {
	dir := setupEnv(t)
	execRunE(t, newStatusCmd(), false, "활성", "작업 중", "다음 단계")
	if b, err := os.ReadFile(filepath.Join(dir, "session-id")); err != nil || len(b) == 0 {
		t.Fatalf("status should create session-id, got len=%d err=%v", len(b), err)
	}
}

func TestModeCmd_CreatesSessionWithoutStart(t *testing.T) {
	dir := setupEnv(t)
	execRunE(t, newModeCmd(), false, "solo", "시작 없이 모드 기록")
	if b, err := os.ReadFile(filepath.Join(dir, "session-id")); err != nil || len(b) == 0 {
		t.Fatalf("mode should create session-id, got len=%d err=%v", len(b), err)
	}
}

func TestTurnStartCmd_CreatesSessionWithoutStart(t *testing.T) {
	dir := setupEnv(t)
	execRunE(t, newTurnStartCmd(), false, "시작 없이 턴")
	if b, err := os.ReadFile(filepath.Join(dir, "session-id")); err != nil || len(b) == 0 {
		t.Fatalf("turn-start should create session-id, got len=%d err=%v", len(b), err)
	}
}

// TestReportCmd_Text runs report --format text on an empty DB.
func TestReportCmd_Text(t *testing.T) {
	setupEnv(t)
	// format defaults to "text", window defaults to "all" (set by StringVar).
	execRunE(t, newReportCmd(), false)
}

// TestReportCmd_JSON runs report --format json on an empty DB.
func TestReportCmd_JSON(t *testing.T) {
	setupEnv(t)
	cmd := newReportCmd()
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format flag: %v", err)
	}
	execRunE(t, cmd, false)
}

// TestReportCmd_BadFormat verifies the bad-format error branch.
func TestReportCmd_BadFormat(t *testing.T) {
	setupEnv(t)
	cmd := newReportCmd()
	if err := cmd.Flags().Set("format", "xml"); err != nil {
		t.Fatalf("set format flag: %v", err)
	}
	execRunE(t, cmd, true)
}

// TestReportCmd_BadWindow verifies the bad-window error branch.
func TestReportCmd_BadWindow(t *testing.T) {
	setupEnv(t)
	cmd := newReportCmd()
	if err := cmd.Flags().Set("window", "month"); err != nil {
		t.Fatalf("set window flag: %v", err)
	}
	execRunE(t, cmd, true)
}

// TestReportCmd_DayWindow runs report with window=day.
func TestReportCmd_DayWindow(t *testing.T) {
	setupEnv(t)
	cmd := newReportCmd()
	if err := cmd.Flags().Set("window", "day"); err != nil {
		t.Fatalf("set window flag: %v", err)
	}
	execRunE(t, cmd, false)
}

// TestViewCmd_AllModes exercises view flat/turns/decisions/blockers.
func TestViewCmd_AllModes(t *testing.T) {
	setupEnv(t)
	for _, mode := range []string{"flat", "turns", "decisions", "blockers"} {
		t.Run(mode, func(t *testing.T) {
			setupEnv(t) // fresh dir per subtest
			execRunE(t, newViewCmd(), false, mode)
		})
	}
}

// TestViewCmd_Default (no args → flat).
func TestViewCmd_Default(t *testing.T) {
	setupEnv(t)
	execRunE(t, newViewCmd(), false)
}

// TestAutoCmd creates a session if none is present.
func TestAutoCmd(t *testing.T) {
	setupEnv(t)
	execRunE(t, newAutoCmd(), false, "자동 세션")
}

// TestStartCmd starts a new session without TMUX.
func TestStartCmd(t *testing.T) {
	setupEnv(t)
	execRunE(t, newStartCmd(), false, "테스트 세션")
}

// TestStartCmd_NoTitle uses the default title.
func TestStartCmd_NoTitle(t *testing.T) {
	setupEnv(t)
	execRunE(t, newStartCmd(), false)
}

// TestCommandChain_FullWorkflow is the primary integration test:
// start → turn-start → entry/decision/evidence/blocker/mode
// → refresh-anchor → turn-end → stop.
// This covers the RunE of 10+ commands in a single shared DB state.
func TestCommandChain_FullWorkflow(t *testing.T) {
	setupEnv(t)

	// 1. start
	if err := newStartCmd().RunE(newStartCmd(), []string{"통합 테스트 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}

	// 2. turn-start (with intent anchor)
	tsCmd := newTurnStartCmd()
	if err := tsCmd.Flags().Set("intent", "테스트 의도"); err != nil {
		t.Fatalf("set intent: %v", err)
	}
	if err := tsCmd.Flags().Set("constraints", "제약1,제약2"); err != nil {
		t.Fatalf("set constraints: %v", err)
	}
	if err := tsCmd.Flags().Set("done-when", "완료 기준"); err != nil {
		t.Fatalf("set done-when: %v", err)
	}
	if err := tsCmd.RunE(tsCmd, []string{"첫 번째 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}

	// 3. entry
	if err := newEntryCmd().RunE(newEntryCmd(), []string{"항목 제목"}); err != nil {
		t.Fatalf("entry: %v", err)
	}

	// 4. decision
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"결정 항목"}); err != nil {
		t.Fatalf("decision: %v", err)
	}

	// 5. evidence
	if err := newEvidenceCmd().RunE(newEvidenceCmd(), []string{"근거 항목"}); err != nil {
		t.Fatalf("evidence: %v", err)
	}

	// 6. blocker
	if err := newBlockerCmd().RunE(newBlockerCmd(), []string{"블로커 항목"}); err != nil {
		t.Fatalf("blocker: %v", err)
	}

	// 7. mode
	if err := newModeCmd().RunE(newModeCmd(), []string{"solo"}); err != nil {
		t.Fatalf("mode: %v", err)
	}

	// 8. entry with detail (2-arg form)
	if err := newEntryCmd().RunE(newEntryCmd(), []string{"상세 항목", "상세 내용"}); err != nil {
		t.Fatalf("entry with detail: %v", err)
	}

	// 9. refresh-anchor (active turn with intent).
	if err := newRefreshAnchorCmd().RunE(newRefreshAnchorCmd(), []string{}); err != nil {
		t.Fatalf("refresh-anchor: %v", err)
	}

	// 10. turn-path
	if err := newTurnPathCmd().RunE(newTurnPathCmd(), []string{}); err != nil {
		t.Fatalf("turn-path: %v", err)
	}

	// 11. report (text, day, week windows)
	for _, window := range []string{"day", "week", "all"} {
		cmd := newReportCmd()
		if err := cmd.Flags().Set("window", window); err != nil {
			t.Fatalf("set window=%s: %v", window, err)
		}
		if err := cmd.RunE(cmd, []string{}); err != nil {
			t.Fatalf("report window=%s: %v", window, err)
		}
	}

	// 12. report json
	jsonCmd := newReportCmd()
	if err := jsonCmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format json: %v", err)
	}
	if err := jsonCmd.RunE(jsonCmd, []string{}); err != nil {
		t.Fatalf("report json: %v", err)
	}

	// 13. turn-end
	if err := newTurnEndCmd().RunE(newTurnEndCmd(), []string{}); err != nil {
		t.Fatalf("turn-end: %v", err)
	}

	// 14. stop
	if err := newStopCmd().RunE(newStopCmd(), []string{}); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

// TestRefreshAnchorCmd_NoTurn verifies the "no active turn" error path.
func TestRefreshAnchorCmd_NoTurn(t *testing.T) {
	setupEnv(t)
	execRunE(t, newRefreshAnchorCmd(), true) // expects error
}

// TestDriftCheckCmd_WithTurn runs drift-check after start+turn-start.
func TestDriftCheckCmd_WithTurn(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"드리프트 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	tsCmd := newTurnStartCmd()
	if err := tsCmd.Flags().Set("intent", "드리프트 의도"); err != nil {
		t.Fatalf("set intent: %v", err)
	}
	if err := tsCmd.RunE(tsCmd, []string{"드리프트 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	execRunE(t, newDriftCheckCmd(), false)
}

// TestDriftCheckCmd_NoTurn verifies the no-turn error path for drift-check.
func TestDriftCheckCmd_NoTurn(t *testing.T) {
	setupEnv(t)
	execRunE(t, newDriftCheckCmd(), true)
}

// TestRefreshAnchorCmd_WithTurnNoAnchor runs refresh-anchor on a turn
// that has no intent (exercises the "no anchor" output branch).
func TestRefreshAnchorCmd_WithTurnNoAnchor(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"앵커 없는 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newTurnStartCmd().RunE(newTurnStartCmd(), []string{"앵커 없는 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	// No intent set → refresh-anchor prints "no anchor" message, no error.
	execRunE(t, newRefreshAnchorCmd(), false)
}

// TestExecute calls the exported Execute function (covers the wrapper).
func TestExecute(t *testing.T) {
	setupEnv(t)
	// Set args to a no-op command (path) that exits 0.
	rootCmd.SetArgs([]string{"path"})
	if err := Execute(); err != nil {
		t.Errorf("Execute path: %v", err)
	}
	// Reset args to avoid leaking state to other tests.
	rootCmd.SetArgs(nil)
}

// ── view command extra branches ───────────────────────────────────────────

// TestViewCmd_TUI_NonInteractive runs view tui --noninteractive (bypasses TUI).
func TestViewCmd_TUI_NonInteractive(t *testing.T) {
	setupEnv(t)
	cmd := newViewCmd()
	if err := cmd.Flags().Set("noninteractive", "true"); err != nil {
		t.Fatalf("set noninteractive: %v", err)
	}
	execRunE(t, cmd, false, "tui")
}

// TestViewCmd_UnknownMode verifies the unknown-mode error branch.
func TestViewCmd_UnknownMode(t *testing.T) {
	setupEnv(t)
	execRunE(t, newViewCmd(), true, "xml-view")
}

// ── turn-path extra branches ──────────────────────────────────────────────

// TestTurnPathCmd_ExplicitN resolves an explicit valid turn number.
func TestTurnPathCmd_ExplicitN(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"경로 테스트"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newTurnStartCmd().RunE(newTurnStartCmd(), []string{"턴 경로"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	execRunE(t, newTurnPathCmd(), false, "1")
}

// TestTurnPathCmd_InvalidN returns an error for a non-numeric argument.
func TestTurnPathCmd_InvalidN(t *testing.T) {
	setupEnv(t)
	execRunE(t, newTurnPathCmd(), true, "abc")
}

// TestTurnPathCmd_ZeroArg returns an error for "0" (n < 1 branch).
func TestTurnPathCmd_ZeroArg(t *testing.T) {
	setupEnv(t)
	execRunE(t, newTurnPathCmd(), true, "0")
}

// TestTurnPathCmd_NoActiveTurn returns an error when no turn is active.
func TestTurnPathCmd_NoActiveTurn(t *testing.T) {
	setupEnv(t)
	// No turn started → CurrentTurnNumber() returns 0.
	execRunE(t, newTurnPathCmd(), true)
}

// ── drift-check extra branches ────────────────────────────────────────────

// TestDriftCheckCmd_WithConstraints_NoDrift exercises the "all entries match" path.
func TestDriftCheckCmd_WithConstraints_NoDrift(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"제약 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	tsCmd := newTurnStartCmd()
	if err := tsCmd.Flags().Set("intent", "의도"); err != nil {
		t.Fatalf("set intent: %v", err)
	}
	if err := tsCmd.Flags().Set("constraints", "항목"); err != nil {
		t.Fatalf("set constraints: %v", err)
	}
	if err := tsCmd.RunE(tsCmd, []string{"제약 있는 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	// Entry title contains "항목" → matches constraint → no drift.
	if err := newEntryCmd().RunE(newEntryCmd(), []string{"항목 진행 중"}); err != nil {
		t.Fatalf("entry: %v", err)
	}
	execRunE(t, newDriftCheckCmd(), false)
}

// TestDriftCheckCmd_WithDrift exercises the drift-detected path (blocker inserted).
func TestDriftCheckCmd_WithDrift(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"드리프트 발생 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	tsCmd := newTurnStartCmd()
	if err := tsCmd.Flags().Set("intent", "의도"); err != nil {
		t.Fatalf("set intent: %v", err)
	}
	if err := tsCmd.Flags().Set("constraints", "관련키워드"); err != nil {
		t.Fatalf("set constraints: %v", err)
	}
	if err := tsCmd.RunE(tsCmd, []string{"드리프트 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	// Entry title does NOT contain "관련키워드" → drift detected.
	if err := newEntryCmd().RunE(newEntryCmd(), []string{"전혀 다른 작업"}); err != nil {
		t.Fatalf("entry: %v", err)
	}
	execRunE(t, newDriftCheckCmd(), false)
}

// TestDriftCheckCmd_ExplicitTurnID_NotFound returns an error for an unknown turn ID.
func TestDriftCheckCmd_ExplicitTurnID_NotFound(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	execRunE(t, newDriftCheckCmd(), true, "nonexistent-turn-uuid")
}

// ── evidence --link branch ────────────────────────────────────────────────

// TestEvidenceCmd_WithLink exercises the INSERT OR IGNORE link branch.
func TestEvidenceCmd_WithLink(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"근거 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newTurnStartCmd().RunE(newTurnStartCmd(), []string{"근거 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	cmd := newEvidenceCmd()
	if err := cmd.Flags().Set("link", "fake-decision-id"); err != nil {
		t.Fatalf("set link: %v", err)
	}
	// INSERT OR IGNORE: no error even with non-existent decision ID.
	execRunE(t, cmd, false, "링크된 근거")
}

// ── migrate extra branches ────────────────────────────────────────────────

// TestMigrateCmd_Down exercises the "down" direction branch.
func TestMigrateCmd_Down(t *testing.T) {
	setupEnv(t)
	execRunE(t, newMigrateCmd(), false, "down")
}

// TestMigrateCmd_InvalidDirection hits the default → cmd.Help() branch.
func TestMigrateCmd_InvalidDirection(t *testing.T) {
	setupEnv(t)
	execRunE(t, newMigrateCmd(), false, "sideways")
}

// ── mode with detail ──────────────────────────────────────────────────────

// TestModeCmd_WithDetail exercises the 2-arg (detail) branch of mode.
func TestModeCmd_WithDetail(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"모드 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	execRunE(t, newModeCmd(), false, "코딩", "자세한 모드 설명")
}

// ── auto with existing session ────────────────────────────────────────────

// TestAutoCmd_ExistingSession runs auto when a session already exists
// (exercises the "sessionID != empty → skip insertSession" branch).
func TestAutoCmd_ExistingSession(t *testing.T) {
	setupEnv(t)
	// First call: creates the session.
	execRunE(t, newAutoCmd(), false, "첫 번째 자동")
	// Second call: session already exists → takes existing-session branch.
	execRunE(t, newAutoCmd(), false, "두 번째 자동")
}

// ── blocker with detail ───────────────────────────────────────────────────

// TestBlockerCmd_WithDetail exercises the 2-arg detail branch of blocker.
func TestBlockerCmd_WithDetail(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"블로커 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	execRunE(t, newBlockerCmd(), false, "블로커 제목", "블로커 상세 내용")
}

// ── evidence with detail ──────────────────────────────────────────────────

// TestEvidenceCmd_WithDetail exercises the 2-arg detail branch of evidence.
func TestEvidenceCmd_WithDetail(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"근거 상세 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	execRunE(t, newEvidenceCmd(), false, "근거 제목", "근거 상세 내용")
}
