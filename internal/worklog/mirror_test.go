package worklog_test

// mirror_test.go: tests for Timestamp, EpochSeconds, FormatDuration,
// EnsureMainMd, AppendEntry, ReplaceStatus, CreateTurnFile, AppendToTurnFile,
// ReadMainMdLines.

import (
	"os"
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

// TestTimestamp returns a non-empty ISO 8601 string containing "T".
func TestTimestamp(t *testing.T) {
	ts := worklog.Timestamp()
	if ts == "" {
		t.Fatal("Timestamp returned empty string")
	}
	if !strings.Contains(ts, "T") {
		t.Errorf("Timestamp %q missing 'T' (not ISO 8601?)", ts)
	}
}

// TestEpochSeconds returns a non-empty numeric string ≥ 10 characters (unix epoch).
func TestEpochSeconds(t *testing.T) {
	s := worklog.EpochSeconds()
	if s == "" {
		t.Fatal("EpochSeconds returned empty string")
	}
	if len(s) < 9 {
		t.Errorf("EpochSeconds %q too short for a unix timestamp", s)
	}
}

// TestFormatDuration covers all three format branches (s / m+s / h+m+s).
func TestFormatDuration(t *testing.T) {
	cases := []struct {
		secs int64
		want string
	}{
		{0, "0s"},
		{1, "1s"},
		{59, "59s"},
		{60, "1m 00s"},
		{90, "1m 30s"},
		{3599, "59m 59s"},
		{3600, "1h 00m 00s"},
		{3661, "1h 01m 01s"},
		{7322, "2h 02m 02s"},
	}
	for _, tc := range cases {
		got := worklog.FormatDuration(tc.secs)
		if got != tc.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

// TestEnsureMainMd creates main.md with the given title.
func TestEnsureMainMd(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "테스트 세션"); err != nil {
		t.Fatalf("EnsureMainMd: %v", err)
	}
	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		t.Fatalf("read main.md: %v", err)
	}
	if !strings.Contains(string(data), "테스트 세션") {
		t.Error("main.md missing title '테스트 세션'")
	}
	// session-start-epoch must be created.
	if worklog.ReadFile(c.SessionStart) == "" {
		t.Error("session-start-epoch not created by EnsureMainMd")
	}
	// Calling again must be idempotent (no error, file not overwritten).
	if err := worklog.EnsureMainMd(c, "Different Title"); err != nil {
		t.Errorf("EnsureMainMd idempotent: %v", err)
	}
	// File still contains original title (idempotent: no overwrite).
	data2, _ := os.ReadFile(c.MainMd)
	if !strings.Contains(string(data2), "테스트 세션") {
		t.Error("EnsureMainMd idempotent: original title overwritten")
	}
}

// TestEnsureMainMd_EmptyTitle uses the default Korean title "작업 기록".
func TestEnsureMainMd_EmptyTitle(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, ""); err != nil {
		t.Fatalf("EnsureMainMd empty title: %v", err)
	}
	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "작업 기록") {
		t.Error("main.md missing default title '작업 기록'")
	}
}

// TestAppendEntry appends an entry and verifies main.md contains it.
func TestAppendEntry(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.AppendEntry(c, "entry", "항목 제목", ""); err != nil {
		t.Fatalf("AppendEntry: %v", err)
	}
	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[entry]") {
		t.Error("AppendEntry: main.md missing [entry] tag")
	}
	if !strings.Contains(string(data), "항목 제목") {
		t.Error("AppendEntry: main.md missing entry title")
	}
}

// TestAppendEntry_WithDetail verifies the detail body is written.
func TestAppendEntry_WithDetail(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.AppendEntry(c, "decision", "결정 항목", "상세 근거"); err != nil {
		t.Fatalf("AppendEntry with detail: %v", err)
	}
	data, _ := os.ReadFile(c.MainMd)
	if !strings.Contains(string(data), "상세 근거") {
		t.Error("AppendEntry: main.md missing detail body")
	}
}

// TestAppendEntry_WithActiveTurn appends to the current turn file as well.
func TestAppendEntry_WithActiveTurn(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	// Simulate an active turn: create turn-start-epoch + turn-counter + turn file.
	if err := worklog.WriteFile(c.TurnStart, worklog.EpochSeconds()); err != nil {
		t.Fatal(err)
	}
	if err := worklog.WriteFile(c.TurnCounter, "1"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.CreateTurnFile(c, 1, "첫 번째 턴"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.AppendEntry(c, "entry", "턴 항목", ""); err != nil {
		t.Fatalf("AppendEntry with active turn: %v", err)
	}
	turnData, err := os.ReadFile(c.TurnFilePath(1))
	if err != nil {
		t.Fatalf("read turn-1.md: %v", err)
	}
	if !strings.Contains(string(turnData), "턴 항목") {
		t.Errorf("Turn file missing entry: %s", turnData)
	}
}

// TestReplaceStatus rewrites the 현재 상태 section.
func TestReplaceStatus(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "교체 테스트"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.ReplaceStatus(c, "활성", "작업 중", "다음 단계"); err != nil {
		t.Fatalf("ReplaceStatus: %v", err)
	}
	data, _ := os.ReadFile(c.MainMd)
	s := string(data)
	for _, want := range []string{"- 상태: 활성", "- 초점: 작업 중", "- 다음: 다음 단계"} {
		if !strings.Contains(s, want) {
			t.Errorf("ReplaceStatus: missing %q in main.md", want)
		}
	}
}

// TestReplaceStatus_NoFocusNextStep verifies empty strings produce no line.
func TestReplaceStatus_NoFocusNextStep(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "테스트"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.ReplaceStatus(c, "완료", "", ""); err != nil {
		t.Fatalf("ReplaceStatus no focus: %v", err)
	}
	data, _ := os.ReadFile(c.MainMd)
	s := string(data)
	if !strings.Contains(s, "- 상태: 완료") {
		t.Error("ReplaceStatus: missing 상태: 완료")
	}
	if strings.Contains(s, "- 초점:") {
		t.Error("ReplaceStatus: should not include 초점 line when empty")
	}
}

// TestCreateTurnFile creates a per-turn markdown file.
func TestCreateTurnFile(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if err := worklog.CreateTurnFile(c, 3, "세 번째 턴"); err != nil {
		t.Fatalf("CreateTurnFile: %v", err)
	}
	data, err := os.ReadFile(c.TurnFilePath(3))
	if err != nil {
		t.Fatalf("read turn-3.md: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "Turn 3:") {
		t.Error("CreateTurnFile: missing 'Turn 3:'")
	}
	if !strings.Contains(s, "세 번째 턴") {
		t.Error("CreateTurnFile: missing title")
	}
}

// TestAppendToTurnFile appends content to an existing turn file.
func TestAppendToTurnFile(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if err := worklog.CreateTurnFile(c, 2, "두 번째 턴"); err != nil {
		t.Fatal(err)
	}
	if err := worklog.AppendToTurnFile(c, 2, "추가 내용\n"); err != nil {
		t.Fatalf("AppendToTurnFile: %v", err)
	}
	data, _ := os.ReadFile(c.TurnFilePath(2))
	if !strings.Contains(string(data), "추가 내용") {
		t.Error("AppendToTurnFile: content not appended")
	}
}

// TestReadMainMdLines parses main.md into individual lines.
func TestReadMainMdLines(t *testing.T) {
	c := makeConfig(t)
	if err := worklog.EnsureMainMd(c, "라인 파싱 테스트"); err != nil {
		t.Fatal(err)
	}
	lines, err := worklog.ReadMainMdLines(c)
	if err != nil {
		t.Fatalf("ReadMainMdLines: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("ReadMainMdLines returned empty slice")
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "라인 파싱 테스트") {
			found = true
		}
	}
	if !found {
		t.Error("ReadMainMdLines: title not found in any line")
	}
}
