package migrate_test

// parsedir_test.go: black-box tests for the exported ParseDir function.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/migrate"
)

// TestParseDir_Full verifies ParseDir reads all v1 directory files correctly:
// session-start-epoch, mode, turn-counter, and main.md.
func TestParseDir_Full(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	pdWriteFile(t, filepath.Join(dir, "session-start-epoch"), "1716192000")
	pdWriteFile(t, filepath.Join(dir, "mode"), "plan")
	pdWriteFile(t, filepath.Join(dir, "turn-counter"), "3")
	pdWriteFile(t, filepath.Join(dir, "main.md"),
		"## 작업 기록\n\n### 2026-05-20T00:00:00Z [entry] 테스트\n세부 내용\n\n")

	data, err := migrate.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if data.SessionStartEpoch != 1716192000 {
		t.Errorf("SessionStartEpoch: got %d, want 1716192000", data.SessionStartEpoch)
	}
	if data.Mode != "plan" {
		t.Errorf("Mode: got %q, want %q", data.Mode, "plan")
	}
	if data.TurnCount != 3 {
		t.Errorf("TurnCount: got %d, want 3", data.TurnCount)
	}
	if len(data.Records) == 0 {
		t.Error("Records should be non-empty for the test main.md")
	}
}

// TestParseDir_Defaults verifies ParseDir applies correct defaults for missing
// optional files (mode → "solo", turn-counter → 0, session-start-epoch → 0).
func TestParseDir_Defaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Only main.md required; omit all optional state files.
	pdWriteFile(t, filepath.Join(dir, "main.md"),
		"### 2026-05-20T00:00:00Z [entry] 최소 항목\n\n")

	data, err := migrate.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if data.SessionStartEpoch != 0 {
		t.Errorf("SessionStartEpoch: got %d, want 0 (default)", data.SessionStartEpoch)
	}
	if data.Mode != "solo" {
		t.Errorf("Mode: got %q, want %q (default)", data.Mode, "solo")
	}
	if data.TurnCount != 0 {
		t.Errorf("TurnCount: got %d, want 0 (default)", data.TurnCount)
	}
}

// TestParseDir_MissingMainMD verifies ParseDir returns an error when main.md
// is absent (the only required file).
func TestParseDir_MissingMainMD(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// No main.md.

	_, err := migrate.ParseDir(dir)
	if err == nil {
		t.Fatal("ParseDir with missing main.md: expected error, got nil")
	}
}

// TestParseDir_InvalidEpoch verifies ParseDir returns an error for a
// non-numeric session-start-epoch file.
func TestParseDir_InvalidEpoch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pdWriteFile(t, filepath.Join(dir, "session-start-epoch"), "not-a-number")
	pdWriteFile(t, filepath.Join(dir, "main.md"),
		"### 2026-05-20T00:00:00Z [entry] 항목\n\n")

	_, err := migrate.ParseDir(dir)
	if err == nil {
		t.Fatal("ParseDir with invalid epoch: expected error, got nil")
	}
}

// TestParseDir_WithTurns verifies ParseDir parses turn-start/end records.
func TestParseDir_WithTurns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pdWriteFile(t, filepath.Join(dir, "main.md"),
		"### 2026-05-20T00:00:00Z [turn-1-start] 첫 번째 턴\n시작.\n\n"+
			"### 2026-05-20T01:00:00Z [entry] 항목\n\n"+
			"### 2026-05-20T02:00:00Z [turn-1-end] 첫 번째 턴\n소요 시간: 2m 0s.\n\n")

	data, err := migrate.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(data.Records) != 3 {
		t.Errorf("Records count: got %d, want 3", len(data.Records))
	}
}

// TestParseDir_InvalidTurnCounter verifies ParseDir returns an error for a
// non-numeric turn-counter file.
func TestParseDir_InvalidTurnCounter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pdWriteFile(t, filepath.Join(dir, "turn-counter"), "bad")
	pdWriteFile(t, filepath.Join(dir, "main.md"),
		"### 2026-05-20T00:00:00Z [entry] 항목\n\n")

	_, err := migrate.ParseDir(dir)
	if err == nil {
		t.Fatal("ParseDir with invalid turn-counter: expected error, got nil")
	}
}

// pdWriteFile is a helper that creates a file with the given text content.
func pdWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("pdWriteFile %s: %v", path, err)
	}
}
