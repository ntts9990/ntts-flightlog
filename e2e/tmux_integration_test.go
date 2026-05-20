//go:build e2e && tmux

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTmuxPaneLifecycle(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not on PATH")
	}

	session := fmt.Sprintf("flightlog-e2e-%d", time.Now().UnixNano())
	worklogDir := filepath.Join(t.TempDir(), ".ntts-flightlog")
	mustRunTmux(t, "new-session", "-d", "-s", session, "-x", "120", "-y", "32")
	t.Cleanup(func() { _ = exec.Command("tmux", "kill-session", "-t", session).Run() })

	sendShell(t, session, fmt.Sprintf("WORKLOG_DIR=%s %s start tmux-test", shellQuote(worklogDir), shellQuote(binPath)))
	paneIDPath := filepath.Join(worklogDir, "pane-id")
	paneID := waitForFile(t, paneIDPath)
	if paneID == "" {
		t.Fatal("pane-id was empty")
	}
	if !tmuxPaneExists(t, paneID) {
		t.Fatalf("pane %s not found after start", paneID)
	}

	out, err := runCmdCombinedEnv(buildEnv(worklogDir, t.TempDir()), "stop")
	if err != nil {
		t.Fatalf("stop: %v\n%s", err, out)
	}
	if _, err := os.Stat(paneIDPath); !os.IsNotExist(err) {
		t.Fatalf("pane-id still exists after stop: %v", err)
	}

	sendShell(t, session, fmt.Sprintf("WORKLOG_DIR=%s %s auto tmux-test", shellQuote(worklogDir), shellQuote(binPath)))
	recreated := waitForFile(t, paneIDPath)
	if recreated == "" || !tmuxPaneExists(t, recreated) {
		t.Fatalf("auto did not recreate pane, pane-id=%q", recreated)
	}
}

func TestTmuxTUIViewNonInteractive(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not on PATH")
	}
	worklogDir := filepath.Join(t.TempDir(), ".ntts-flightlog")
	env := buildEnv(worklogDir, t.TempDir())
	if out, err := runCmdCombinedEnv(env, "start", "tui render"); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	if out, err := runCmdCombinedEnv(env, "turn-start", "render turn"); err != nil {
		t.Fatalf("turn-start: %v\n%s", err, out)
	}
	if out, err := runCmdCombinedEnv(env, "entry", "render entry"); err != nil {
		t.Fatalf("entry: %v\n%s", err, out)
	}

	out, err := runCmdCombinedEnv(env, "view", "tui", "--noninteractive", "--view", "flat")
	if err != nil {
		t.Fatalf("view tui --noninteractive: %v\n%s", err, out)
	}
	if !strings.Contains(out, "render entry") {
		t.Fatalf("TUI flat output missing entry:\n%s", out)
	}
}

func mustRunTmux(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("tmux", args...).CombinedOutput(); err != nil {
		t.Fatalf("tmux %v: %v\n%s", args, err, out)
	}
}

func sendShell(t *testing.T, session, command string) {
	t.Helper()
	mustRunTmux(t, "send-keys", "-t", session+":0.0", command, "C-m")
}

func waitForFile(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
	return ""
}

func tmuxPaneExists(t *testing.T, paneID string) bool {
	t.Helper()
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}").Output()
	if err != nil {
		t.Fatalf("tmux list-panes: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == paneID {
			return true
		}
	}
	return false
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
