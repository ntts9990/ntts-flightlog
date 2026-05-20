//go:build e2e

package e2e

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentDetectionIntegrationMatrix(t *testing.T) {
	cases := []struct {
		name    string
		env     []string
		want    string
		wantSig string
	}{
		{name: "claude", env: []string{"CLAUDE_DESKTOP_VERSION=0.7.5"}, want: "claude", wantSig: "claude"},
		{name: "codex", env: []string{"CODEX_HOME=/tmp/codex-home"}, want: "codex", wantSig: "codex"},
		{name: "gemini", env: []string{"GEMINI_API_KEY=test-key"}, want: "gemini", wantSig: "gemini"},
		{name: "priority", env: []string{"CLAUDE_DESKTOP_VERSION=0.7.5", "CODEX_HOME=/tmp/codex-home", "GEMINI_API_KEY=test-key"}, want: "claude", wantSig: "claude"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			worklogDir := filepath.Join(t.TempDir(), ".ntts-flightlog")
			env := append(buildEnv(worklogDir, t.TempDir()), tc.env...)
			out, err := runCmdCombinedEnv(env, "start", "agent "+tc.name)
			if err != nil {
				t.Fatalf("start: %v\n%s", err, out)
			}
			got := dbScalar(t, worklogDir, `SELECT COALESCE(agent_detected, '') FROM sessions LIMIT 1`)
			if got != tc.want {
				t.Fatalf("agent_detected = %q, want %q", got, tc.want)
			}
			stats, err := runCmdCombinedEnv(env, "agent-stats", "--format", "json")
			if err != nil {
				t.Fatalf("agent-stats: %v\n%s", err, stats)
			}
			if !strings.Contains(stats, tc.wantSig) {
				t.Fatalf("agent-stats missing %q:\n%s", tc.wantSig, stats)
			}
		})
	}
}

func TestAgentOverridePreservesDetectedValue(t *testing.T) {
	worklogDir := filepath.Join(t.TempDir(), ".ntts-flightlog")
	env := append(buildEnv(worklogDir, t.TempDir()), "CODEX_HOME=/tmp/codex-home")
	out, err := runCmdCombinedEnv(env, "--agent", "gemini", "start", "override")
	if err != nil {
		t.Fatalf("start --agent: %v\n%s", err, out)
	}
	got := dbScalar(t, worklogDir, `SELECT COALESCE(agent_detected, '') || '/' || COALESCE(agent_override, '') FROM sessions LIMIT 1`)
	if got != "codex/gemini" {
		t.Fatalf("detected/override = %q, want codex/gemini", got)
	}
}

func runCmdCombinedEnv(env []string, args ...string) (string, error) {
	cmd := exec.Command(binPath, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}
