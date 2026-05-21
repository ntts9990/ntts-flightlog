package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const RedactionVersion = "storage-redaction-2026-05-21"

var (
	envSecretPattern     = regexp.MustCompile(`(?i)\b([A-Z0-9_]*(?:API_KEY|TOKEN|SECRET|PASSWORD))\s*=\s*([^\s]+)`)
	bearerPattern        = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`)
	privateKeyPattern    = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
	basicAuthURLPattern  = regexp.MustCompile(`(?i)(https?://)[^/\s:@]+:[^/\s@]+@`)
	githubTokenPattern   = regexp.MustCompile(`\b(?:ghp|github_pat|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{12,}\b`)
	absoluteHomePattern  = regexp.MustCompile(`/Users/[^/\s]+/`)
	tempSecretPathRegexp = regexp.MustCompile(`(?i)(/tmp/|/private/tmp/)[^\s]*(secret|token|key)[^\s]*`)
)

type EventInput struct {
	Source     string
	EventName  string
	EventTime  string
	Summary    string
	Severity   string
	DedupeKey  string
	Command    string
	ExitCode   *int
	DurationMS *int64
	Payload    map[string]any
}

type SanitizedEvent struct {
	Source            string
	EventName         string
	EventTime         string
	Summary           string
	Severity          string
	DedupeKey         string
	CommandSummary    string
	ExitCode          *int
	DurationMS        *int64
	DroppedFieldCount int
	RedactionVersion  string
}

func ParseEventJSON(raw []byte, sourceFlag, eventFlag string) (EventInput, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return EventInput{}, fmt.Errorf("invalid JSON event")
	}
	source := firstString(sourceFlag, stringField(payload, "source"), stringField(payload, "agent"))
	eventName := firstString(eventFlag, stringField(payload, "event_name"), stringField(payload, "event"), stringField(payload, "name"))
	if source == "" {
		source = "generic"
	}
	if eventName == "" {
		return EventInput{}, fmt.Errorf("event name is required")
	}
	exitCode := intPtrField(payload, "exit_code")
	if exitCode == nil {
		exitCode = intPtrField(payload, "status_code")
	}
	duration := int64PtrField(payload, "duration_ms")
	return EventInput{
		Source:     source,
		EventName:  eventName,
		EventTime:  firstString(stringField(payload, "event_time"), stringField(payload, "timestamp"), stringField(payload, "time")),
		Summary:    firstString(stringField(payload, "summary"), stringField(payload, "message"), stringField(payload, "title")),
		Severity:   firstString(stringField(payload, "severity"), stringField(payload, "level")),
		DedupeKey:  firstString(stringField(payload, "dedupe_key"), stringField(payload, "id")),
		Command:    firstString(stringField(payload, "command"), stringField(payload, "cmd")),
		ExitCode:   exitCode,
		DurationMS: duration,
		Payload:    payload,
	}, nil
}

func SanitizeEvent(input EventInput, fallbackTime string) SanitizedEvent {
	summary := RedactText(input.Summary)
	if summary == "" {
		summary = RedactText(input.EventName)
	}
	commandSummary := SummarizeCommand(input.Command)
	eventTime := input.EventTime
	if eventTime == "" {
		eventTime = fallbackTime
	}
	dedupeKey := input.DedupeKey
	if dedupeKey == "" {
		dedupeKey = BuildDedupeKey(input.Source, input.EventName, summary, input.Command, input.ExitCode)
	} else {
		dedupeKey = BuildDedupeKey(input.Source, input.EventName, dedupeKey)
	}
	return SanitizedEvent{
		Source:            RedactText(input.Source),
		EventName:         RedactText(input.EventName),
		EventTime:         eventTime,
		Summary:           summary,
		Severity:          RedactText(input.Severity),
		DedupeKey:         dedupeKey,
		CommandSummary:    commandSummary,
		ExitCode:          input.ExitCode,
		DurationMS:        input.DurationMS,
		DroppedFieldCount: DroppedFieldCount(input.Payload),
		RedactionVersion:  RedactionVersion,
	}
}

func RedactText(value string) string {
	if value == "" {
		return ""
	}
	value = privateKeyPattern.ReplaceAllString(value, "[REDACTED:private-key]")
	value = envSecretPattern.ReplaceAllString(value, "$1=[REDACTED:secret]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [REDACTED:token]")
	value = githubTokenPattern.ReplaceAllString(value, "[REDACTED:token]")
	value = basicAuthURLPattern.ReplaceAllString(value, "$1[REDACTED:credentials]@")
	value = tempSecretPathRegexp.ReplaceAllString(value, "[REDACTED:path]")
	value = absoluteHomePattern.ReplaceAllString(value, "<home>/")
	return value
}

func SummarizeCommand(command string) string {
	command = strings.TrimSpace(RedactText(command))
	if command == "" {
		return ""
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	base := filepath.Base(fields[0])
	if len(fields) == 1 {
		return base
	}
	return base + " " + strings.Join(fields[1:min(len(fields), 4)], " ")
}

func BuildDedupeKey(parts ...any) string {
	var b strings.Builder
	for _, part := range parts {
		if part == nil {
			continue
		}
		b.WriteString(fmt.Sprint(part))
		b.WriteByte('\x00')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func DroppedFieldCount(payload map[string]any) int {
	allowed := map[string]bool{
		"source": true, "agent": true, "event_name": true, "event": true, "name": true,
		"event_time": true, "timestamp": true, "time": true,
		"summary": true, "message": true, "title": true,
		"severity": true, "level": true, "dedupe_key": true, "id": true,
		"command": true, "cmd": true, "exit_code": true, "status_code": true, "duration_ms": true,
	}
	count := 0
	for key := range payload {
		if !allowed[key] {
			count++
		}
	}
	return count
}

func firstString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func stringField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64, bool:
		return fmt.Sprint(v)
	default:
		return ""
	}
}

func intPtrField(payload map[string]any, key string) *int {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	switch v := value.(type) {
	case float64:
		n := int(v)
		return &n
	case int:
		return &v
	}
	return nil
}

func int64PtrField(payload map[string]any, key string) *int64 {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	switch v := value.(type) {
	case float64:
		n := int64(v)
		return &n
	case int64:
		return &v
	case int:
		n := int64(v)
		return &n
	}
	return nil
}
