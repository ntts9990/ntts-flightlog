//go:build integration

// Integration test: compare v2 ANSI rendering byte-for-byte against v1 awk
// binary on a curated minimal fixture.
//
// Run with: go test -tags integration ./internal/tui/views/ -v
//
// Requires:
//   - bin/ntts-flightlog (v1 reference binary) accessible at project root
//   - /bin/sh with awk available
//
// The fixture is crafted to isolate entry rendering (no status block, no
// section headers, no turn-start detail lines). These structural differences
// between v1/v2 are documented in styles.go; only entry-level ANSI codes are
// subject to the B-exit byte-equality gate.

package views

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// findProjectRoot walks up from the test binary's directory to locate the root
// (identified by the presence of go.mod).
func findProjectRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root (go.mod) not found")
		}
		dir = parent
	}
}

// entryOnlyLines filters an ANSI string to only lines that contain entry-level
// rendering (◆, ✓, !!, ▣, ▶, ■ glyphs or turn separator lines).
// This strips structural chrome (session/section headers, status blocks)
// so the remaining bytes are what the B-exit gate covers.
func entryOnlyLines(s string) []string {
	var result []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := sc.Text()
		// Include ANSI-colored lines that contain entry glyphs or turn borders.
		if strings.ContainsAny(line, "◆✓!!▣▶■────") && strings.Contains(line, "\033") {
			result = append(result, line)
		}
	}
	return result
}

// fixtureMarkdown is a minimal main.md with only entry lines — no status
// block, no section headers, no turn-start detail lines. v1 awk renders these
// identically to v2's WriteEntry output.
const fixtureMarkdown = `### 2026-05-20T10:00:00Z [entry] entry 테스트
### 2026-05-20T10:01:00Z [decision] decision 테스트
### 2026-05-20T10:02:00Z [evidence] evidence 테스트
### 2026-05-20T10:03:00Z [blocker] blocker 테스트
### 2026-05-20T10:04:00Z [mode] mode 테스트
`

// TestByteEquality_EntryLines runs the v1 awk renderer on the fixture and
// compares its entry-level output byte-for-byte with v2's WriteEntry output.
func TestByteEquality_EntryLines(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Skipf("project root not found: %v", err)
	}

	v1Binary := filepath.Join(root, "bin", "ntts-flightlog")
	if _, err := os.Stat(v1Binary); err != nil {
		t.Skipf("v1 binary not found at %s, skipping integration test", v1Binary)
	}

	// Write fixture to temp file.
	tmp, err := os.CreateTemp("", "byteq-fixture-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(fixtureMarkdown); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	// Run v1 awk on the fixture.
	cmd := exec.Command(v1Binary, "view", "flat")
	cmd.Env = append(os.Environ(), "WORKLOG_FILE="+tmp.Name(), "TURNS_DIR=/tmp/turns")
	v1Out, err := cmd.Output()
	if err != nil {
		t.Skipf("v1 binary execution failed: %v", err)
	}

	// Build v2 rendering of the same fixture via WriteEntry.
	var sb strings.Builder
	type row struct {
		ts, kind, title string
	}
	entries := []row{
		{"2026-05-20T10:00:00Z", "entry", "entry 테스트"},
		{"2026-05-20T10:01:00Z", "decision", "decision 테스트"},
		{"2026-05-20T10:02:00Z", "evidence", "evidence 테스트"},
		{"2026-05-20T10:03:00Z", "blocker", "blocker 테스트"},
		{"2026-05-20T10:04:00Z", "mode", "mode 테스트"},
	}
	for _, e := range entries {
		WriteEntry(&sb, e.ts, e.kind, e.title)
	}

	v1Lines := entryOnlyLines(string(v1Out))
	v2Lines := entryOnlyLines(sb.String())

	if len(v1Lines) == 0 {
		t.Fatal("v1 produced no entry-level lines — fixture may be malformed")
	}
	if len(v1Lines) != len(v2Lines) {
		t.Errorf("line count mismatch: v1=%d v2=%d\nv1 lines: %q\nv2 lines: %q",
			len(v1Lines), len(v2Lines), v1Lines, v2Lines)
		return
	}
	for i, v1line := range v1Lines {
		if v1line != v2Lines[i] {
			t.Errorf("line %d byte mismatch:\n  v1: %q\n  v2: %q", i+1, v1line, v2Lines[i])
		}
	}
	if !t.Failed() {
		t.Logf("BYTE-EQUAL: %d entry lines match v1 awk output exactly", len(v1Lines))
	}
}
