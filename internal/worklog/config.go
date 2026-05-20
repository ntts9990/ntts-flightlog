// Package worklog manages the .ntts-flightlog directory, main.md mirror, and
// per-turn markdown files that provide v1 backward compatibility.
package worklog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all resolved paths for a worklog directory.
type Config struct {
	Dir           string // e.g. .ntts-flightlog
	MainMd        string // Dir/main.md
	TurnsDir      string // Dir/turns
	DBPath        string // Dir/flightlog.db
	PaneFile      string // Dir/pane-id
	SessionStart  string // Dir/session-start-epoch  (v1 compat)
	TurnStart     string // Dir/turn-start-epoch     (v1 compat)
	TurnCounter   string // Dir/turn-counter         (v1 compat)
	ModeFile      string // Dir/mode                 (v1 compat)
	SessionIDFile string // Dir/session-id           (v2: active session UUID)
	TurnIDFile    string // Dir/turn-id              (v2: active turn UUID)
}

// DefaultConfig resolves the worklog directory from the environment or default.
func DefaultConfig() *Config {
	dir := defaultWorklogDir()
	return configFor(dir)
}

func configFor(dir string) *Config {
	return &Config{
		Dir:           dir,
		MainMd:        filepath.Join(dir, "main.md"),
		TurnsDir:      filepath.Join(dir, "turns"),
		DBPath:        filepath.Join(dir, "flightlog.db"),
		PaneFile:      filepath.Join(dir, "pane-id"),
		SessionStart:  filepath.Join(dir, "session-start-epoch"),
		TurnStart:     filepath.Join(dir, "turn-start-epoch"),
		TurnCounter:   filepath.Join(dir, "turn-counter"),
		ModeFile:      filepath.Join(dir, "mode"),
		SessionIDFile: filepath.Join(dir, "session-id"),
		TurnIDFile:    filepath.Join(dir, "turn-id"),
	}
}

func defaultWorklogDir() string {
	if v := os.Getenv("WORKLOG_DIR"); v != "" {
		return v
	}
	if _, err := os.Stat(".ntts-flightlog"); err == nil {
		return ".ntts-flightlog"
	}
	if _, err := os.Stat(".omx/worklog"); err == nil {
		return ".omx/worklog"
	}
	return ".ntts-flightlog"
}

// EnsureDir creates the worklog directory and turns subdirectory.
func (c *Config) EnsureDir() error {
	if err := os.MkdirAll(c.TurnsDir, 0o755); err != nil {
		return fmt.Errorf("worklog: mkdir %s: %w", c.TurnsDir, err)
	}
	return nil
}

// ReadFile reads a small state file and trims whitespace.
// Returns "" if the file does not exist.
func ReadFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// WriteFile writes content to a state file atomically (write to tmp, rename).
func WriteFile(path, content string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content+"\n"), 0o644); err != nil {
		return fmt.Errorf("writeFile %s: %w", path, err)
	}
	return os.Rename(tmp, path)
}

// CurrentTurnNumber returns the current turn number (0 if none).
func (c *Config) CurrentTurnNumber() int {
	s := ReadFile(c.TurnCounter)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

// NextTurnNumber increments and persists the turn counter, returning the new value.
func (c *Config) NextTurnNumber() (int, error) {
	n := c.CurrentTurnNumber() + 1
	if err := WriteFile(c.TurnCounter, strconv.Itoa(n)); err != nil {
		return 0, err
	}
	return n, nil
}

// TurnFilePath returns the absolute path of the markdown file for turn N.
func (c *Config) TurnFilePath(n int) string {
	return filepath.Join(c.TurnsDir, fmt.Sprintf("turn-%d.md", n))
}

// CurrentMode reads the mode file (returns "미지정" if absent).
func (c *Config) CurrentMode() string {
	if m := ReadFile(c.ModeFile); m != "" {
		return m
	}
	return "미지정"
}

// ActiveSessionID reads the v2 session-id file.
func (c *Config) ActiveSessionID() string {
	return ReadFile(c.SessionIDFile)
}

// ActiveTurnID reads the v2 turn-id file.
func (c *Config) ActiveTurnID() string {
	return ReadFile(c.TurnIDFile)
}
