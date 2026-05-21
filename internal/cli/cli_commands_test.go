package cli

// cli_commands_test.go: exercises cobra command RunE handlers directly,
// bypassing flag parsing. Each newXxxCmd() applies flag defaults via StringVar,
// so RunE receives correct zero-value defaults without cobra's Parse step.
//
// Strategy: chain commands in session order (start → turn-start → … → stop)
// so that the DB state required by each command is already present.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
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

func TestLaneTurnsKeepSeparateActivePointers(t *testing.T) {
	dir := setupEnv(t)
	oldLane := laneFlag
	t.Cleanup(func() { laneFlag = oldLane })

	laneFlag = "worker-a"
	execRunE(t, newTurnStartCmd(), false, "worker A 턴")
	execRunE(t, newEntryCmd(), false, "A 작업", "lane A")

	laneFlag = "worker-b"
	execRunE(t, newTurnStartCmd(), false, "worker B 턴")
	execRunE(t, newEntryCmd(), false, "B 작업", "lane B")

	if _, err := os.Stat(filepath.Join(dir, "lanes", "worker-a", "turn-id")); err != nil {
		t.Fatalf("worker-a active turn missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lanes", "worker-b", "turn-id")); err != nil {
		t.Fatalf("worker-b active turn missing: %v", err)
	}

	laneFlag = "worker-a"
	execRunE(t, newTurnEndCmd(), false, "A 완료")
	if _, err := os.Stat(filepath.Join(dir, "lanes", "worker-a", "turn-id")); !os.IsNotExist(err) {
		t.Fatalf("worker-a active turn should be cleared, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lanes", "worker-b", "turn-id")); err != nil {
		t.Fatalf("worker-b active turn should remain: %v", err)
	}

	store, err := db.Open(filepath.Join(dir, "flightlog.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	rows, err := store.Query(`SELECT lane, COUNT(*) FROM turns GROUP BY lane ORDER BY lane`)
	if err != nil {
		t.Fatalf("query lanes: %v", err)
	}
	defer rows.Close()
	got := map[string]int{}
	for rows.Next() {
		var lane string
		var count int
		if err := rows.Scan(&lane, &count); err != nil {
			t.Fatalf("scan lane: %v", err)
		}
		got[lane] = count
	}
	if got["worker-a"] != 1 || got["worker-b"] != 1 {
		t.Fatalf("turn lane counts = %#v", got)
	}

	aTurn, err := os.ReadFile(filepath.Join(dir, "turns", "turn-1.md"))
	if err != nil {
		t.Fatalf("read turn-1: %v", err)
	}
	bTurn, err := os.ReadFile(filepath.Join(dir, "turns", "turn-2.md"))
	if err != nil {
		t.Fatalf("read turn-2: %v", err)
	}
	if !strings.Contains(string(aTurn), "A 작업") || strings.Contains(string(aTurn), "B 작업") {
		t.Fatalf("turn-1 lane mirror wrong:\n%s", string(aTurn))
	}
	if !strings.Contains(string(bTurn), "B 작업") || strings.Contains(string(bTurn), "A 작업") {
		t.Fatalf("turn-2 lane mirror wrong:\n%s", string(bTurn))
	}
}

func TestIngestCmd_RedactsAndPromotesTestPass(t *testing.T) {
	dir := setupEnv(t)
	execRunE(t, newTurnStartCmd(), false, "ingest turn")

	cmd := newIngestCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(`{
	  "source": "codex",
	  "event_name": "test.finished",
	  "summary": "go test ./... OPENAI_API_KEY=sk-secret passed",
	  "command": "go test ./...",
	  "exit_code": 0,
	  "stdout": "raw stdout must not be stored"
	}`))
	execRunE(t, cmd, false)

	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode ingest response: %v\n%s", err, out.String())
	}
	if resp["promotion_status"] != "promoted" || resp["duplicate"] != false {
		t.Fatalf("response = %#v", resp)
	}

	store, err := db.Open(filepath.Join(dir, "flightlog.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	var summary string
	var dropped int
	if err := store.QueryRow(`SELECT summary, dropped_field_count FROM agent_events WHERE event_name = 'test.finished'`).Scan(&summary, &dropped); err != nil {
		t.Fatalf("query agent_events: %v", err)
	}
	if strings.Contains(summary, "sk-secret") || strings.Contains(summary, "raw stdout") {
		t.Fatalf("agent event leaked raw payload: %q", summary)
	}
	if dropped != 1 {
		t.Fatalf("dropped_field_count = %d, want 1", dropped)
	}
	var evidenceCount int
	if err := store.QueryRow(`SELECT COUNT(*) FROM entries WHERE kind = 'evidence' AND title LIKE 'Hook evidence candidate:%'`).Scan(&evidenceCount); err != nil {
		t.Fatalf("query evidence entries: %v", err)
	}
	if evidenceCount != 1 {
		t.Fatalf("evidenceCount = %d, want 1", evidenceCount)
	}
}

func TestIngestCmd_PromotesTestFailureToBlocker(t *testing.T) {
	dir := setupEnv(t)
	cmd := newIngestCmd()
	cmd.SetIn(strings.NewReader(`{
	  "source": "codex",
	  "event_name": "test.finished",
	  "summary": "go test ./internal/db failed",
	  "exit_code": 1,
	  "dedupe_key": "failed-test-1"
	}`))
	execRunE(t, cmd, false)

	store, err := db.Open(filepath.Join(dir, "flightlog.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	var blockerCount int
	if err := store.QueryRow(`SELECT COUNT(*) FROM blockers b JOIN entries e ON e.id = b.entry_id WHERE e.title LIKE 'Hook blocker candidate:%'`).Scan(&blockerCount); err != nil {
		t.Fatalf("query blockers: %v", err)
	}
	if blockerCount != 1 {
		t.Fatalf("blockerCount = %d, want 1", blockerCount)
	}
}

func TestIngestCmd_DeduplicatesEvents(t *testing.T) {
	dir := setupEnv(t)
	raw := `{
	  "source": "codex",
	  "event_name": "test.finished",
	  "summary": "go test ./... passed",
	  "exit_code": 0,
	  "dedupe_key": "same-event"
	}`
	first := newIngestCmd()
	first.SetIn(strings.NewReader(raw))
	execRunE(t, first, false)

	second := newIngestCmd()
	var out bytes.Buffer
	second.SetOut(&out)
	second.SetIn(strings.NewReader(raw))
	execRunE(t, second, false)

	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode duplicate response: %v", err)
	}
	if resp["duplicate"] != true || resp["promotion_status"] != "duplicate" {
		t.Fatalf("duplicate response = %#v", resp)
	}

	store, err := db.Open(filepath.Join(dir, "flightlog.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	var eventCount, evidenceCount int
	if err := store.QueryRow(`SELECT COUNT(*) FROM agent_events`).Scan(&eventCount); err != nil {
		t.Fatalf("query events: %v", err)
	}
	if err := store.QueryRow(`SELECT COUNT(*) FROM entries WHERE title LIKE 'Hook evidence candidate:%'`).Scan(&evidenceCount); err != nil {
		t.Fatalf("query entries: %v", err)
	}
	if eventCount != 1 || evidenceCount != 1 {
		t.Fatalf("counts event=%d evidence=%d, want 1/1", eventCount, evidenceCount)
	}
}

func TestIngestCmd_InvalidJSONDoesNotLeakPayload(t *testing.T) {
	setupEnv(t)
	cmd := newIngestCmd()
	cmd.SetIn(strings.NewReader(`{"event_name":"test.finished","summary":"OPENAI_API_KEY=sk-secret"`))
	err := execRunE(t, cmd, true)
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if strings.Contains(err.Error(), "sk-secret") || strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("invalid JSON error leaked payload: %v", err)
	}
}

func TestHooksPrintCmd_DoesNotMutateConfig(t *testing.T) {
	setupEnv(t)
	cmd := newHooksPrintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Flags().Set("agent", "codex"); err != nil {
		t.Fatalf("set agent: %v", err)
	}
	execRunE(t, cmd, false)
	got := out.String()
	for _, want := range []string{"Codex hook starter kit", "ingest --source codex", "Dropped by ingest"} {
		if !strings.Contains(got, want) {
			t.Fatalf("hooks print missing %q in:\n%s", want, got)
		}
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := os.Stat(filepath.Join(home, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("hooks print should not mutate config, stat err=%v", err)
	}
}

func TestHooksPrintCmd_RejectsUnknownAgent(t *testing.T) {
	setupEnv(t)
	cmd := newHooksPrintCmd()
	if err := cmd.Flags().Set("agent", "unknown"); err != nil {
		t.Fatalf("set agent: %v", err)
	}
	execRunE(t, cmd, true)
}

func TestEvidenceCheckCmd_JSONAdvisory(t *testing.T) {
	root := t.TempDir()
	writeEvidenceFixture(t, root)
	cmd := newEvidenceCheckCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Flags().Set("root", root); err != nil {
		t.Fatalf("set root: %v", err)
	}
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	execRunE(t, cmd, false)

	var got struct {
		OK      bool `json:"ok"`
		Summary struct {
			PlaceholderCount int `json:"placeholder_count"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode evidence-check json: %v\n%s", err, out.String())
	}
	if !got.OK || got.Summary.PlaceholderCount != 0 {
		t.Fatalf("unexpected evidence-check result: %#v", got)
	}
}

func TestEvidenceCheckCmd_StrictFailsOnPlaceholders(t *testing.T) {
	root := t.TempDir()
	writeEvidenceFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, "docs", "v2-ga-acceptance-evidence.md"), []byte("# v2 GA Acceptance Evidence\n\n## Self-Retro\n\n- turn_duration: TODO quote\n\n## Agent-Operator\n\n- turn_duration: TODO quote\n\n## Team-Share\n\n- turn_duration: TODO quote\n"), 0o644); err != nil {
		t.Fatalf("write placeholder evidence: %v", err)
	}
	cmd := newEvidenceCheckCmd()
	if err := cmd.Flags().Set("root", root); err != nil {
		t.Fatalf("set root: %v", err)
	}
	if err := cmd.Flags().Set("strict", "true"); err != nil {
		t.Fatalf("set strict: %v", err)
	}
	execRunE(t, cmd, true)
}

func TestEvidenceReportCmd_CitesPlaceholdersAndNextAction(t *testing.T) {
	root := t.TempDir()
	writeEvidenceFixture(t, root)
	if err := os.WriteFile(filepath.Join(root, "docs", "v2-ga-acceptance-evidence.md"), []byte("# v2 GA Acceptance Evidence\n\n## Team-Share\n\n- turn_duration / turn 소요시간: TODO quote.\n- blocker_accumulation / blocker 누적시간: 2 blockers.\n- agent_completion / agent 완료율: 90%.\n- agent_blocker_freq / agent blocker 빈도: 0.1.\n"), 0o644); err != nil {
		t.Fatalf("write evidence doc: %v", err)
	}
	cmd := newEvidenceReportCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Flags().Set("root", root); err != nil {
		t.Fatalf("set root: %v", err)
	}
	if err := cmd.Flags().Set("persona", "team-share"); err != nil {
		t.Fatalf("set persona: %v", err)
	}
	execRunE(t, cmd, false)
	got := out.String()
	for _, want := range []string{"persona: team-share", "TODO quote", "evidence_bound_decisions"} {
		if !strings.Contains(got, want) {
			t.Fatalf("evidence-report missing %q in:\n%s", want, got)
		}
	}
}

func writeEvidenceFixture(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{"docs", filepath.Join(".omc", "specs")} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	evidenceDoc := `# v2 GA Acceptance Evidence

## Self-Retro

- turn_duration / turn 소요시간: 10m.
- blocker_accumulation / blocker 누적시간: 0.
- agent_completion / agent 완료율: 100%.
- agent_blocker_freq / agent blocker 빈도: 0.
- evidence_bound_decisions / evidence-bound decision: 4/4.
- Behavior change: [CHANGED-BY-METRIC: turn_duration]

## Agent-Operator

- turn_duration / turn 소요시간: 10m.
- blocker_accumulation / blocker 누적시간: 0.
- agent_completion / agent 완료율: 100%.
- agent_blocker_freq / agent blocker 빈도: 0.
- evidence_bound_decisions / evidence-bound decision: 4/4.
- Adversarial review: linked.

## Team-Share

- turn_duration / turn 소요시간: 10m.
- blocker_accumulation / blocker 누적시간: 0.
- agent_completion / agent 완료율: 100%.
- agent_blocker_freq / agent blocker 빈도: 0.
- evidence_bound_decisions / evidence-bound decision: 4/4.
- External recipient ack: acknowledged.
`
	files := map[string]string{
		"docs/v2-ga-acceptance-evidence.md":         evidenceDoc,
		"docs/phase-e-persona-recruitment.md":       "# recruitment\n",
		"docs/adversarial-review-framework.md":      "# adversarial review\n",
		"docs/e0-3-agent-tmux-sanity.md":            "| agent | status |\n| `claude` | pass |\n| `codex` | pass |\n| `gemini` | pass |\n",
		".omc/specs/v2-agent-operator-decisions.md": "# agent operator\n",
		".omc/specs/v2-team-share-report.md":        "# team share\n",
		".omc/specs/v2-adversarial-review.md":       "# adversarial review\n",
	}
	var alpha strings.Builder
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&alpha, "### 2026-05-%02d entry\n", i)
		if i == 1 {
			alpha.WriteString("[CHANGED-BY-METRIC: turn_duration]\n")
		}
	}
	files[".omc/specs/alpha-dogfood-log.md"] = alpha.String()
	for path, body := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
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

func TestAttentionCmd_Text(t *testing.T) {
	setupEnv(t)
	seedHandoffFixture(t)

	cmd := newAttentionCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	got := out.String()
	for _, want := range []string{"NTTS Flightlog attention", "주의 필요", "근거 없는 결정 유지", "다음:"} {
		if !strings.Contains(got, want) {
			t.Errorf("attention text missing %q in:\n%s", want, got)
		}
	}
}

func TestAttentionCmd_JSON(t *testing.T) {
	setupEnv(t)
	seedHandoffFixture(t)

	cmd := newAttentionCmd()
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	var got struct {
		Summary struct {
			Total int `json:"total"`
		} `json:"summary"`
		Items []struct {
			SourceType        string `json:"source_type"`
			Reason            string `json:"reason"`
			RecommendedAction string `json:"recommended_action"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("attention json parse: %v\n%s", err, out.String())
	}
	if got.Summary.Total == 0 || len(got.Items) == 0 {
		t.Fatalf("attention json should contain items: %#v", got)
	}
	if got.Items[0].SourceType == "" || got.Items[0].Reason == "" || got.Items[0].RecommendedAction == "" {
		t.Fatalf("attention item missing required fields: %#v", got.Items[0])
	}
}

func TestAttentionCmd_BadFlags(t *testing.T) {
	setupEnv(t)
	cmd := newAttentionCmd()
	if err := cmd.Flags().Set("format", "xml"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	execRunE(t, cmd, true)

	cmd = newAttentionCmd()
	if err := cmd.Flags().Set("window", "month"); err != nil {
		t.Fatalf("set window: %v", err)
	}
	execRunE(t, cmd, true)
}

func TestHandoffCmd_Text(t *testing.T) {
	setupEnv(t)
	seedHandoffFixture(t)

	cmd := newHandoffCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	got := out.String()
	for _, want := range []string{
		"NTTS Flightlog handoff",
		"현재 턴",
		"handoff 턴",
		"세션 인계 패킷 생성",
		"열린 블로커",
		"대기 중인 외부 확인",
		"근거 없는 결정",
		"근거 없는 결정 유지",
		"최근 근거",
		"추천 다음 행동",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("handoff text missing %q in:\n%s", want, got)
		}
	}
}

func TestHandoffCmd_JSON(t *testing.T) {
	setupEnv(t)
	seedHandoffFixture(t)

	cmd := newHandoffCmd()
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	var got struct {
		ActiveTurn struct {
			Title       string   `json:"title"`
			Intent      string   `json:"intent"`
			Constraints []string `json:"constraints"`
		} `json:"active_turn"`
		OpenBlockers             []handoffBlocker  `json:"open_blockers"`
		DecisionsNeedingEvidence []handoffDecision `json:"decisions_needing_evidence"`
		LatestEvidence           []handoffEvidence `json:"latest_evidence"`
		RecommendedNext          string            `json:"recommended_next"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("handoff json parse: %v\n%s", err, out.String())
	}
	if got.ActiveTurn.Title != "handoff 턴" {
		t.Fatalf("active_turn.title = %q", got.ActiveTurn.Title)
	}
	if got.ActiveTurn.Intent != "세션 인계 패킷 생성" {
		t.Fatalf("active_turn.intent = %q", got.ActiveTurn.Intent)
	}
	if len(got.ActiveTurn.Constraints) != 2 {
		t.Fatalf("constraints = %#v, want 2 items", got.ActiveTurn.Constraints)
	}
	if len(got.OpenBlockers) != 1 {
		t.Fatalf("open_blockers = %d, want 1", len(got.OpenBlockers))
	}
	if len(got.DecisionsNeedingEvidence) != 1 {
		t.Fatalf("decisions_needing_evidence = %d, want 1", len(got.DecisionsNeedingEvidence))
	}
	if len(got.LatestEvidence) != 1 {
		t.Fatalf("latest_evidence = %d, want 1", len(got.LatestEvidence))
	}
	if !strings.Contains(got.RecommendedNext, "블로커") {
		t.Fatalf("recommended_next = %q, want blocker-oriented next action", got.RecommendedNext)
	}
}

func TestHandoffCmd_BadFormat(t *testing.T) {
	setupEnv(t)
	cmd := newHandoffCmd()
	if err := cmd.Flags().Set("format", "xml"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	execRunE(t, cmd, true)
}

func TestHandoffCmd_CurrentSessionOnly(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"old session"}); err != nil {
		t.Fatalf("start old: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"old stale decision"}); err != nil {
		t.Fatalf("old decision: %v", err)
	}
	if err := newStartCmd().RunE(newStartCmd(), []string{"new session"}); err != nil {
		t.Fatalf("start new: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"new active decision"}); err != nil {
		t.Fatalf("new decision: %v", err)
	}

	cmd := newHandoffCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	got := out.String()
	if strings.Contains(got, "old stale decision") {
		t.Fatalf("handoff should not include old session decision:\n%s", got)
	}
	if !strings.Contains(got, "new active decision") {
		t.Fatalf("handoff missing current session decision:\n%s", got)
	}
}

func TestShareCmd_Markdown(t *testing.T) {
	setupEnv(t)
	seedHandoffFixture(t)
	if err := newTurnEndCmd().RunE(newTurnEndCmd(), []string{"handoff 검증 완료"}); err != nil {
		t.Fatalf("turn-end: %v", err)
	}

	cmd := newShareCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	got := out.String()
	for _, want := range []string{
		"NTTS Flightlog Share",
		"Summary / 요약",
		"Completed Turns / 완료 턴",
		"handoff 턴",
		"Active Blockers / 열린 블로커",
		"대기 중인 외부 확인",
		"Decisions And Evidence / 결정과 근거",
		"Metric Highlights / 메트릭 하이라이트",
		"Requested Review/Help / 요청 사항",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("share markdown missing %q in:\n%s", want, got)
		}
	}
}

func TestShareCmd_JSON(t *testing.T) {
	setupEnv(t)
	seedHandoffFixture(t)

	cmd := newShareCmd()
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)

	var got struct {
		Summary struct {
			Turns int `json:"turns"`
		} `json:"summary"`
		ActiveBlockers   []shareBlocker      `json:"active_blockers"`
		Decisions        []shareDecision     `json:"decisions"`
		MetricHighlights []map[string]string `json:"metric_highlights"`
		RequestedReview  []shareReviewItem   `json:"requested_review"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("share json parse: %v\n%s", err, out.String())
	}
	if got.Summary.Turns == 0 {
		t.Fatalf("share json summary missing turns: %#v", got.Summary)
	}
	if len(got.ActiveBlockers) == 0 || len(got.Decisions) == 0 || len(got.MetricHighlights) == 0 {
		t.Fatalf("share json missing sections: %#v", got)
	}
	if len(got.RequestedReview) == 0 {
		t.Fatalf("share json should include attention-backed review requests: %#v", got)
	}
}

func TestShareCmd_BadFlags(t *testing.T) {
	setupEnv(t)
	cmd := newShareCmd()
	if err := cmd.Flags().Set("format", "text"); err != nil {
		t.Fatalf("set format: %v", err)
	}
	execRunE(t, cmd, true)

	cmd = newShareCmd()
	if err := cmd.Flags().Set("window", "month"); err != nil {
		t.Fatalf("set window: %v", err)
	}
	execRunE(t, cmd, true)
}

func seedHandoffFixture(t *testing.T) {
	t.Helper()
	if err := newStartCmd().RunE(newStartCmd(), []string{"handoff 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newStatusCmd().RunE(newStatusCmd(), []string{"활성", "handoff 검증", "패킷 출력"}); err != nil {
		t.Fatalf("status: %v", err)
	}
	tsCmd := newTurnStartCmd()
	if err := tsCmd.Flags().Set("intent", "세션 인계 패킷 생성"); err != nil {
		t.Fatalf("set intent: %v", err)
	}
	if err := tsCmd.Flags().Set("constraints", "60줄 이하,JSON 지원"); err != nil {
		t.Fatalf("set constraints: %v", err)
	}
	if err := tsCmd.Flags().Set("done-when", "text/json 출력 검증"); err != nil {
		t.Fatalf("set done-when: %v", err)
	}
	if err := tsCmd.RunE(tsCmd, []string{"handoff 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"근거 있는 결정"}); err != nil {
		t.Fatalf("decision linked: %v", err)
	}
	linkCmd := newEvidenceCmd()
	if err := linkCmd.Flags().Set("link", "근거 있는 결정"); err != nil {
		t.Fatalf("set link: %v", err)
	}
	if err := linkCmd.RunE(linkCmd, []string{"테스트 근거"}); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"근거 없는 결정 유지"}); err != nil {
		t.Fatalf("decision unlinked: %v", err)
	}
	if err := newBlockerCmd().RunE(newBlockerCmd(), []string{"대기 중인 외부 확인", "응답 전까지 배포 보류"}); err != nil {
		t.Fatalf("blocker: %v", err)
	}
}

func TestDoctorCmd(t *testing.T) {
	setupEnv(t)
	cmd := newDoctorCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	execRunE(t, cmd, false)
	got := out.String()
	for _, want := range []string{"NTTS Flightlog doctor", "binary:", "db: ok", "migrations:", "tmux_pane:", "skill_wrappers:"} {
		if !strings.Contains(got, want) {
			t.Errorf("doctor output missing %q in:\n%s", want, got)
		}
	}
}

// TestViewCmd_AllModes exercises all one-shot view modes.
func TestViewCmd_AllModes(t *testing.T) {
	setupEnv(t)
	for _, mode := range []string{"flat", "turns", "decisions", "blockers", "report", "visual"} {
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
	if err := newTurnEndCmd().RunE(newTurnEndCmd(), []string{"명시적 턴 결과"}); err != nil {
		t.Fatalf("turn-end: %v", err)
	}
	var outcome *string
	s, err := openSession()
	if err != nil {
		t.Fatalf("openSession after turn-end: %v", err)
	}
	defer s.close()
	if err := s.store.QueryRow(`SELECT outcome FROM turns WHERE sequence_no = 1`).Scan(&outcome); err != nil {
		t.Fatalf("query turn outcome: %v", err)
	}
	if outcome == nil || *outcome != "명시적 턴 결과" {
		t.Fatalf("turn outcome = %v, want 명시적 턴 결과", outcome)
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

// TestEvidenceCmd_WithLink exercises decision lookup and link insertion.
func TestEvidenceCmd_WithLink(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"근거 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newTurnStartCmd().RunE(newTurnStartCmd(), []string{"근거 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"중요 아키텍처 결정"}); err != nil {
		t.Fatalf("decision: %v", err)
	}
	cmd := newEvidenceCmd()
	if err := cmd.Flags().Set("link", "아키텍처"); err != nil {
		t.Fatalf("set link: %v", err)
	}
	execRunE(t, cmd, false, "링크된 근거")

	s, err := openSession()
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	defer s.close()

	var linkCount int
	if err := s.store.QueryRow(`SELECT COUNT(*) FROM decision_evidence_links`).Scan(&linkCount); err != nil {
		t.Fatalf("query links: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("decision_evidence_links count = %d, want 1", linkCount)
	}
}

func TestEvidenceCmd_WithMissingLink(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"근거 누락 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	cmd := newEvidenceCmd()
	if err := cmd.Flags().Set("link", "없는 결정"); err != nil {
		t.Fatalf("set link: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"링크 실패 근거"}); err == nil {
		t.Fatal("evidence --link should fail when no decision matches")
	}
}

func TestEvidenceCmd_WithAmbiguousLink(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"근거 모호성 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"API 유지"}); err != nil {
		t.Fatalf("decision 1: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"API 변경"}); err != nil {
		t.Fatalf("decision 2: %v", err)
	}
	cmd := newEvidenceCmd()
	if err := cmd.Flags().Set("link", "API"); err != nil {
		t.Fatalf("set link: %v", err)
	}
	if err := cmd.RunE(cmd, []string{"모호한 근거"}); err == nil {
		t.Fatal("evidence --link should reject ambiguous decision matches")
	}
}

func TestDecisionSupersedeCmd_ByTitle(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"결정 대체 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newTurnStartCmd().RunE(newTurnStartCmd(), []string{"결정 대체 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"SQLite renderer 유지", "기존 결정"}); err != nil {
		t.Fatalf("decision: %v", err)
	}
	if err := newDecisionSupersedeCmd().RunE(newDecisionSupersedeCmd(), []string{"SQLite renderer", "Sidecar renderer 유지", "view 의미가 더 정확함"}); err != nil {
		t.Fatalf("decision-supersede: %v", err)
	}

	s, err := openSession()
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	defer s.close()

	var acceptedCount, supersededCount int
	if err := s.store.QueryRow(`SELECT COUNT(*) FROM decision_status WHERE status = 'accepted'`).Scan(&acceptedCount); err != nil {
		t.Fatalf("query accepted decisions: %v", err)
	}
	if err := s.store.QueryRow(`SELECT COUNT(*) FROM decision_status WHERE status = 'superseded'`).Scan(&supersededCount); err != nil {
		t.Fatalf("query superseded decisions: %v", err)
	}
	if acceptedCount != 1 || supersededCount != 1 {
		t.Fatalf("decision status counts accepted=%d superseded=%d, want 1/1", acceptedCount, supersededCount)
	}

	var rationale *string
	if err := s.store.QueryRow(`SELECT rationale FROM decision_status WHERE status = 'superseded'`).Scan(&rationale); err != nil {
		t.Fatalf("query supersede rationale: %v", err)
	}
	if rationale == nil || *rationale != "view 의미가 더 정확함" {
		t.Fatalf("rationale = %v, want view 의미가 더 정확함", rationale)
	}
}

func TestDecisionSupersedeCmd_AmbiguousOldDecision(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"결정 모호성 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"API 유지"}); err != nil {
		t.Fatalf("decision 1: %v", err)
	}
	if err := newDecisionCmd().RunE(newDecisionCmd(), []string{"API 변경"}); err != nil {
		t.Fatalf("decision 2: %v", err)
	}
	if err := newDecisionSupersedeCmd().RunE(newDecisionSupersedeCmd(), []string{"API", "API 동결"}); err == nil {
		t.Fatal("decision-supersede should reject ambiguous old decision matches")
	}
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

func TestBlockerResolveCmd_ByTitle(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"블로커 해결 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newTurnStartCmd().RunE(newTurnStartCmd(), []string{"블로커 해결 턴"}); err != nil {
		t.Fatalf("turn-start: %v", err)
	}
	if err := newBlockerCmd().RunE(newBlockerCmd(), []string{"빌드 실패", "테스트가 막힘"}); err != nil {
		t.Fatalf("blocker: %v", err)
	}
	if err := newBlockerResolveCmd().RunE(newBlockerResolveCmd(), []string{"빌드 실패", "의존성 복구"}); err != nil {
		t.Fatalf("blocker-resolve: %v", err)
	}

	s, err := openSession()
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	defer s.close()

	var status string
	var closedAt *string
	var accumulatedSeconds int64
	var resolutionNote *string
	if err := s.store.QueryRow(
		`SELECT status, closed_at, accumulated_seconds, resolution_note FROM blockers WHERE title = ?`,
		"빌드 실패",
	).Scan(&status, &closedAt, &accumulatedSeconds, &resolutionNote); err != nil {
		t.Fatalf("query blocker: %v", err)
	}
	if status != "resolved" {
		t.Fatalf("blocker status = %q, want resolved", status)
	}
	if closedAt == nil || *closedAt == "" {
		t.Fatal("resolved blocker should have closed_at")
	}
	if accumulatedSeconds < 0 {
		t.Fatalf("accumulatedSeconds = %d, want >= 0", accumulatedSeconds)
	}
	if resolutionNote == nil || *resolutionNote != "의존성 복구" {
		t.Fatalf("resolutionNote = %v, want 의존성 복구", resolutionNote)
	}

	var evidenceCount int
	if err := s.store.QueryRow(
		`SELECT COUNT(*) FROM entries WHERE kind = 'evidence' AND title LIKE '블로커 해결:%'`,
	).Scan(&evidenceCount); err != nil {
		t.Fatalf("query evidence count: %v", err)
	}
	if evidenceCount != 1 {
		t.Fatalf("resolution evidence count = %d, want 1", evidenceCount)
	}
}

func TestBlockerResolveCmd_AmbiguousTitle(t *testing.T) {
	setupEnv(t)
	if err := newStartCmd().RunE(newStartCmd(), []string{"블로커 모호성 세션"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := newBlockerCmd().RunE(newBlockerCmd(), []string{"API 실패"}); err != nil {
		t.Fatalf("blocker 1: %v", err)
	}
	if err := newBlockerCmd().RunE(newBlockerCmd(), []string{"API 지연"}); err != nil {
		t.Fatalf("blocker 2: %v", err)
	}
	if err := newBlockerResolveCmd().RunE(newBlockerResolveCmd(), []string{"API"}); err == nil {
		t.Fatal("blocker-resolve should reject ambiguous title matches")
	}
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
