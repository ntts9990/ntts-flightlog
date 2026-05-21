package views_test

// views_extra_test.go: coverage tests for WriteTurnAnchor, RenderFlat,
// RenderTurns, RenderBlockers, RenderDecisions, RenderReport, itoa, LoadAll,
// SeqSum. Supplements ansi_test.go which covers WriteEntry/WriteTurnStart/etc.

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/tui/views"
)

// ── WriteTurnAnchor ────────────────────────────────────────────────────────

func TestWriteTurnAnchor_NoIntent(t *testing.T) {
	var sb strings.Builder
	views.WriteTurnAnchor(&sb, views.Turn{}) // all fields invalid → no output
	if sb.Len() != 0 {
		t.Errorf("WriteTurnAnchor no intent: expected empty, got %q", sb.String())
	}
}

func TestWriteTurnAnchor_IntentOnly(t *testing.T) {
	var sb strings.Builder
	turn := views.Turn{
		Intent: sql.NullString{String: "테스트 의도", Valid: true},
	}
	views.WriteTurnAnchor(&sb, turn)
	got := sb.String()
	if !strings.Contains(got, "테스트 의도") {
		t.Errorf("WriteTurnAnchor intent only: missing intent in %q", got)
	}
	if strings.Contains(got, "📐") {
		t.Error("WriteTurnAnchor intent only: should not contain constraints line")
	}
	if strings.Contains(got, "✅") {
		t.Error("WriteTurnAnchor intent only: should not contain doneWhen line")
	}
}

func TestWriteTurnAnchor_AllFields(t *testing.T) {
	var sb strings.Builder
	turn := views.Turn{
		Intent:          sql.NullString{String: "전체 의도", Valid: true},
		ConstraintsJSON: sql.NullString{String: `["제약1","제약2"]`, Valid: true},
		DoneWhen:        sql.NullString{String: "완료 조건", Valid: true},
	}
	views.WriteTurnAnchor(&sb, turn)
	got := sb.String()
	for _, want := range []string{"⚓", "전체 의도", "📐", "✅", "완료 조건"} {
		if !strings.Contains(got, want) {
			t.Errorf("WriteTurnAnchor all fields: missing %q in %q", want, got)
		}
	}
}

func TestWriteTurnAnchor_EmptyIntent(t *testing.T) {
	// Valid=true but empty string → still no output (intent.String == "")
	var sb strings.Builder
	turn := views.Turn{
		Intent: sql.NullString{String: "", Valid: true},
	}
	views.WriteTurnAnchor(&sb, turn)
	// No intent content = function returns early (intent.String is "")
	// The function checks !t.Intent.Valid || t.Intent.String == "" to return.
	if sb.Len() != 0 {
		t.Errorf("WriteTurnAnchor empty intent: expected empty, got %q", sb.String())
	}
}

// ── RenderFlat ────────────────────────────────────────────────────────────

func TestRenderFlat_Nil(t *testing.T) {
	got := views.RenderFlat(nil, "/tmp/turns")
	if !strings.Contains(got, "워크로그가 비어") {
		t.Errorf("RenderFlat nil: expected placeholder, got %q", got)
	}
}

func TestRenderFlat_Empty(t *testing.T) {
	data := &views.WorklogData{}
	got := views.RenderFlat(data, "/tmp/turns")
	if !strings.Contains(got, "워크로그가 비어") {
		t.Errorf("RenderFlat empty: expected placeholder, got %q", got)
	}
}

func TestRenderFlat_WithSessionAndEntries(t *testing.T) {
	data := &views.WorklogData{
		Sessions: []views.Session{
			{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo",
				Title: sql.NullString{String: "내 세션", Valid: true}},
		},
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "엔트리", CreatedAt: "2026-05-20T10:01:00Z"},
			{ID: "e2", SessionID: "s1", Kind: "decision", Title: "결정",
				CreatedAt: "2026-05-20T10:02:00Z",
				Detail:    sql.NullString{String: "상세 내용", Valid: true}},
		},
	}
	got := views.RenderFlat(data, "/tmp/turns")
	if !strings.Contains(got, "내 세션") {
		t.Error("RenderFlat: missing session title")
	}
	if !strings.Contains(got, "엔트리") {
		t.Error("RenderFlat: missing entry title")
	}
	if !strings.Contains(got, "결정") {
		t.Error("RenderFlat: missing decision title")
	}
	if !strings.Contains(got, "상세 내용") {
		t.Error("RenderFlat: missing detail body")
	}
}

func TestRenderFlat_WithTurnAndAnchor(t *testing.T) {
	turn := views.Turn{
		ID: "t1", SessionID: "s1", SequenceNo: 1,
		StartedAt: "2026-05-20T10:00:00Z",
		Title:     sql.NullString{String: "첫 번째 턴", Valid: true},
		Intent:    sql.NullString{String: "앵커 의도", Valid: true},
		EndedAt:   sql.NullString{String: "2026-05-20T11:00:00Z", Valid: true},
	}
	entry := views.Entry{
		ID: "e1", SessionID: "s1", Kind: "entry", Title: "마지막 엔트리",
		CreatedAt: "2026-05-20T10:30:00Z",
		TurnID:    sql.NullString{String: "t1", Valid: true},
	}
	data := &views.WorklogData{
		Sessions: []views.Session{
			{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"},
		},
		Turns:   []views.Turn{turn},
		Entries: []views.Entry{entry},
	}
	got := views.RenderFlat(data, "/tmp/turns")
	if !strings.Contains(got, "앵커 의도") {
		t.Errorf("RenderFlat: missing anchor intent; got:\n%s", got)
	}
}

func TestRenderFlat_TurnEndedWithNonEntrySummary(t *testing.T) {
	// Covers the "if last.Kind == 'entry'" branch in WriteTurnEnd summary logic.
	turn := views.Turn{
		ID: "t1", SessionID: "s1", SequenceNo: 1,
		StartedAt: "2026-05-20T10:00:00Z",
		EndedAt:   sql.NullString{String: "2026-05-20T11:00:00Z", Valid: true},
	}
	data := &views.WorklogData{
		Sessions: []views.Session{{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"}},
		Turns:    []views.Turn{turn},
	}
	got := views.RenderFlat(data, "/tmp/turns")
	// Should render without panic; turn-end line should appear.
	if !strings.Contains(got, "turn-1-end") {
		t.Errorf("RenderFlat turn ended: missing turn-1-end; got:\n%s", got)
	}
}

func TestRenderFlat_SessionNoTitle(t *testing.T) {
	// Session with no title should fall back to "NTTS Flightlog".
	data := &views.WorklogData{
		Sessions: []views.Session{
			{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"},
		},
	}
	got := views.RenderFlat(data, "/tmp/turns")
	if !strings.Contains(got, "Flightlog") {
		t.Errorf("RenderFlat no title: expected fallback title, got:\n%s", got)
	}
}

// ── RenderTurns ───────────────────────────────────────────────────────────

func TestRenderTurns_Nil(t *testing.T) {
	got := views.RenderTurns(nil, "/tmp/turns")
	if !strings.Contains(got, "turn") {
		t.Errorf("RenderTurns nil: expected placeholder, got %q", got)
	}
}

func TestRenderTurns_Empty(t *testing.T) {
	got := views.RenderTurns(&views.WorklogData{}, "/tmp/turns")
	if !strings.Contains(got, "turn") {
		t.Errorf("RenderTurns empty: expected placeholder, got %q", got)
	}
}

func TestRenderTurns_WithTurns(t *testing.T) {
	turn := views.Turn{
		ID: "t1", SessionID: "s1", SequenceNo: 2,
		StartedAt: "2026-05-20T10:00:00Z",
		Title:     sql.NullString{String: "두 번째 턴", Valid: true},
		Intent:    sql.NullString{String: "의도 내용", Valid: true},
	}
	entry := views.Entry{
		ID: "e1", SessionID: "s1", Kind: "entry", Title: "턴 항목",
		CreatedAt: "2026-05-20T10:01:00Z",
		TurnID:    sql.NullString{String: "t1", Valid: true},
		Detail:    sql.NullString{String: "원문 상세는 턴 인덱스에서 숨김", Valid: true},
	}
	data := &views.WorklogData{
		Sessions: []views.Session{{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"}},
		Turns:    []views.Turn{turn},
		Entries:  []views.Entry{entry},
	}
	got := views.RenderTurns(data, "/tmp/turns")
	if !strings.Contains(got, "두 번째 턴") {
		t.Errorf("RenderTurns: missing turn title; got:\n%s", got)
	}
	if !strings.Contains(got, "턴 항목") {
		t.Errorf("RenderTurns: missing entry title; got:\n%s", got)
	}
	if !strings.Contains(got, "신호: entry 1") {
		t.Errorf("RenderTurns: missing signal counts; got:\n%s", got)
	}
	if strings.Contains(got, "원문 상세") {
		t.Errorf("RenderTurns: should summarize, not repeat entry details; got:\n%s", got)
	}
}

func TestRenderTurns_EndedTurnLastEntryKindEntry(t *testing.T) {
	// Covers: last entry kind=="entry" → use its title as summary.
	turn := views.Turn{
		ID: "t1", SessionID: "s1", SequenceNo: 1,
		StartedAt: "2026-05-20T10:00:00Z",
		EndedAt:   sql.NullString{String: "2026-05-20T11:00:00Z", Valid: true},
	}
	entry := views.Entry{
		ID: "e1", SessionID: "s1", Kind: "entry", Title: "마지막 작업",
		CreatedAt: "2026-05-20T10:30:00Z",
		TurnID:    sql.NullString{String: "t1", Valid: true},
	}
	data := &views.WorklogData{
		Turns:   []views.Turn{turn},
		Entries: []views.Entry{entry},
	}
	got := views.RenderTurns(data, "/tmp/turns")
	if !strings.Contains(got, "마지막 작업") {
		t.Errorf("RenderTurns ended+entry: missing last entry as summary; got:\n%s", got)
	}
}

func TestRenderTurns_UsesExplicitOutcome(t *testing.T) {
	turn := views.Turn{
		ID: "t1", SessionID: "s1", SequenceNo: 1,
		StartedAt: "2026-05-20T10:00:00Z",
		EndedAt:   sql.NullString{String: "2026-05-20T11:00:00Z", Valid: true},
		Outcome:   sql.NullString{String: "명시적 결과", Valid: true},
	}
	entry := views.Entry{
		ID: "e1", SessionID: "s1", Kind: "entry", Title: "마지막 엔트리",
		CreatedAt: "2026-05-20T10:30:00Z",
		TurnID:    sql.NullString{String: "t1", Valid: true},
	}
	data := &views.WorklogData{
		Turns:   []views.Turn{turn},
		Entries: []views.Entry{entry},
	}
	got := views.RenderTurns(data, "/tmp/turns")
	if !strings.Contains(got, "결과: 명시적 결과") {
		t.Errorf("RenderTurns explicit outcome: missing outcome; got:\n%s", got)
	}
	if strings.Contains(got, "결과: 마지막 엔트리") {
		t.Errorf("RenderTurns explicit outcome: should not use last entry; got:\n%s", got)
	}
}

func TestRenderTurns_TurnNoTitle(t *testing.T) {
	// Turn with no title should fall back to "(제목 없음)".
	turn := views.Turn{
		ID: "t1", SessionID: "s1", SequenceNo: 1,
		StartedAt: "2026-05-20T10:00:00Z",
	}
	data := &views.WorklogData{
		Turns: []views.Turn{turn},
	}
	got := views.RenderTurns(data, "/tmp/turns")
	if !strings.Contains(got, "제목 없음") {
		t.Errorf("RenderTurns no title: expected fallback; got:\n%s", got)
	}
}

// ── RenderBlockers ────────────────────────────────────────────────────────

func TestRenderBlockers_Nil(t *testing.T) {
	got := views.RenderBlockers(nil)
	if !strings.Contains(got, "블로커가 없습니다") {
		t.Errorf("RenderBlockers nil: expected placeholder, got %q", got)
	}
}

func TestRenderBlockers_NoBlockers(t *testing.T) {
	data := &views.WorklogData{
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "일반", CreatedAt: "2026-05-20T10:00:00Z"},
		},
	}
	got := views.RenderBlockers(data)
	if !strings.Contains(got, "블로커가 없습니다") {
		t.Errorf("RenderBlockers no blockers: expected placeholder, got %q", got)
	}
}

func TestRenderBlockers_WithBlockers(t *testing.T) {
	data := &views.WorklogData{
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "일반", CreatedAt: "2026-05-20T10:00:00Z"},
			{ID: "e2", SessionID: "s1", Kind: "blocker", Title: "블로킹 이슈",
				CreatedAt: "2026-05-20T10:01:00Z",
				Detail:    sql.NullString{String: "차단 상세", Valid: true}},
		},
	}
	got := views.RenderBlockers(data)
	if !strings.Contains(got, "블로킹 이슈") {
		t.Error("RenderBlockers: missing blocker title")
	}
	if !strings.Contains(got, "차단 상세") {
		t.Error("RenderBlockers: missing blocker detail")
	}
	if strings.Contains(got, "일반") {
		t.Error("RenderBlockers: should not include non-blocker entry")
	}
}

func TestRenderBlockers_WithStateRows(t *testing.T) {
	data := &views.WorklogData{
		Turns: []views.Turn{
			{ID: "t1", SessionID: "s1", SequenceNo: 1, StartedAt: "2026-05-20T10:00:00Z",
				Title: sql.NullString{String: "블로커 턴", Valid: true}},
		},
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "blocker", Title: "열린 상세",
				Detail: sql.NullString{String: "상세 원인", Valid: true}},
		},
		Blockers: []views.Blocker{
			{ID: "b1", EntryID: sql.NullString{String: "e1", Valid: true},
				TurnID: sql.NullString{String: "t1", Valid: true},
				Title:  "열린 블로커", OpenedAt: "2026-05-20T10:01:00Z", Status: "open",
				AccumulatedSeconds: 120},
			{ID: "b2", Title: "해결 블로커", OpenedAt: "2026-05-20T10:02:00Z",
				ClosedAt:           sql.NullString{String: "2026-05-20T10:03:00Z", Valid: true},
				Status:             "resolved",
				AccumulatedSeconds: 60},
		},
	}
	got := views.RenderBlockers(data)
	for _, want := range []string{"열림", "해결됨", "열린 블로커", "해결 블로커", "turn-1", "상세 원인", "열린 시간: 2m 00s", "총 차단: 1m 00s"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderBlockers state rows: missing %q in:\n%s", want, got)
		}
	}
}

// ── RenderDecisions ───────────────────────────────────────────────────────

func TestRenderDecisions_Nil(t *testing.T) {
	got := views.RenderDecisions(nil)
	if !strings.Contains(got, "결정 사항이 아직 없습니다") {
		t.Errorf("RenderDecisions nil: expected placeholder, got %q", got)
	}
}

func TestRenderDecisions_NoDecisions(t *testing.T) {
	data := &views.WorklogData{
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "일반", CreatedAt: "2026-05-20T10:00:00Z"},
		},
	}
	got := views.RenderDecisions(data)
	if !strings.Contains(got, "결정 사항이 아직 없습니다") {
		t.Errorf("RenderDecisions no decisions: expected placeholder, got %q", got)
	}
}

func TestRenderDecisions_WithDecisions(t *testing.T) {
	data := &views.WorklogData{
		Turns: []views.Turn{
			{ID: "t1", SessionID: "s1", SequenceNo: 1, StartedAt: "2026-05-20T10:00:00Z",
				Title: sql.NullString{String: "결정 턴", Valid: true}},
		},
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "일반", CreatedAt: "2026-05-20T10:00:00Z"},
			{ID: "e2", SessionID: "s1", Kind: "decision", Title: "중요 결정",
				CreatedAt: "2026-05-20T10:01:00Z",
				TurnID:    sql.NullString{String: "t1", Valid: true},
				Detail:    sql.NullString{String: "결정 근거", Valid: true}},
			{ID: "e3", SessionID: "s1", Kind: "evidence", Title: "검증 근거",
				CreatedAt: "2026-05-20T10:02:00Z",
				TurnID:    sql.NullString{String: "t1", Valid: true}},
		},
		DecisionEvidenceLinks: []views.DecisionEvidenceLink{
			{DecisionEntryID: "e2", EvidenceEntryID: "e3"},
		},
		DecisionStates: []views.DecisionState{
			{DecisionEntryID: "e2", Status: "accepted"},
		},
	}
	got := views.RenderDecisions(data)
	if !strings.Contains(got, "중요 결정") {
		t.Error("RenderDecisions: missing decision title")
	}
	if !strings.Contains(got, "결정 근거") {
		t.Error("RenderDecisions: missing decision detail")
	}
	if !strings.Contains(got, "id: e2") {
		t.Errorf("RenderDecisions: missing short ID; got:\n%s", got)
	}
	if strings.Contains(got, "일반") {
		t.Error("RenderDecisions: should not include non-decision entry")
	}
	for _, want := range []string{"유효한 결정", "상태: accepted", "turn-1", "결정 턴", "linked 1", "same-turn 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderDecisions: missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDecisions_GroupsSuperseded(t *testing.T) {
	data := &views.WorklogData{
		Entries: []views.Entry{
			{ID: "old-decision", SessionID: "s1", Kind: "decision", Title: "옛 결정", CreatedAt: "2026-05-20T10:00:00Z"},
			{ID: "new-decision", SessionID: "s1", Kind: "decision", Title: "새 결정", CreatedAt: "2026-05-20T10:01:00Z"},
		},
		DecisionStates: []views.DecisionState{
			{DecisionEntryID: "old-decision", Status: "superseded",
				SupersededByEntryID: sql.NullString{String: "new-decision", Valid: true},
				Rationale:           sql.NullString{String: "더 나은 경로", Valid: true}},
			{DecisionEntryID: "new-decision", Status: "accepted"},
		},
	}
	got := views.RenderDecisions(data)
	for _, want := range []string{"유효한 결정", "새 결정", "대체된 결정", "옛 결정", "상태: superseded", "대체됨: new-deci", "사유: 더 나은 경로"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderDecisions superseded: missing %q in:\n%s", want, got)
		}
	}
}

// ── RenderReport ──────────────────────────────────────────────────────────

func TestRenderReport_Nil(t *testing.T) {
	got := views.RenderReport(nil)
	if !strings.Contains(got, "리포트") {
		t.Errorf("RenderReport nil: expected report header, got %q", got)
	}
	if !strings.Contains(got, "데이터 없음") {
		t.Errorf("RenderReport nil: expected empty-data message, got %q", got)
	}
}

func TestRenderReport_WithData(t *testing.T) {
	data := &views.WorklogData{
		Sessions: []views.Session{
			{ID: "s1", StartedAt: "2026-05-20T10:00:00Z", Mode: "solo"},
		},
		Turns: []views.Turn{
			{ID: "t1", SessionID: "s1", SequenceNo: 1, StartedAt: "2026-05-20T10:00:00Z", Status: "active"},
			{ID: "t2", SessionID: "s1", SequenceNo: 2, StartedAt: "2026-05-20T10:05:00Z",
				Status:    "complete",
				ElapsedMs: sql.NullInt64{Int64: 120000, Valid: true}},
		},
		Entries: []views.Entry{
			{ID: "e1", SessionID: "s1", Kind: "entry", Title: "항목", CreatedAt: "2026-05-20T10:01:00Z"},
			{ID: "e2", SessionID: "s1", Kind: "decision", Title: "결정", CreatedAt: "2026-05-20T10:02:00Z"},
			{ID: "e3", SessionID: "s1", Kind: "evidence", Title: "근거", CreatedAt: "2026-05-20T10:03:00Z"},
			{ID: "e4", SessionID: "s1", Kind: "blocker", Title: "블로커", CreatedAt: "2026-05-20T10:04:00Z"},
			{ID: "e5", SessionID: "s1", Kind: "decision", Title: "옛 결정", CreatedAt: "2026-05-20T10:05:00Z"},
		},
		Blockers: []views.Blocker{
			{ID: "b1", Title: "열린 블로커", OpenedAt: "2026-05-20T10:04:00Z", Status: "open"},
			{ID: "b2", Title: "해결 블로커", OpenedAt: "2026-05-20T10:06:00Z", Status: "resolved"},
		},
		DecisionEvidenceLinks: []views.DecisionEvidenceLink{
			{DecisionEntryID: "e2", EvidenceEntryID: "e3"},
		},
		DecisionStates: []views.DecisionState{
			{DecisionEntryID: "e2", Status: "accepted"},
			{DecisionEntryID: "e5", Status: "superseded"},
		},
	}
	got := views.RenderReport(data)
	if strings.Contains(got, "Phase B4") {
		t.Errorf("RenderReport: should not render placeholder copy; got:\n%s", got)
	}
	// Must include counts for sessions/turns/entries.
	if !strings.Contains(got, "세션") {
		t.Error("RenderReport: missing 세션 count")
	}
	if !strings.Contains(got, "턴") {
		t.Error("RenderReport: missing 턴 count")
	}
	if !strings.Contains(got, "엔트리") {
		t.Error("RenderReport: missing 엔트리 count")
	}
	// decision, evidence, blocker sub-counts.
	if !strings.Contains(got, "결정") {
		t.Error("RenderReport: missing decision sub-count")
	}
	if !strings.Contains(got, "근거") {
		t.Error("RenderReport: missing evidence sub-count")
	}
	if !strings.Contains(got, "블로커") {
		t.Error("RenderReport: missing blocker sub-count")
	}
	for _, want := range []string{"완료 턴: 1", "진행 중: 1", "평균 완료 시간: 2m 00s", "accepted 1", "superseded 1", "근거 연결 결정: 1/2 (50%)", "열린 블로커: 1", "해결됨: 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderReport: missing %q in:\n%s", want, got)
		}
	}
}

// ── LoadAll / SeqSum (require a live in-memory DB) ────────────────────────

func TestLoadAll_EmptyDB(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	wd, err := views.LoadAll(d)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if wd == nil {
		t.Fatal("LoadAll returned nil WorklogData")
	}
	if len(wd.Sessions) != 0 {
		t.Errorf("LoadAll empty: Sessions = %d, want 0", len(wd.Sessions))
	}
	if len(wd.Turns) != 0 {
		t.Errorf("LoadAll empty: Turns = %d, want 0", len(wd.Turns))
	}
	if len(wd.Entries) != 0 {
		t.Errorf("LoadAll empty: Entries = %d, want 0", len(wd.Entries))
	}
}

func TestSeqSum_EmptyDB(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	sum, err := views.SeqSum(d)
	if err != nil {
		t.Fatalf("SeqSum: %v", err)
	}
	if sum != 0 {
		t.Errorf("SeqSum empty = %d, want 0", sum)
	}
}

func TestLoadAll_WithData(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	// Insert a session row directly.
	var sessionID string
	if err := d.QueryRow(`INSERT INTO sessions (started_at, mode) VALUES ('2026-05-20T10:00:00Z', 'solo') RETURNING id`).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	var turnID string
	if err := d.QueryRow(`INSERT INTO turns (session_id, sequence_no, title, started_at, outcome) VALUES (?, 1, '로드 턴', '2026-05-20T10:01:00Z', '로드 결과') RETURNING id`, sessionID).Scan(&turnID); err != nil {
		t.Fatalf("insert turn: %v", err)
	}
	var decisionID string
	if err := d.QueryRow(`INSERT INTO entries (session_id, turn_id, kind, title, created_at) VALUES (?, ?, 'decision', '로드 결정', '2026-05-20T10:02:00Z') RETURNING id`, sessionID, turnID).Scan(&decisionID); err != nil {
		t.Fatalf("insert decision: %v", err)
	}
	var evidenceID string
	if err := d.QueryRow(`INSERT INTO entries (session_id, turn_id, kind, title, created_at) VALUES (?, ?, 'evidence', '로드 근거', '2026-05-20T10:03:00Z') RETURNING id`, sessionID, turnID).Scan(&evidenceID); err != nil {
		t.Fatalf("insert evidence: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO decision_evidence_links (decision_entry_id, evidence_entry_id, created_at) VALUES (?, ?, '2026-05-20T10:04:00Z')`, decisionID, evidenceID); err != nil {
		t.Fatalf("insert decision evidence link: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO decision_status (decision_entry_id, status) VALUES (?, 'accepted')`, decisionID); err != nil {
		t.Fatalf("insert decision status: %v", err)
	}
	if _, err := d.Exec(`INSERT INTO blockers (turn_id, entry_id, title, opened_at) VALUES (?, ?, '로드 블로커', '2026-05-20T10:05:00Z')`, turnID, decisionID); err != nil {
		t.Fatalf("insert blocker: %v", err)
	}

	wd, err := views.LoadAll(d)
	if err != nil {
		t.Fatalf("LoadAll with data: %v", err)
	}
	if len(wd.Sessions) != 1 {
		t.Errorf("LoadAll with data: Sessions = %d, want 1", len(wd.Sessions))
	}
	if len(wd.Blockers) != 1 {
		t.Errorf("LoadAll with data: Blockers = %d, want 1", len(wd.Blockers))
	}
	if len(wd.DecisionEvidenceLinks) != 1 {
		t.Errorf("LoadAll with data: DecisionEvidenceLinks = %d, want 1", len(wd.DecisionEvidenceLinks))
	}
	if len(wd.DecisionStates) != 1 {
		t.Errorf("LoadAll with data: DecisionStates = %d, want 1", len(wd.DecisionStates))
	}

	sum, err := views.SeqSum(d)
	if err != nil {
		t.Fatalf("SeqSum with session: %v", err)
	}
	if sum != 7 {
		t.Errorf("SeqSum with full fixture = %d, want 7", sum)
	}
}
