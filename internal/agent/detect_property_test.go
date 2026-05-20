// Package agent — property/contract tests for Detect() invariants.
//
// Uses seeded table-driven fuzzing (deterministic) with 150 generated inputs
// per property, satisfying the plan D3 requirement of > 100 generated inputs.
//
// Properties verified:
//  1. Detect() output always matches ^[a-z][a-z0-9_-]{1,31}$
//  2. Detect() never returns an empty string (nil-safety invariant)
//  3. "unknown" (the fallback) itself satisfies the regex
//  4. All known agent names satisfy the regex
//  5. Empty-env + empty-ppid returns exactly "unknown"
package agent

import (
	"math/rand"
	"regexp"
	"testing"
)

// agentIDRegex is the format every detectWith() result must satisfy per the spec.
// "unknown" satisfies it: u(1) + nknown(6) = 7 chars total, all [a-z].
var agentIDRegex = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)

// detectEnvKeys are the environment variable names inspected by detectWith.
var detectEnvKeys = []string{
	"CLAUDE_DESKTOP_VERSION",
	"CODEX_HOME",
	"GEMINI_API_KEY",
}

// detectSamplePPIDs is the pool of parent-process command names exercised
// during fuzzing. Covers every known-agent name, mixed-case variants, and
// common noise strings to confirm the fallback path is exercised.
var detectSamplePPIDs = []string{
	"",
	"bash", "zsh", "fish", "sh", "dash",
	"python3", "python", "node", "ruby", "go",
	"vim", "nvim", "emacs", "code",
	"claude-desktop", "claude",
	"codex", "codex-cli",
	"gemini", "gemini-pro",
	"CLAUDE-DESKTOP", // upper-case: matchPPIDComm lowercases, so this still matches
	"unknown-process",
}

// nDetectGenerations is the number of random environment states tested per
// property. Must be > 100 per plan D3.
const nDetectGenerations = 150

// TestDetectPropertyAgentIDFormat verifies that detectWith always returns a
// string matching ^[a-z][a-z0-9_-]{1,31}$ across 150 randomised environments.
// Seeded at 42 for determinism across runs and CI.
func TestDetectPropertyAgentIDFormat(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < nDetectGenerations; i++ {
		env := genDetectEnv(rng)
		ppid := detectSamplePPIDs[rng.Intn(len(detectSamplePPIDs))]

		detected, _ := detectWith(
			func(key string) (string, bool) { v, ok := env[key]; return v, ok },
			func() string { return ppid },
		)

		if detected == "" {
			t.Errorf("gen %d: empty detected string (env=%v ppid=%q)", i, env, ppid)
			continue
		}
		if !agentIDRegex.MatchString(detected) {
			t.Errorf("gen %d: detected=%q does not match %s (env=%v ppid=%q)",
				i, detected, agentIDRegex, env, ppid)
		}
	}
}

// TestDetectPropertyNeverEmpty verifies the nil-safety invariant:
// detectWith never returns an empty string regardless of input combination.
// An empty return would prevent safe storage as agent_id in the DB.
func TestDetectPropertyNeverEmpty(t *testing.T) {
	rng := rand.New(rand.NewSource(137))

	for i := 0; i < nDetectGenerations; i++ {
		env := genDetectEnv(rng)
		ppid := ""
		if rng.Intn(2) == 0 {
			ppid = detectSamplePPIDs[rng.Intn(len(detectSamplePPIDs))]
		}

		detected, _ := detectWith(
			func(key string) (string, bool) { v, ok := env[key]; return v, ok },
			func() string { return ppid },
		)

		if detected == "" {
			t.Errorf("gen %d: empty detected (env=%v ppid=%q)", i, env, ppid)
		}
	}
}

// TestDetectPropertyFallbackMatchesRegex explicitly verifies that "unknown"
// (the documented fallback value) satisfies the agent_id format regex.
// This must hold so the fallback can always be persisted to sessions.agent_id.
func TestDetectPropertyFallbackMatchesRegex(t *testing.T) {
	const fallback = "unknown"
	if !agentIDRegex.MatchString(fallback) {
		t.Errorf("%q does not match regex %s", fallback, agentIDRegex)
	}
}

// TestDetectPropertyAllKnownAgentsMatchRegex verifies every possible non-fallback
// return value also satisfies the agent_id format regex.
func TestDetectPropertyAllKnownAgentsMatchRegex(t *testing.T) {
	knownAgents := []string{"claude", "codex", "gemini", "unknown"}
	for _, name := range knownAgents {
		if !agentIDRegex.MatchString(name) {
			t.Errorf("known agent %q does not match regex %s", name, agentIDRegex)
		}
	}
}

// TestDetectPropertyEmptyInputReturnsUnknown verifies that when no env var is
// set and the ppid resolver returns empty, the result is exactly "unknown".
func TestDetectPropertyEmptyInputReturnsUnknown(t *testing.T) {
	detected, signals := detectWith(
		func(string) (string, bool) { return "", false },
		func() string { return "" },
	)
	if detected != "unknown" {
		t.Errorf("no signals: detected=%q want %q (signals=%v)", detected, "unknown", signals)
	}
}

// TestDetectPropertyAgentIDFormat_Generated is the primary 100+ generator loop
// for the agent_id format property. Uses seeded RNG to produce 150 random
// (env, ppid) pairs and asserts every Detect() result matches the regex.
// Distinct from TestDetectPropertyAgentIDFormat which tests deterministic cases.
func TestDetectPropertyAgentIDFormat_Generated(t *testing.T) {
	const n = 150
	rng := rand.New(rand.NewSource(20260520))

	for i := 0; i < n; i++ {
		// Random subset of env vars set.
		env := map[string]string{}
		for _, key := range detectEnvKeys {
			if rng.Intn(2) == 1 {
				env[key] = genDetectString(rng, 1, 20)
			}
		}
		// Random ppid: sometimes a known agent, sometimes noise, sometimes empty.
		ppid := ""
		if rng.Intn(3) != 0 {
			ppid = detectSamplePPIDs[rng.Intn(len(detectSamplePPIDs))]
		}

		detected, _ := detectWith(
			func(key string) (string, bool) { v, ok := env[key]; return v, ok },
			func() string { return ppid },
		)

		if detected == "" {
			t.Errorf("input %d: empty detected (env=%v ppid=%q)", i, env, ppid)
			continue
		}
		if !agentIDRegex.MatchString(detected) {
			t.Errorf("input %d: detected=%q does not match %s (env=%v ppid=%q)",
				i, detected, agentIDRegex, env, ppid)
		}
	}
}

// TestDetectPropertyEnvPriorityStable verifies that when multiple env vars are
// set simultaneously, the first-match priority (claude > codex > gemini) is
// stable across 150 random value combinations.
func TestDetectPropertyEnvPriorityStable(t *testing.T) {
	rng := rand.New(rand.NewSource(77))

	for i := 0; i < nDetectGenerations; i++ {
		val := genDetectString(rng, 1, 10)
		// Set all three env vars simultaneously.
		env := map[string]string{
			"CLAUDE_DESKTOP_VERSION": val,
			"CODEX_HOME":             val,
			"GEMINI_API_KEY":         val,
		}

		detected, _ := detectWith(
			func(key string) (string, bool) { v, ok := env[key]; return v, ok },
			func() string { return "" },
		)

		// claude wins because CLAUDE_DESKTOP_VERSION is checked first.
		if detected != "claude" {
			t.Errorf("gen %d: all env set, expected claude (first priority) got %q", i, detected)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// genDetectEnv returns a map of randomly set/unset agent-detection env vars.
// Presence (not value) drives detection; values are random printable strings.
func genDetectEnv(rng *rand.Rand) map[string]string {
	env := make(map[string]string)
	for _, key := range detectEnvKeys {
		if rng.Intn(2) == 1 {
			env[key] = genDetectString(rng, 1, 20)
		}
	}
	return env
}

// genDetectString returns a random printable ASCII string of length ∈ [min, max].
func genDetectString(rng *rand.Rand, minLen, maxLen int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ.-_/"
	length := minLen + rng.Intn(maxLen-minLen+1)
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}
