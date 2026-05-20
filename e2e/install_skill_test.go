//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallSkillPlacementForThreeAgents(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	script := filepath.Join(repoRoot, "scripts", "install.sh")

	cases := []struct {
		flag string
		path string
	}{
		{"--codex", ".codex/skills/ntts-flightlog/SKILL.md"},
		{"--claude", ".claude/skills/ntts-flightlog/SKILL.md"},
		{"--gemini", ".gemini/skills/ntts-flightlog/SKILL.md"},
	}

	for _, tc := range cases {
		t.Run(tc.flag, func(t *testing.T) {
			prefix := t.TempDir()
			cmd := exec.Command("bash", script, tc.flag, "--no-cli")
			cmd.Env = append(os.Environ(), "INSTALL_PREFIX="+prefix)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("install %s: %v\n%s", tc.flag, err, out)
			}
			if _, err := os.Stat(filepath.Join(prefix, tc.path)); err != nil {
				t.Fatalf("expected skill at %s: %v\noutput:\n%s", tc.path, err, out)
			}
		})
	}

	t.Run("--all", func(t *testing.T) {
		prefix := t.TempDir()
		cmd := exec.Command("bash", script, "--all", "--no-cli")
		cmd.Env = append(os.Environ(), "INSTALL_PREFIX="+prefix)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("install --all: %v\n%s", err, out)
		}
		for _, rel := range []string{
			".codex/skills/ntts-flightlog/SKILL.md",
			".claude/skills/ntts-flightlog/SKILL.md",
			".gemini/skills/ntts-flightlog/SKILL.md",
		} {
			if _, err := os.Stat(filepath.Join(prefix, rel)); err != nil {
				t.Fatalf("expected skill at %s: %v\noutput:\n%s", rel, err, out)
			}
		}
	})
}
