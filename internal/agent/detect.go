// Package agent provides agent auto-detection via environment heuristics
// and process-tree inspection.
package agent

import (
	"fmt"
	"os"
	"strings"
)

// Detect returns the detected agent name and the signals that led to that
// determination. Returns ("unknown", signals_attempted) if no signal matches.
func Detect() (detected string, signals []string) {
	return detectWith(os.LookupEnv, readPPIDComm)
}

// detectWith is the testable core; callers inject an env lookup function and
// a ppid command resolver so unit tests can exercise every branch without
// touching the real environment or process tree.
func detectWith(
	lookupEnv func(string) (string, bool),
	ppidComm func() string,
) (string, []string) {
	var signals []string
	var detected string

	// Env heuristics evaluated in priority order: claude → codex → gemini.
	// All matching env vars are recorded as signals even if a winner is already found.
	envChecks := []struct {
		key   string
		agent string
	}{
		{"CLAUDE_DESKTOP_VERSION", "claude"},
		{"CODEX_HOME", "codex"},
		{"GEMINI_API_KEY", "gemini"},
	}
	for _, chk := range envChecks {
		if val, ok := lookupEnv(chk.key); ok {
			signals = append(signals, fmt.Sprintf("env:%s=%s", chk.key, val))
			if detected == "" {
				detected = chk.agent
			}
		}
	}

	// Process-tree heuristic: inspect the parent process command name.
	if comm := ppidComm(); comm != "" {
		if ppidAgent := matchPPIDComm(comm); ppidAgent != "" {
			signals = append(signals, fmt.Sprintf("ppid:%s", comm))
			if detected == "" {
				detected = ppidAgent
			}
		}
	}

	if detected == "" {
		detected = "unknown"
	}
	return detected, signals
}

// matchPPIDComm maps a process command name to an agent identifier.
// Returns "" if no known agent is recognised.
func matchPPIDComm(comm string) string {
	lower := strings.ToLower(strings.TrimSpace(comm))
	switch {
	case strings.Contains(lower, "claude"):
		return "claude"
	case strings.Contains(lower, "codex"):
		return "codex"
	case strings.Contains(lower, "gemini"):
		return "gemini"
	}
	return ""
}
