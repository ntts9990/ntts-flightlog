//go:build e2e

package e2e

// integration_test.go: multi-step integration flow tests for the flightlog
// binary. Uses the shared binPath set by TestMain in p0_test.go.

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestFullWorkflow exercises the core session pipeline:
// start → turn-start → entry → decision → evidence → report --format json
// Verifies that all 5 metric keys are present in the JSON output.
func TestFullWorkflow(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		return runFlightlogInDir(t, dir, args...)
	}

	// 1. Start a session (no TMUX → non-interactive, exits 0).
	if out, err := run("start", "통합 테스트 세션"); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}

	// 2. Start a turn.
	if out, err := run("turn-start", "첫 번째 턴"); err != nil {
		t.Fatalf("turn-start: %v\n%s", err, out)
	}

	// 3. Write a decision.
	if out, err := run("decision", "통합 테스트 결정"); err != nil {
		t.Fatalf("decision: %v\n%s", err, out)
	}

	// 4. Write evidence.
	if out, err := run("evidence", "통합 테스트 근거"); err != nil {
		t.Fatalf("evidence: %v\n%s", err, out)
	}

	// 5. Write a regular entry.
	if out, err := run("entry", "일반 작업 항목"); err != nil {
		t.Fatalf("entry: %v\n%s", err, out)
	}

	// 6. Report: verify valid JSON with all 5 metric sections.
	out, err := run("report", "--format", "json")
	if err != nil {
		t.Fatalf("report: %v\n%s", err, out)
	}

	var report map[string]any
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("report JSON invalid: %v\nraw:\n%s", err, out)
	}

	metrics, ok := report["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("report.metrics missing or wrong type: %v", report)
	}

	for _, k := range []string{
		"turn_duration", "blocker_accumulation",
		"agent_completion", "agent_blocker_freq", "evidence_bound_decisions",
	} {
		if _, ok := metrics[k]; !ok {
			t.Errorf("report.metrics missing key %q", k)
		}
	}
}

// TestViewAfterEntries verifies `flightlog view flat` reflects entries written
// in a workflow.
func TestViewAfterEntries(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		return runFlightlogInDir(t, dir, args...)
	}

	if out, err := run("start", "뷰 테스트"); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	if out, err := run("turn-start", "턴 1"); err != nil {
		t.Fatalf("turn-start: %v\n%s", err, out)
	}
	if out, err := run("entry", "뷰 확인 항목"); err != nil {
		t.Fatalf("entry: %v\n%s", err, out)
	}

	out, err := run("view", "flat")
	if err != nil {
		t.Fatalf("view flat: %v\n%s", err, out)
	}
	if !strings.Contains(out, "뷰 확인 항목") {
		t.Errorf("view flat output missing entry title:\n%s", out)
	}
}

// TestMigrateCommand verifies `flightlog migrate` exits 0.
func TestMigrateCommand(t *testing.T) {
	out, err := runFlightlog(t, "migrate")
	if err != nil {
		t.Fatalf("migrate exited non-zero: %v\noutput:\n%s", err, out)
	}
}

// TestTurnStartWithAnchor verifies anchor flags work end-to-end.
func TestTurnStartWithAnchor(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		return runFlightlogInDir(t, dir, args...)
	}

	if out, err := run("start", "앵커 테스트"); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}

	out, err := run("turn-start", "앵커 턴",
		"--intent", "DB 스키마만 수정",
		"--constraints", "git history만,DB 직접 쿼리 금지",
		"--done-when", "마이그레이션 완료")
	if err != nil {
		t.Fatalf("turn-start --intent: %v\n%s", err, out)
	}
	if !strings.Contains(out, "DB 스키마만 수정") {
		t.Errorf("anchor block missing from stdout:\n%s", out)
	}
}

// TestRefreshAnchor verifies `refresh-anchor` prints the intent to stdout.
func TestRefreshAnchor(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) (string, error) {
		return runFlightlogInDir(t, dir, args...)
	}

	if out, err := run("start", "앵커 테스트"); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	if out, err := run("turn-start", "앵커 턴", "--intent", "의도 확인"); err != nil {
		t.Fatalf("turn-start: %v\n%s", err, out)
	}

	out, err := run("refresh-anchor")
	if err != nil {
		t.Fatalf("refresh-anchor: %v\n%s", err, out)
	}
	if !strings.Contains(out, "의도 확인") {
		t.Errorf("refresh-anchor missing intent:\n%s", out)
	}
}

// TestReportWithAgentFlag verifies `report --agent claude` exits 0.
func TestReportWithAgentFlag(t *testing.T) {
	_, err := runFlightlog(t, "report", "--format", "json", "--agent", "claude")
	if err != nil {
		t.Fatalf("report --agent claude: %v", err)
	}
}
