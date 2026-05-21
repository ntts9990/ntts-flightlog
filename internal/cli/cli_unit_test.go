package cli

// cli_unit_test.go: white-box unit tests for pure helper functions and
// DB-backed session operations in internal/cli. These tests do NOT invoke
// cobra commands — they call unexported helpers directly to maximise coverage
// without relying on the global rootCmd state.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

// ── nullStr ────────────────────────────────────────────────────────────────

func TestNullStr_Empty(t *testing.T) {
	if nullStr("") != nil {
		t.Error("nullStr(\"\") should return nil")
	}
}

func TestNullStr_NonEmpty(t *testing.T) {
	s := nullStr("hello")
	if s == nil {
		t.Fatal("nullStr(\"hello\") should return non-nil *string")
	}
	if *s != "hello" {
		t.Errorf("nullStr(\"hello\") = %q, want hello", *s)
	}
}

// ── now ────────────────────────────────────────────────────────────────────

func TestNow_ReturnsISO8601(t *testing.T) {
	ts := now()
	if ts == "" {
		t.Fatal("now() returned empty string")
	}
	if !strings.Contains(ts, "T") {
		t.Errorf("now() = %q, not ISO 8601 (missing 'T')", ts)
	}
	if len(ts) < 19 {
		t.Errorf("now() = %q, too short for ISO 8601", ts)
	}
}

// ── versionString / SetVersion ────────────────────────────────────────────

func TestVersionString_Empty(t *testing.T) {
	// When appVersion is empty, versionString should still not panic.
	old := appVersion
	appVersion = ""
	defer func() { appVersion = old }()

	got := versionString()
	if !strings.Contains(got, "flightlog") {
		t.Errorf("versionString empty = %q, want 'flightlog …'", got)
	}
}

func TestSetVersion(t *testing.T) {
	old := appVersion
	defer func() { appVersion = old }()

	SetVersion("1.2.3")
	if appVersion != "1.2.3" {
		t.Errorf("SetVersion: appVersion = %q, want 1.2.3", appVersion)
	}
	if got := versionString(); !strings.Contains(got, "1.2.3") {
		t.Errorf("versionString after SetVersion = %q, want to contain 1.2.3", got)
	}
}

// ── fmtDurationSec ─────────────────────────────────────────────────────────

func TestFmtDurationSec(t *testing.T) {
	cases := []struct {
		secs int64
		want string
	}{
		{0, "0s"},
		{-5, "0s"},
		{1, "1s"},
		{59, "59s"},
		{60, "1m 0s"},
		{90, "1m 30s"},
		{3600, "1h 0m 0s"},
		{3661, "1h 1m 1s"},
		{7322, "2h 2m 2s"},
	}
	for _, tc := range cases {
		got := fmtDurationSec(tc.secs)
		if got != tc.want {
			t.Errorf("fmtDurationSec(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

// ── fmtDurationMS ──────────────────────────────────────────────────────────

func TestFmtDurationMS(t *testing.T) {
	cases := []struct {
		ms   int64
		want string
	}{
		{0, "0s"},
		{500, "0s"}, // < 1s
		{1000, "1s"},
		{60000, "1m 0s"},
		{3600000, "1h 0m 0s"},
	}
	for _, tc := range cases {
		got := fmtDurationMS(tc.ms)
		if got != tc.want {
			t.Errorf("fmtDurationMS(%d) = %q, want %q", tc.ms, got, tc.want)
		}
	}
}

// ── buildAnchorBlock ───────────────────────────────────────────────────────

func TestBuildAnchorBlock_AllEmpty(t *testing.T) {
	got := buildAnchorBlock("", nil, "")
	if got != "" {
		t.Errorf("buildAnchorBlock all empty: expected \"\", got %q", got)
	}
}

func TestBuildAnchorBlock_IntentOnly(t *testing.T) {
	got := buildAnchorBlock("DB 스키마만", nil, "")
	if !strings.Contains(got, "⚓ 의도: DB 스키마만") {
		t.Errorf("buildAnchorBlock intent only: missing intent in %q", got)
	}
	if strings.Contains(got, "📐") {
		t.Error("buildAnchorBlock intent only: should not contain constraints line")
	}
}

func TestBuildAnchorBlock_AllFields(t *testing.T) {
	got := buildAnchorBlock("테스트 의도", []string{"제약1", "제약2"}, "완료 시점")
	for _, want := range []string{"⚓ 의도: 테스트 의도", "📐 제약: 제약1 | 제약2", "✅ 완료조건: 완료 시점"} {
		if !strings.Contains(got, want) {
			t.Errorf("buildAnchorBlock all fields: missing %q in %q", want, got)
		}
	}
}

func TestBuildAnchorBlock_ConstraintsOnly(t *testing.T) {
	got := buildAnchorBlock("", []string{"c1"}, "")
	if !strings.Contains(got, "📐 제약: c1") {
		t.Errorf("buildAnchorBlock constraints only: got %q", got)
	}
}

func TestBuildAnchorBlock_DoneWhenOnly(t *testing.T) {
	got := buildAnchorBlock("", nil, "마이그레이션 완료")
	if !strings.Contains(got, "✅ 완료조건: 마이그레이션 완료") {
		t.Errorf("buildAnchorBlock doneWhen only: got %q", got)
	}
}

// ── renderAnchorBlock ──────────────────────────────────────────────────────

func TestRenderAnchorBlock_AllInvalid(t *testing.T) {
	row := anchorRow{}
	if got := renderAnchorBlock(row); got != "" {
		t.Errorf("renderAnchorBlock all invalid: expected \"\", got %q", got)
	}
}

func TestRenderAnchorBlock_IntentOnly(t *testing.T) {
	row := anchorRow{
		id:     "turn-1",
		intent: sql.NullString{String: "의도", Valid: true},
	}
	got := renderAnchorBlock(row)
	if !strings.Contains(got, "의도") {
		t.Errorf("renderAnchorBlock intent: missing intent in %q", got)
	}
	if !strings.Contains(got, "Turn Intent Anchor") {
		t.Errorf("renderAnchorBlock intent: missing header in %q", got)
	}
}

func TestRenderAnchorBlock_AllFields(t *testing.T) {
	row := anchorRow{
		id:              "turn-1",
		intent:          sql.NullString{String: "전체 의도", Valid: true},
		constraintsJSON: sql.NullString{String: `["제약A","제약B"]`, Valid: true},
		doneWhen:        sql.NullString{String: "완료 기준", Valid: true},
	}
	got := renderAnchorBlock(row)
	for _, want := range []string{"전체 의도", "📐 제약: 제약A | 제약B", "✅ 완료조건: 완료 기준"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderAnchorBlock all fields: missing %q in %q", want, got)
		}
	}
}

func TestRenderAnchorBlock_InvalidJSON(t *testing.T) {
	// Invalid JSON constraints → constraints line omitted, no panic.
	row := anchorRow{
		id:              "turn-1",
		intent:          sql.NullString{String: "의도", Valid: true},
		constraintsJSON: sql.NullString{String: "not-json", Valid: true},
		doneWhen:        sql.NullString{String: "기준", Valid: true},
	}
	got := renderAnchorBlock(row)
	if !strings.Contains(got, "의도") {
		t.Errorf("renderAnchorBlock invalid JSON: missing intent in %q", got)
	}
	if strings.Contains(got, "📐") {
		t.Error("renderAnchorBlock invalid JSON: should not contain constraints line")
	}
}

func TestRenderAnchorBlock_OnlyDoneWhen(t *testing.T) {
	row := anchorRow{
		doneWhen: sql.NullString{String: "검증 완료", Valid: true},
	}
	got := renderAnchorBlock(row)
	if !strings.Contains(got, "✅ 완료조건: 검증 완료") {
		t.Errorf("renderAnchorBlock doneWhen: got %q", got)
	}
}

// ── formatText ─────────────────────────────────────────────────────────────

func TestFormatText_EmptySnapshot(t *testing.T) {
	snap := &metrics.Snapshot{}
	got := formatText(snap, "all", "")
	if !strings.Contains(got, "NTTS Flightlog") {
		t.Error("formatText empty: missing header")
	}
	if !strings.Contains(got, "데이터 없음") {
		t.Error("formatText empty: missing '데이터 없음' placeholder")
	}
	if !strings.Contains(got, "결정 없음") {
		t.Error("formatText empty: missing '결정 없음' for EvidenceBound")
	}
}

func TestFormatText_WindowLabels(t *testing.T) {
	snap := &metrics.Snapshot{}
	cases := []struct {
		window string
		label  string
	}{
		{"day", "오늘"},
		{"week", "이번 주"},
		{"all", "전체"},
		{"", "전체"},
	}
	for _, tc := range cases {
		got := formatText(snap, tc.window, "")
		if !strings.Contains(got, tc.label) {
			t.Errorf("formatText window=%q: missing label %q", tc.window, tc.label)
		}
	}
}

func TestFormatText_AgentLabel(t *testing.T) {
	snap := &metrics.Snapshot{}
	got := formatText(snap, "all", "claude")
	if !strings.Contains(got, "claude") {
		t.Errorf("formatText agent=claude: missing agent label; got:\n%s", got)
	}
}

func TestFormatText_WithTurnDurations(t *testing.T) {
	ms := int64(90000) // 1m30s
	snap := &metrics.Snapshot{
		TurnDurations: []metrics.TurnDuration{
			{TurnID: "turn-abc", AgentID: "claude", ElapsedMS: &ms},
			{TurnID: "turn-xyz", AgentID: "", ElapsedMS: nil},
		},
	}
	got := formatText(snap, "all", "")
	if !strings.Contains(got, "turn-abc") {
		t.Error("formatText turn durations: missing turn-abc")
	}
	if !strings.Contains(got, "claude") {
		t.Error("formatText turn durations: missing agent")
	}
	if !strings.Contains(got, "unknown") {
		t.Error("formatText turn durations: missing 'unknown' for empty agent")
	}
	if !strings.Contains(got, "—") {
		t.Error("formatText turn durations: missing '—' for nil elapsed")
	}
}

func TestFormatText_WithBlockerAccumulations(t *testing.T) {
	snap := &metrics.Snapshot{
		BlockerAccumulations: []metrics.BlockerAccumulation{
			{BlockerID: "blk-1", OpenedAt: "2026-05-20T10:00:00Z", ClosedAt: "2026-05-20T11:00:00Z", AccumulatedSeconds: 3600},
			{BlockerID: "blk-2", OpenedAt: "2026-05-20T12:00:00Z", ClosedAt: "", AccumulatedSeconds: 0},
		},
	}
	got := formatText(snap, "all", "")
	if !strings.Contains(got, "blk-1") {
		t.Error("formatText blockers: missing blk-1")
	}
	if !strings.Contains(got, "닫힘") {
		t.Error("formatText blockers: missing '닫힘' for closed blocker")
	}
	if !strings.Contains(got, "열림 중") {
		t.Error("formatText blockers: missing '열림 중' for open blocker")
	}
}

func TestFormatText_WithAgentCompletion(t *testing.T) {
	snap := &metrics.Snapshot{
		AgentCompletion: []metrics.AgentCompletion{
			{AgentID: "claude", CompleteCount: 3, TotalCount: 5, CompletionRate: 0.6},
		},
	}
	got := formatText(snap, "all", "")
	if !strings.Contains(got, "claude") {
		t.Error("formatText agent completion: missing agent ID")
	}
	if !strings.Contains(got, "3 / 5") {
		t.Error("formatText agent completion: missing counts")
	}
}

func TestFormatText_WithAgentBlockerFreq(t *testing.T) {
	snap := &metrics.Snapshot{
		AgentBlockerFreq: []metrics.AgentBlockerFreq{
			{AgentID: "codex", BlockerFreq: 0.5, BlockerCount: 2, TurnCount: 4},
		},
	}
	got := formatText(snap, "all", "")
	if !strings.Contains(got, "codex") {
		t.Error("formatText blocker freq: missing agent ID")
	}
}

func TestFormatText_WithEvidenceBound(t *testing.T) {
	snap := &metrics.Snapshot{
		EvidenceBound: metrics.EvidenceBoundDecisions{
			LinkedCount: 2, TotalCount: 5, Ratio: 0.4,
		},
	}
	got := formatText(snap, "all", "")
	if !strings.Contains(got, "2 / 5") {
		t.Error("formatText evidence bound: missing counts")
	}
}

func TestDescribeSkillWrapper(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.sh")
	if got := describeSkillWrapper(missing); got != "not installed" {
		t.Errorf("describe missing wrapper = %q", got)
	}

	wrapper := filepath.Join(dir, "flightlog.sh")
	if err := os.WriteFile(wrapper, []byte("NTTS_FLIGHTLOG_BIN\ncommand -v ntts-flightlog\nexec\n"), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	if got := describeSkillWrapper(wrapper); got != "delegates to Go CLI" {
		t.Errorf("describe delegating wrapper = %q", got)
	}

	legacy := filepath.Join(dir, "legacy.sh")
	if err := os.WriteFile(legacy, []byte("echo legacy\n"), 0o755); err != nil {
		t.Fatalf("write legacy wrapper: %v", err)
	}
	if got := describeSkillWrapper(legacy); got != "installed but delegation pattern not recognized" {
		t.Errorf("describe legacy wrapper = %q", got)
	}
}

// ── openSession (requires WORKLOG_DIR) ─────────────────────────────────────

func TestOpenSession_TempDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WORKLOG_DIR", dir)
	// Ensure agentFlag is clear for this test.
	old := agentFlag
	agentFlag = ""
	defer func() { agentFlag = old }()

	s, err := openSession()
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	defer s.close()

	if s.cfg == nil {
		t.Error("openSession: cfg is nil")
	}
	if s.store == nil {
		t.Error("openSession: store is nil")
	}
}

func TestOpenSession_AgentOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WORKLOG_DIR", dir)
	old := agentFlag
	agentFlag = "claude"
	defer func() { agentFlag = old }()

	s, err := openSession()
	if err != nil {
		t.Fatalf("openSession agent override: %v", err)
	}
	defer s.close()

	if s.override != "claude" {
		t.Errorf("openSession override: override = %q, want claude", s.override)
	}
	if s.agentID != "claude" {
		t.Errorf("openSession override: agentID = %q, want claude", s.agentID)
	}
}

// ── DB-backed helpers via makeTestSession ──────────────────────────────────

// makeTestSession creates a session with an in-memory DB for unit testing.
func makeTestSession(t *testing.T) *session {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WORKLOG_DIR", dir)
	cfg := worklog.DefaultConfig()
	if err := cfg.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	store, err := db.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &session{cfg: cfg, store: store}
}

func TestInsertSession(t *testing.T) {
	s := makeTestSession(t)
	id, err := insertSession(s, "테스트 세션", "solo")
	if err != nil {
		t.Fatalf("insertSession: %v", err)
	}
	if id == "" {
		t.Error("insertSession returned empty ID")
	}
}

func TestInsertTurnWithAnchor(t *testing.T) {
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	id, err := insertTurnWithAnchor(s, sessID, 1, "첫 번째 턴", "의도", `["c1"]`, "완료 기준", "")
	if err != nil {
		t.Fatalf("insertTurnWithAnchor: %v", err)
	}
	if id == "" {
		t.Error("insertTurnWithAnchor returned empty ID")
	}
}

func TestInsertTurn(t *testing.T) {
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	id, err := insertTurn(s, sessID, 1, "기본 턴")
	if err != nil {
		t.Fatalf("insertTurn: %v", err)
	}
	if id == "" {
		t.Error("insertTurn returned empty ID")
	}
}

func TestInsertEntry(t *testing.T) {
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	if err := worklog.WriteFile(s.cfg.SessionIDFile, sessID); err != nil {
		t.Fatal(err)
	}
	id, err := insertEntry(s, db.KindEntry, "항목 제목", "상세")
	if err != nil {
		t.Fatalf("insertEntry: %v", err)
	}
	if id == "" {
		t.Error("insertEntry returned empty ID")
	}
}

func TestInsertEntry_CreatesSessionWithoutActiveSession(t *testing.T) {
	s := makeTestSession(t)
	id, err := insertEntry(s, db.KindEntry, "항목 제목", "상세")
	if err != nil {
		t.Fatalf("insertEntry without active session: %v", err)
	}
	if id == "" {
		t.Error("insertEntry returned empty ID")
	}
	if sessionID := s.cfg.ActiveSessionID(); sessionID == "" {
		t.Error("insertEntry should persist an active session ID")
	}
}

func TestWriteEntry(t *testing.T) {
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	if err := worklog.WriteFile(s.cfg.SessionIDFile, sessID); err != nil {
		t.Fatal(err)
	}
	if err := writeEntry(s, db.KindDecision, "결정 항목", ""); err != nil {
		t.Fatalf("writeEntry: %v", err)
	}
}

func TestWriteEntry_AllKinds(t *testing.T) {
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	if err := worklog.WriteFile(s.cfg.SessionIDFile, sessID); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{db.KindEntry, db.KindDecision, db.KindEvidence, db.KindBlocker} {
		if err := writeEntry(s, kind, kind+" 제목", ""); err != nil {
			t.Errorf("writeEntry kind=%s: %v", kind, err)
		}
	}
}

func TestMaybeReminderAnchor_NoActiveTurn(t *testing.T) {
	// With no active turn ID file, maybeReminderAnchor should return early.
	s := makeTestSession(t)
	maybeReminderAnchor(s) // must not panic
}

func TestMaybeReminderAnchor_WithTurn(t *testing.T) {
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	if err := worklog.WriteFile(s.cfg.SessionIDFile, sessID); err != nil {
		t.Fatal(err)
	}
	turnID, _ := insertTurnWithAnchor(s, sessID, 1, "의도 있는 턴", "테스트 의도", "", "", "")
	if err := worklog.WriteFile(s.cfg.TurnIDFile, turnID); err != nil {
		t.Fatal(err)
	}
	// With < 5 entries since last shown, no reminder should fire.
	maybeReminderAnchor(s)
}

func TestMaybeReminderAnchor_Fires(t *testing.T) {
	// Insert ≥5 entries so the anchor reminder branch fires.
	s := makeTestSession(t)
	sessID, _ := insertSession(s, "세션", "solo")
	if err := worklog.WriteFile(s.cfg.SessionIDFile, sessID); err != nil {
		t.Fatal(err)
	}
	turnID, _ := insertTurnWithAnchor(s, sessID, 1, "앵커 있는 턴", "리마인더 테스트 의도", "", "", "")
	if err := worklog.WriteFile(s.cfg.TurnIDFile, turnID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		_, _ = insertEntry(s, db.KindEntry, "항목", "")
	}
	// entriesSince=6 ≥ 5 → prints reminder and updates anchor_last_shown_at.
	maybeReminderAnchor(s)
}

// ── parseChecksum ──────────────────────────────────────────────────────────

func TestParseChecksum_Found(t *testing.T) {
	data := "abc123  flightlog_linux_amd64.tar.gz\ndef456  flightlog_darwin_arm64.tar.gz\n"
	got, err := parseChecksum(data, "flightlog_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("parseChecksum found: %v", err)
	}
	if got != "def456" {
		t.Errorf("parseChecksum found: got %q, want %q", got, "def456")
	}
}

func TestParseChecksum_NotFound(t *testing.T) {
	data := "abc123  other_file.tar.gz\n"
	_, err := parseChecksum(data, "missing.tar.gz")
	if err == nil {
		t.Error("parseChecksum not found: expected error, got nil")
	}
}

func TestParseChecksum_EmptyAndShortLines(t *testing.T) {
	// Empty lines and lines with < 2 fields are skipped.
	data := "\n\nabc123  target.zip\n\n"
	got, err := parseChecksum(data, "target.zip")
	if err != nil {
		t.Fatalf("parseChecksum empty lines: %v", err)
	}
	if got != "abc123" {
		t.Errorf("parseChecksum empty lines: got %q, want %q", got, "abc123")
	}
}

// ── sha256File ────────────────────────────────────────────────────────────

func TestSha256File_HappyPath(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "sha256-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("hello"); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	got, err := sha256File(f.Name())
	if err != nil {
		t.Fatalf("sha256File: %v", err)
	}
	// SHA-256("hello") known value.
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("sha256File: got %q, want %q", got, want)
	}
}

func TestSha256File_NotFound(t *testing.T) {
	_, err := sha256File(filepath.Join(t.TempDir(), "nonexistent.bin"))
	if err == nil {
		t.Error("sha256File missing file: expected error, got nil")
	}
}

// ── archive helpers ────────────────────────────────────────────────────────

// makeTarGz creates a temp tar.gz archive with a single file named `name`.
func makeTarGz(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:     name,
		Typeflag: tar.TypeReg,
		Size:     int64(len(content)),
		Mode:     0o755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gw.Close()
	return path
}

// makeZip creates a temp zip archive with a single file named `name`.
func makeZip(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = zw.Close()
	return path
}

// ── extractFromTarGz ──────────────────────────────────────────────────────

func TestExtractFromTarGz_HappyPath(t *testing.T) {
	archivePath := makeTarGz(t, "flightlog", []byte("fake-binary-content"))
	outPath, err := extractFromTarGz(archivePath)
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}
	defer os.Remove(outPath)
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "fake-binary-content" {
		t.Errorf("extracted content mismatch: got %q", data)
	}
}

func TestExtractFromTarGz_BinaryNotFound(t *testing.T) {
	archivePath := makeTarGz(t, "other_file", []byte("data"))
	_, err := extractFromTarGz(archivePath)
	if err == nil {
		t.Error("extractFromTarGz missing binary: expected error")
	}
}

func TestExtractFromTarGz_InvalidGzip(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "bad-gzip-*")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("not-a-gzip")
	f.Close()
	_, err = extractFromTarGz(f.Name())
	if err == nil {
		t.Error("extractFromTarGz bad gzip: expected error")
	}
}

func TestExtractFromTarGz_FileNotFound(t *testing.T) {
	_, err := extractFromTarGz(filepath.Join(t.TempDir(), "missing.tar.gz"))
	if err == nil {
		t.Error("extractFromTarGz missing file: expected error")
	}
}

// ── extractFromZip ────────────────────────────────────────────────────────

func TestExtractFromZip_HappyPath(t *testing.T) {
	archivePath := makeZip(t, "flightlog.exe", []byte("exe-content"))
	outPath, err := extractFromZip(archivePath)
	if err != nil {
		t.Fatalf("extractFromZip: %v", err)
	}
	defer os.Remove(outPath)
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read extracted exe: %v", err)
	}
	if string(data) != "exe-content" {
		t.Errorf("extracted exe content mismatch: got %q", data)
	}
}

func TestExtractFromZip_BinaryNotFound(t *testing.T) {
	archivePath := makeZip(t, "other.dll", []byte("data"))
	_, err := extractFromZip(archivePath)
	if err == nil {
		t.Error("extractFromZip missing exe: expected error")
	}
}

func TestExtractFromZip_InvalidZip(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "bad-zip-*")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("not-a-zip")
	f.Close()
	_, err = extractFromZip(f.Name())
	if err == nil {
		t.Error("extractFromZip bad zip: expected error")
	}
}

// ── extractBinary ─────────────────────────────────────────────────────────

func TestExtractBinary_Linux(t *testing.T) {
	archivePath := makeTarGz(t, "flightlog", []byte("bin"))
	outPath, err := extractBinary(archivePath, "linux")
	if err != nil {
		t.Fatalf("extractBinary linux: %v", err)
	}
	defer os.Remove(outPath)
}

func TestExtractBinary_Windows(t *testing.T) {
	archivePath := makeZip(t, "flightlog.exe", []byte("exe"))
	outPath, err := extractBinary(archivePath, "windows")
	if err != nil {
		t.Fatalf("extractBinary windows: %v", err)
	}
	defer os.Remove(outPath)
}

// ── extractFromTarGz with directory entry ─────────────────────────────────

// TestExtractFromTarGz_WithDirEntry covers the non-regular-file continue branch.
func TestExtractFromTarGz_WithDirEntry(t *testing.T) {
	// Create a tar.gz with a directory entry followed by the target binary.
	path := filepath.Join(t.TempDir(), "mixed.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Directory entry: Typeflag = tar.TypeDir → should be skipped (continue).
	dirHdr := &tar.Header{
		Name:     "subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}
	if err := tw.WriteHeader(dirHdr); err != nil {
		t.Fatal(err)
	}

	// Regular file with a non-matching name: also skipped.
	skipHdr := &tar.Header{
		Name:     "subdir/README",
		Typeflag: tar.TypeReg,
		Size:     4,
		Mode:     0o644,
	}
	if err := tw.WriteHeader(skipHdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("skip")); err != nil {
		t.Fatal(err)
	}

	// The target binary.
	content := []byte("real-bin")
	binHdr := &tar.Header{
		Name:     "flightlog",
		Typeflag: tar.TypeReg,
		Size:     int64(len(content)),
		Mode:     0o755,
	}
	if err := tw.WriteHeader(binHdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gw.Close()
	_ = f.Close()

	outPath, err := extractFromTarGz(path)
	if err != nil {
		t.Fatalf("extractFromTarGz with dir: %v", err)
	}
	defer os.Remove(outPath)
}

// ── viewerScript ──────────────────────────────────────────────────────────

// TestViewerScript_Default verifies the REFRESH_SECONDS default branch.
func TestViewerScript_Default(t *testing.T) {
	t.Setenv("REFRESH_SECONDS", "")
	script := viewerScript("/usr/bin/flightlog", "/tmp/main.md", "/tmp/worklog", "/tmp/turns")
	if script == "" {
		t.Error("viewerScript: returned empty string")
	}
	if !strings.Contains(script, "flightlog") {
		t.Error("viewerScript: binary path not in script")
	}
	for _, key := range []string{"r|R|ㄱ", "q|Q|ㅂ"} {
		if !strings.Contains(script, key) {
			t.Errorf("viewerScript: missing Korean key binding %q", key)
		}
	}
	if !strings.Contains(script, "visual:[6]시각화") || !strings.Contains(script, `6) view="visual"`) {
		t.Error("viewerScript: missing visual report menu binding")
	}
	for _, label := range []string{"[r/ㄱ]새로고침", "[q/ㅂ]종료"} {
		if strings.Contains(script, label) {
			t.Errorf("viewerScript: Korean alias should not be shown in label %q", label)
		}
	}
	if strings.Contains(script, "?7l") {
		t.Error("viewerScript: should not disable terminal autowrap")
	}
	if !strings.Contains(script, "?7h") {
		t.Error("viewerScript: should enable terminal autowrap before drawing")
	}
	if !strings.Contains(script, `case "$view" in`) {
		t.Error("viewerScript: should choose truncation policy by view")
	}
	if !strings.Contains(script, `flat)`) || !strings.Contains(script, `tail -n "$content_h"`) {
		t.Error("viewerScript: flat view should keep recency with tail")
	}
	if !strings.Contains(script, `sed -n "1,${content_h}p"`) {
		t.Error("viewerScript: summary views should keep the top of the rendered view")
	}
}

// TestViewerScript_CustomRefresh verifies the custom REFRESH_SECONDS branch.
func TestViewerScript_CustomRefresh(t *testing.T) {
	t.Setenv("REFRESH_SECONDS", "5")
	script := viewerScript("/usr/local/bin/flightlog", "/a/main.md", "/a", "/a/turns")
	if !strings.Contains(script, "5") {
		t.Errorf("viewerScript custom refresh: '5' not found in script: %q", script[:100])
	}
}

// ── paneAlive ────────────────────────────────────────────────────────────

// TestPaneAlive_EmptyFile returns false when no pane file exists.
func TestPaneAlive_EmptyFile(t *testing.T) {
	s := makeTestSession(t)
	if paneAlive(s.cfg) {
		t.Error("paneAlive empty file: expected false")
	}
}

// TestPaneAlive_WithNonexistentPaneID writes a pane ID that can never match
// a real tmux pane, so paneAlive returns false regardless of tmux state.
func TestPaneAlive_WithNonexistentPaneID(t *testing.T) {
	s := makeTestSession(t)
	// Use an ID that is syntactically invalid for tmux panes (format: %N).
	if err := worklog.WriteFile(s.cfg.PaneFile, "test-nonexistent-pane-id"); err != nil {
		t.Fatal(err)
	}
	// paneAlive will: (a) exec tmux and get err → false, OR
	// (b) exec tmux, iterate output, find no match → false. Either way false.
	if paneAlive(s.cfg) {
		t.Error("paneAlive nonexistent ID: expected false")
	}
}

// ── self_upgrade pure helpers (not covered by self_upgrade_test.go) ───────

func TestEnsureV_AddPrefix(t *testing.T) {
	if ensureV("1.2.3") != "v1.2.3" {
		t.Error("ensureV: expected v prefix added")
	}
}

func TestEnsureV_AlreadyPrefixed(t *testing.T) {
	if ensureV("v1.2.3") != "v1.2.3" {
		t.Error("ensureV: should not double-prefix")
	}
}
