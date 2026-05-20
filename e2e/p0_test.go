//go:build e2e

// Package e2e contains end-to-end P0 smoke scenarios for ntts-flightlog v2.
// Each scenario runs in an isolated tmpdir with a fresh SQLite DB so tests
// are order-independent and race-clean.
//
// Run:
//
//	go test ./e2e -tags=e2e -run TestP0Scenarios -race -count=1 -v
package e2e

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
	"gopkg.in/yaml.v3"
)

// binPath is set once in TestMain and shared across all parallel sub-tests.
var binPath string

// TestMain builds the flightlog binary once before running all tests.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "flightlog-p0-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mktemp:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	binPath = filepath.Join(tmp, "flightlog")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", binPath,
		"github.com/ntts9990/ntts-flightlog/cmd/flightlog")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// ── YAML types ────────────────────────────────────────────────────────────────

// ScenarioFile is the top-level YAML document.
type ScenarioFile struct {
	Scenarios []Scenario `yaml:"scenarios"`
}

// Scenario describes one P0 test case.
type Scenario struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	// Setup is a list of CLI invocations run sequentially before Invoke.
	// Each element is a []string of args passed to the flightlog binary.
	Setup [][]string `yaml:"setup"`
	// Invoke is the list of CLI invocations whose combined stdout is checked.
	Invoke [][]string `yaml:"invoke"`

	// Dynamic invoke: query the DB for a scalar, substitute {0} in InvokeTemplate.
	PreInvokeDBQuery string   `yaml:"pre_invoke_db_query"`
	InvokeTemplate   []string `yaml:"invoke_template"`

	ExpectStdoutMatch string      `yaml:"expect_stdout_match"`
	ExpectStateMatch  *StateMatch `yaml:"expect_state_match"`
}

// StateMatch describes post-invoke assertions.
type StateMatch struct {
	// FileExists: path relative to the worklog dir that must exist.
	FileExists string `yaml:"file_exists"`
	// FileContainsAll: each string must appear in main.md.
	FileContainsAll []string `yaml:"file_contains_all"`
	// SQLite round-trip: run SQLiteQuery, compare first-column first-row to SQLiteExpect.
	SQLiteQuery  string `yaml:"sqlite_query"`
	SQLiteExpect string `yaml:"sqlite_expect"`
}

// ── Test runner ───────────────────────────────────────────────────────────────

func TestP0Scenarios(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "testdata", "p0_scenarios.yaml"))
	if err != nil {
		t.Fatalf("load p0_scenarios.yaml: %v", err)
	}

	var sf ScenarioFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		t.Fatalf("parse p0_scenarios.yaml: %v", err)
	}

	if got := len(sf.Scenarios); got != 26 {
		t.Fatalf("expected 26 scenarios in p0_scenarios.yaml, got %d", got)
	}

	for _, sc := range sf.Scenarios {
		sc := sc // capture loop variable
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			runScenario(t, sc)
		})
	}
}

// runScenario executes one scenario in a fresh isolated tmpdir.
func runScenario(t *testing.T, sc Scenario) {
	t.Helper()

	// Each scenario gets its own directory tree so there is zero shared state.
	tmpDir := t.TempDir()
	worklogDir := filepath.Join(tmpDir, ".ntts-flightlog")

	env := buildEnv(worklogDir, tmpDir)

	// --- Setup phase ---
	for i, args := range sc.Setup {
		out, err := runCmdCombined(env, args...)
		if err != nil {
			t.Fatalf("setup[%d] %v: %v\noutput:\n%s", i, args, err, out)
		}
	}

	// --- Invoke phase ---
	var stdoutBuf strings.Builder

	if sc.PreInvokeDBQuery != "" && len(sc.InvokeTemplate) > 0 {
		// Dynamic invoke: extract a scalar from the DB and substitute into the template.
		id := dbScalar(t, worklogDir, sc.PreInvokeDBQuery)
		args := make([]string, len(sc.InvokeTemplate))
		for i, a := range sc.InvokeTemplate {
			args[i] = strings.ReplaceAll(a, "{0}", id)
		}
		out, err := runCmdCombined(env, args...)
		if err != nil {
			t.Fatalf("invoke_template %v: %v\noutput:\n%s", args, err, out)
		}
		stdoutBuf.WriteString(out)
	} else {
		for i, args := range sc.Invoke {
			out, err := runCmdCombined(env, args...)
			if err != nil {
				t.Fatalf("invoke[%d] %v: %v\noutput:\n%s", i, args, err, out)
			}
			stdoutBuf.WriteString(out)
		}
	}

	stdout := stdoutBuf.String()

	// --- Assertions ---
	if sc.ExpectStdoutMatch != "" {
		if !strings.Contains(stdout, sc.ExpectStdoutMatch) {
			t.Errorf("[%s] stdout does not contain %q\nfull output:\n%s",
				sc.Name, sc.ExpectStdoutMatch, stdout)
		}
	}

	sm := sc.ExpectStateMatch
	if sm == nil {
		return
	}

	if sm.FileExists != "" {
		path := filepath.Join(worklogDir, sm.FileExists)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("[%s] file_exists: %s not found", sc.Name, path)
		}
	}

	if len(sm.FileContainsAll) > 0 {
		mainMd := filepath.Join(worklogDir, "main.md")
		content, err := os.ReadFile(mainMd)
		if err != nil {
			t.Fatalf("[%s] read main.md: %v", sc.Name, err)
		}
		for _, want := range sm.FileContainsAll {
			if !strings.Contains(string(content), want) {
				t.Errorf("[%s] main.md does not contain %q\n--- main.md ---\n%s",
					sc.Name, want, content)
			}
		}
	}

	if sm.SQLiteQuery != "" {
		got := dbScalar(t, worklogDir, sm.SQLiteQuery)
		if got != sm.SQLiteExpect {
			t.Errorf("[%s] sqlite:\n  query:  %s\n  want:   %q\n  got:    %q",
				sc.Name, sm.SQLiteQuery, sm.SQLiteExpect, got)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// buildEnv constructs a minimal, reproducible environment for the subprocess.
// TMUX is intentionally absent so `start` uses the non-interactive path.
// Agent-detection env vars are absent so detection falls through to "unknown".
func buildEnv(worklogDir, homeDir string) []string {
	env := []string{
		"WORKLOG_DIR=" + worklogDir,
		"HOME=" + homeDir,
		"PATH=" + os.Getenv("PATH"),
	}
	// Preserve GOPATH / GOROOT in case the subprocess shells out to `go`
	// (e.g. self-upgrade detection), but we don't need them for the binary itself.
	if v := os.Getenv("GOPATH"); v != "" {
		env = append(env, "GOPATH="+v)
	}
	if v := os.Getenv("GOROOT"); v != "" {
		env = append(env, "GOROOT="+v)
	}
	if v := os.Getenv("TMPDIR"); v != "" {
		env = append(env, "TMPDIR="+v)
	} else if v := os.Getenv("TEMP"); v != "" {
		env = append(env, "TEMP="+v)
	}
	return env
}

// runCmdCombined runs flightlog with the given args and returns combined stdout+stderr.
func runCmdCombined(env []string, args ...string) (string, error) {
	cmd := exec.Command(binPath, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// dbScalar opens the worklog SQLite DB, runs query, and returns the first
// column of the first row as a string.  Fails the test if the DB is
// unreachable or the query returns no rows.
func dbScalar(t *testing.T, worklogDir, query string) string {
	t.Helper()
	dbPath := filepath.Join(worklogDir, "flightlog.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("dbScalar: open %s: %v", dbPath, err)
	}
	defer db.Close()

	var result string
	if err := db.QueryRow(query).Scan(&result); err != nil {
		t.Fatalf("dbScalar: query=%q: %v", query, err)
	}
	return result
}
