package worklog_test

// config_test.go: unit tests for Config, DefaultConfig, defaultWorklogDir,
// EnsureDir, ReadFile, WriteFile, and turn/session state helpers.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

// makeConfig creates a Config backed by a fresh temp directory and sets
// WORKLOG_DIR so DefaultConfig returns it.
func makeConfig(t *testing.T) *worklog.Config {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WORKLOG_DIR", dir)
	return worklog.DefaultConfig()
}

// TestDefaultConfig_EnvOverride verifies WORKLOG_DIR is respected.
func TestDefaultConfig_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WORKLOG_DIR", dir)
	c := worklog.DefaultConfig()
	if c.Dir != dir {
		t.Errorf("Dir = %q, want %q", c.Dir, dir)
	}
	if c.DBPath != filepath.Join(dir, "flightlog.db") {
		t.Errorf("DBPath = %q", c.DBPath)
	}
	if c.MainMd != filepath.Join(dir, "main.md") {
		t.Errorf("MainMd = %q", c.MainMd)
	}
	if c.TurnsDir != filepath.Join(dir, "turns") {
		t.Errorf("TurnsDir = %q", c.TurnsDir)
	}
	if c.LanesDir != filepath.Join(dir, "lanes") {
		t.Errorf("LanesDir = %q", c.LanesDir)
	}
	if c.PaneFile != filepath.Join(dir, "pane-id") {
		t.Errorf("PaneFile = %q", c.PaneFile)
	}
	if c.SessionIDFile != filepath.Join(dir, "session-id") {
		t.Errorf("SessionIDFile = %q", c.SessionIDFile)
	}
	if c.TurnIDFile != filepath.Join(dir, "turn-id") {
		t.Errorf("TurnIDFile = %q", c.TurnIDFile)
	}
}

// TestDefaultConfig_Fallback verifies a non-empty Dir when env is unset.
func TestDefaultConfig_Fallback(t *testing.T) {
	t.Setenv("WORKLOG_DIR", "")
	c := worklog.DefaultConfig()
	if c.Dir == "" {
		t.Error("DefaultConfig fallback: Dir is empty")
	}
}

// TestEnsureDir creates the full directory tree.
func TestEnsureDir(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	for _, path := range []string{c.Dir, c.TurnsDir, c.LanesDir} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("EnsureDir: %q not created: %v", path, err)
		}
	}
}

// TestReadFile_Missing returns "" for a nonexistent file.
func TestReadFile_Missing(t *testing.T) {
	got := worklog.ReadFile("/nonexistent/definitely/not/here.txt")
	if got != "" {
		t.Errorf("ReadFile missing: got %q, want empty", got)
	}
}

func TestLaneStatePaths(t *testing.T) {
	c := makeConfig(t)
	if got := worklog.SafeLaneName("worker/a b"); got != "worker_a_b" {
		t.Fatalf("SafeLaneName = %q", got)
	}
	if got := c.LaneTurnIDFile(""); got != c.TurnIDFile {
		t.Fatalf("default lane turn-id file = %q, want %q", got, c.TurnIDFile)
	}
	want := filepath.Join(c.LanesDir, "worker_a_b", "turn-id")
	if got := c.LaneTurnIDFile("worker/a b"); got != want {
		t.Fatalf("lane turn-id file = %q, want %q", got, want)
	}
	if err := c.EnsureLaneDir("worker/a b"); err != nil {
		t.Fatalf("EnsureLaneDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(c.LanesDir, "worker_a_b")); err != nil {
		t.Fatalf("lane dir missing: %v", err)
	}
}

// TestWriteFile_ReadFile round-trip with whitespace trimming.
func TestWriteFile_ReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.txt")
	if err := worklog.WriteFile(path, "hello"); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if got := worklog.ReadFile(path); got != "hello" {
		t.Errorf("ReadFile = %q, want %q", got, "hello")
	}
}

// TestCurrentTurnNumber_Empty returns 0 when file is absent.
func TestCurrentTurnNumber_Empty(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if n := c.CurrentTurnNumber(); n != 0 {
		t.Errorf("CurrentTurnNumber empty = %d, want 0", n)
	}
}

// TestCurrentTurnNumber_Set parses the counter file.
func TestCurrentTurnNumber_Set(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if err := worklog.WriteFile(c.TurnCounter, "7"); err != nil {
		t.Fatal(err)
	}
	if n := c.CurrentTurnNumber(); n != 7 {
		t.Errorf("CurrentTurnNumber set = %d, want 7", n)
	}
}

// TestNextTurnNumber increments the counter on each call.
func TestNextTurnNumber(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	n1, err := c.NextTurnNumber()
	if err != nil {
		t.Fatalf("NextTurnNumber first: %v", err)
	}
	if n1 != 1 {
		t.Errorf("NextTurnNumber first = %d, want 1", n1)
	}
	n2, err := c.NextTurnNumber()
	if err != nil {
		t.Fatalf("NextTurnNumber second: %v", err)
	}
	if n2 != 2 {
		t.Errorf("NextTurnNumber second = %d, want 2", n2)
	}
}

// TestTurnFilePath verifies the turn-%d.md path format.
func TestTurnFilePath(t *testing.T) {
	c := makeConfig(t)
	p := c.TurnFilePath(5)
	want := filepath.Join(c.TurnsDir, "turn-5.md")
	if p != want {
		t.Errorf("TurnFilePath(5) = %q, want %q", p, want)
	}
}

// TestCurrentMode returns "미지정" when absent, actual value when present.
func TestCurrentMode(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if m := c.CurrentMode(); m != "미지정" {
		t.Errorf("CurrentMode default = %q, want 미지정", m)
	}
	if err := worklog.WriteFile(c.ModeFile, "solo"); err != nil {
		t.Fatal(err)
	}
	if m := c.CurrentMode(); m != "solo" {
		t.Errorf("CurrentMode set = %q, want solo", m)
	}
}

// TestActiveIDs verifies ActiveSessionID and ActiveTurnID read the correct files.
func TestActiveIDs(t *testing.T) {
	c := makeConfig(t)
	if err := c.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if s := c.ActiveSessionID(); s != "" {
		t.Errorf("ActiveSessionID initially = %q, want empty", s)
	}
	if id := c.ActiveTurnID(); id != "" {
		t.Errorf("ActiveTurnID initially = %q, want empty", id)
	}
	if err := worklog.WriteFile(c.SessionIDFile, "sess-abc"); err != nil {
		t.Fatal(err)
	}
	if s := c.ActiveSessionID(); s != "sess-abc" {
		t.Errorf("ActiveSessionID = %q, want sess-abc", s)
	}
	if err := worklog.WriteFile(c.TurnIDFile, "turn-xyz"); err != nil {
		t.Fatal(err)
	}
	if id := c.ActiveTurnID(); id != "turn-xyz" {
		t.Errorf("ActiveTurnID = %q, want turn-xyz", id)
	}
}
