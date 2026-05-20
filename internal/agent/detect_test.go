package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// agentFixture describes one detection scenario loaded from a JSON file.
type agentFixture struct {
	Env                     map[string]string `json:"env"`
	PPIDChain               []string          `json:"ppid_chain"`
	ExpectedDetected        string            `json:"expected_detected"`
	ExpectedSignalsContains []string          `json:"expected_signals_contains"`
}

// TestDetectWithFixtures loads every *.json file under testdata/agent_fixtures
// and verifies that detectWith produces the expected detected value and that
// every string in expected_signals_contains appears as a substring of at least
// one element of the returned signals slice.
func TestDetectWithFixtures(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "agent_fixtures")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read fixture dir %s: %v", fixtureDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(fixtureDir, entry.Name()))
			if err != nil {
				t.Fatalf("read fixture %s: %v", entry.Name(), err)
			}
			var fix agentFixture
			if err := json.Unmarshal(data, &fix); err != nil {
				t.Fatalf("parse fixture %s: %v", entry.Name(), err)
			}

			// Inject mock env lookup and ppid resolver from fixture data.
			lookupEnv := func(key string) (string, bool) {
				val, ok := fix.Env[key]
				return val, ok
			}
			ppidComm := func() string {
				if len(fix.PPIDChain) > 0 {
					return fix.PPIDChain[0]
				}
				return ""
			}

			detected, signals := detectWith(lookupEnv, ppidComm)

			if detected != fix.ExpectedDetected {
				t.Errorf("detected = %q, want %q (signals: %v)", detected, fix.ExpectedDetected, signals)
			}

			for _, want := range fix.ExpectedSignalsContains {
				found := false
				for _, sig := range signals {
					if strings.Contains(sig, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("signals %v: expected an entry containing %q", signals, want)
				}
			}
		})
	}
}

// TestDetectPublicAPI verifies that the public Detect() function compiles and
// returns a non-empty detected value (the exact value depends on the test
// runner's environment, so we only check the invariant).
func TestDetectPublicAPI(t *testing.T) {
	detected, _ := Detect()
	if detected == "" {
		t.Error("Detect() returned empty detected string; want at least \"unknown\"")
	}
}
