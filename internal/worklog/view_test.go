package worklog_test

// view_test.go: tests for RenderFlat, RenderTurns, FilterByKind,
// and renderMarkdownANSI (exercised indirectly).

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

// TestRenderFlat_NoFile prints a placeholder when main.md is absent.
func TestRenderFlat_NoFile(t *testing.T) {
	c := makeConfig(t)
	var buf bytes.Buffer
	if err := worklog.RenderFlat(c, &buf); err != nil {
		t.Fatalf("RenderFlat no file: %v", err)
	}
	out := buf.String()
	// v1 message contains "main.md" or "flightlog start" guidance.
	if !strings.Contains(out, "main.md") && !strings.Contains(out, "flightlog") {
		t.Errorf("RenderFlat no file: unexpected: %q", out)
	}
}

// TestRenderFlat_AllKinds exercises all entry kinds including anchors.
func TestRenderFlat_AllKinds(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "뷰 테스트"); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"entry", "decision", "evidence", "blocker", "mode"} {
		if err := worklog.AppendEntry(c, kind, kind+" 제목", kind+" 상세"); err != nil {
			t.Fatalf("AppendEntry %s: %v", kind, err)
		}
	}
	var buf bytes.Buffer
	if err := worklog.RenderFlat(c, &buf); err != nil {
		t.Fatalf("RenderFlat all kinds: %v", err)
	}
	out := buf.String()
	for _, kind := range []string{"entry", "decision", "evidence", "blocker", "mode"} {
		if !strings.Contains(out, kind+" 제목") {
			t.Errorf("RenderFlat: missing %s 제목 in output", kind)
		}
	}
}

// TestRenderFlat_TurnStartEnd exercises the turn-N-start and turn-N-end rendering.
func TestRenderFlat_TurnStartEnd(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	ts := worklog.Timestamp()
	// Build content with turn-start, a regular entry, an unknown kind, and turn-end.
	content := "# 턴 테스트\n\n업데이트: " + ts + "\n시작: " + ts + "\n\n## 작업 기록\n\n"
	content += "### " + ts + " [turn-1-start] 첫 번째 턴\n"
	content += "### " + ts + " [entry] 일반 항목\n"
	content += "### " + ts + " [unknown-kind] 알 수 없는 타입\n"
	content += "### " + ts + " [turn-1-end] 첫 번째 턴 완료\n"
	// Also include a line that exercises the dim/default branches.
	content += "일반 문단 텍스트\n"
	content += "  들여쓰기 텍스트\n"
	if err := os.WriteFile(c.MainMd, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := worklog.RenderFlat(c, &buf); err != nil {
		t.Fatalf("RenderFlat turns: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "turn-1-start") {
		t.Errorf("RenderFlat: missing turn-1-start; got:\n%s", out)
	}
	if !strings.Contains(out, "turn-1-end") {
		t.Errorf("RenderFlat: missing turn-1-end; got:\n%s", out)
	}
}

// TestRenderFlat_AnchorLines exercises the ⚓/📐/✅/─── anchor rendering paths.
func TestRenderFlat_AnchorLines(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	content := "# 앵커 테스트\n\n업데이트: 2026-05-20T10:00:00Z\n\n"
	content += "⚓ 의도: 테스트 의도\n"
	content += "📐 제약: 제약1\n"
	content += "✅ 완료조건: 완료\n"
	content += "─── ⚓ Turn Intent Anchor ───\n"
	if err := os.WriteFile(c.MainMd, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := worklog.RenderFlat(c, &buf); err != nil {
		t.Fatalf("RenderFlat anchor: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("RenderFlat anchor: empty output")
	}
}

// TestRenderTurns_Empty prints placeholder when no turn files exist.
func TestRenderTurns_Empty(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := worklog.RenderTurns(c, &buf); err != nil {
		t.Fatalf("RenderTurns empty: %v", err)
	}
	if !strings.Contains(buf.String(), "turn") {
		t.Errorf("RenderTurns empty: unexpected output: %q", buf.String())
	}
}

// TestRenderTurns_WithFiles renders multiple turn files in numeric order.
func TestRenderTurns_WithFiles(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if err := worklog.CreateTurnFile(c, 2, "두 번째 턴"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.CreateTurnFile(c, 1, "첫 번째 턴"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := worklog.RenderTurns(c, &buf); err != nil {
		t.Fatalf("RenderTurns: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "첫 번째 턴") {
		t.Error("RenderTurns: missing first turn title")
	}
	if !strings.Contains(out, "두 번째 턴") {
		t.Error("RenderTurns: missing second turn title")
	}
	// Verify ordering: turn 1 should appear before turn 2 in output.
	idx1 := strings.Index(out, "첫 번째 턴")
	idx2 := strings.Index(out, "두 번째 턴")
	if idx1 > idx2 {
		t.Error("RenderTurns: turn 1 should appear before turn 2")
	}
}

// TestFilterByKind_NoFile returns nil (no error) when main.md is absent.
func TestFilterByKind_NoFile(t *testing.T) {
	c := makeConfig(t)
	var buf bytes.Buffer
	if err := worklog.FilterByKind(c, "decision", &buf); err != nil {
		t.Fatalf("FilterByKind no file: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("FilterByKind no file: expected empty output, got %q", buf.String())
	}
}

// TestFilterByKind_Decision shows only decision entries.
func TestFilterByKind_Decision(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "필터 테스트"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.AppendEntry(c, "entry", "엔트리", ""); err != nil {
		t.Fatal(err)
	}
	if err := worklog.AppendEntry(c, "decision", "결정 항목", "근거 본문"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := worklog.FilterByKind(c, "decision", &buf); err != nil {
		t.Fatalf("FilterByKind decision: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "결정 항목") {
		t.Error("FilterByKind decision: missing decision entry")
	}
	if strings.Contains(out, "[entry]") {
		t.Error("FilterByKind decision: should not contain entry kind")
	}
}

// TestFilterByKind_Blocker filters blocker entries with detail.
func TestFilterByKind_Blocker(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "블로커 테스트"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.AppendEntry(c, "blocker", "블로커 항목", "차단 상세"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := worklog.FilterByKind(c, "blocker", &buf); err != nil {
		t.Fatalf("FilterByKind blocker: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "블로커 항목") {
		t.Error("FilterByKind blocker: missing blocker entry")
	}
}
