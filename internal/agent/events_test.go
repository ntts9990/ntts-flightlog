package agent

import (
	"strings"
	"testing"
)

func TestRedactTextCoversSecretsAndHomePaths(t *testing.T) {
	input := "OPENAI_API_KEY=sk-secret Authorization: Bearer abc123 /Users/alice/repo/file.go https://u:p@example.com"
	got := RedactText(input)
	for _, forbidden := range []string{"sk-secret", "abc123", "/Users/alice", "u:p@"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("redacted text leaked %q in %q", forbidden, got)
		}
	}
	for _, want := range []string{"[REDACTED:secret]", "Bearer [REDACTED:token]", "<home>/repo/file.go", "[REDACTED:credentials]"} {
		if !strings.Contains(got, want) {
			t.Fatalf("redacted text missing %q in %q", want, got)
		}
	}
}

func TestParseAndSanitizeEventDropsRawPayloadFields(t *testing.T) {
	raw := []byte(`{
	  "source": "codex",
	  "event_name": "test.finished",
	  "summary": "go test ./... OPENAI_API_KEY=sk-secret",
	  "exit_code": 0,
	  "duration_ms": 1200,
	  "stdout": "private output"
	}`)
	event, err := ParseEventJSON(raw, "", "")
	if err != nil {
		t.Fatalf("ParseEventJSON: %v", err)
	}
	sanitized := SanitizeEvent(event, "2026-05-21T00:00:00Z")
	if strings.Contains(sanitized.Summary, "sk-secret") {
		t.Fatalf("summary leaked secret: %q", sanitized.Summary)
	}
	if sanitized.DroppedFieldCount != 1 {
		t.Fatalf("DroppedFieldCount = %d, want 1", sanitized.DroppedFieldCount)
	}
	if sanitized.ExitCode == nil || *sanitized.ExitCode != 0 {
		t.Fatalf("ExitCode = %#v, want 0", sanitized.ExitCode)
	}
}
