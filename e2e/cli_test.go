//go:build e2e

package e2e

// cli_test.go: E2E subprocess tests for the flightlog binary.
// Run with: go test ./e2e/... -tags=e2e
//
// TestMain is defined in p0_test.go; the shared binary is in binPath.
// Use -update flag to regenerate golden snapshots:
//   go test ./e2e/... -tags=e2e -update

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// update regenerates golden snapshot files when true.
var update = flag.Bool("update", false, "update golden snapshot files")

// ── helpers ───────────────────────────────────────────────────────────────────

// runFlightlog runs the flightlog binary with given args in a fresh temp
// WORKLOG_DIR. Returns combined stdout+stderr and exit error (nil = exit 0).
func runFlightlog(t *testing.T, args ...string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "WORKLOG_DIR="+dir, "TMUX=")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runFlightlogInDir runs flightlog in a caller-supplied worklog directory (for
// multi-step flows that share state across commands).
func runFlightlogInDir(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(), "WORKLOG_DIR="+dir, "TMUX=")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// goldenFile returns the absolute path to a golden file under testdata/golden/.
func goldenFile(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata", "golden", name)
}

// assertGolden compares got against the named golden file.
// With -update it overwrites instead of comparing.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenFile(name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", name, err)
		}
		t.Logf("updated golden: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\n(re-run with -update to create)", name, err)
	}
	if string(want) != got {
		t.Errorf("golden mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s",
			name, want, got)
	}
}

// normalizeTimestamp replaces generated_at values with "TIMESTAMP" so JSON
// golden files are stable across runs.
var timestampRE = regexp.MustCompile(`"generated_at"\s*:\s*"[^"]*"`)

func normalizeTimestamp(s string) string {
	return timestampRE.ReplaceAllString(s, `"generated_at": "TIMESTAMP"`)
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestHelp verifies `flightlog --help` exits 0 and produces stable output.
func TestHelp(t *testing.T) {
	out, err := runFlightlog(t, "--help")
	if err != nil {
		t.Fatalf("--help exited non-zero: %v\noutput:\n%s", err, out)
	}
	assertGolden(t, "cli_help.txt", out)
}

// TestVersion verifies `flightlog --version` exits 0 and includes "flightlog".
func TestVersion(t *testing.T) {
	out, err := runFlightlog(t, "--version")
	if err != nil {
		t.Fatalf("--version exited non-zero: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "flightlog") {
		t.Errorf("--version output missing 'flightlog': %q", out)
	}
}

// TestReportJSONEmpty verifies `flightlog report --format json` on a fresh
// empty worklog exits 0, produces valid JSON, and matches the golden snapshot.
func TestReportJSONEmpty(t *testing.T) {
	out, err := runFlightlog(t, "report", "--format", "json")
	if err != nil {
		t.Fatalf("report --format json exited non-zero: %v\noutput:\n%s", err, out)
	}

	// Must be valid JSON.
	var v any
	if jsonErr := json.Unmarshal([]byte(out), &v); jsonErr != nil {
		t.Fatalf("report output is not valid JSON: %v\nraw:\n%s", jsonErr, out)
	}

	normalized := normalizeTimestamp(out)
	assertGolden(t, "cli_report_empty.json", normalized)
}

// TestReportTextEmpty verifies `flightlog report --format text` exits 0.
func TestReportTextEmpty(t *testing.T) {
	out, err := runFlightlog(t, "report", "--format", "text")
	if err != nil {
		t.Fatalf("report --format text exited non-zero: %v\noutput:\n%s", err, out)
	}
	if out == "" {
		t.Error("report --format text produced empty output")
	}
}

// TestReportBadFormat verifies `flightlog report --format bad` exits non-zero.
func TestReportBadFormat(t *testing.T) {
	_, err := runFlightlog(t, "report", "--format", "bad")
	if err == nil {
		t.Fatal("report --format bad: expected non-zero exit, got nil")
	}
}

// TestViewFlatEmpty verifies `flightlog view flat` on an empty worklog exits 0.
func TestViewFlatEmpty(t *testing.T) {
	_, err := runFlightlog(t, "view", "flat")
	if err != nil {
		t.Fatalf("view flat exited non-zero: %v", err)
	}
}

// TestPathCmd verifies `flightlog path` exits 0 and prints a directory path.
func TestPathCmd(t *testing.T) {
	dir := t.TempDir()
	out, err := runFlightlogInDir(t, dir, "path")
	if err != nil {
		t.Fatalf("path exited non-zero: %v\noutput:\n%s", err, out)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("path produced empty output")
	}
}

// TestUnknownCommand verifies an unknown subcommand exits non-zero.
func TestUnknownCommand(t *testing.T) {
	_, err := runFlightlog(t, "no-such-subcommand-xyz")
	if err == nil {
		t.Fatal("unknown command: expected non-zero exit, got nil")
	}
}

// TestReportWindowFlag verifies --window day|week|all all exit 0.
func TestReportWindowFlag(t *testing.T) {
	for _, w := range []string{"day", "week", "all"} {
		t.Run(w, func(t *testing.T) {
			_, err := runFlightlog(t, "report", "--format", "json", "--window", w)
			if err != nil {
				t.Fatalf("report --window %s exited non-zero: %v", w, err)
			}
		})
	}
}

// TestStartNoTmux verifies `flightlog start` exits 0 without TMUX env set.
func TestStartNoTmux(t *testing.T) {
	out, err := runFlightlog(t, "start", "E2E test session")
	if err != nil {
		t.Fatalf("start exited non-zero: %v\noutput:\n%s", err, out)
	}
}
