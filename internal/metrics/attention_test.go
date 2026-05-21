package metrics_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

func TestQueryAttentionRules(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	seedAttentionFixture(t, d)
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	snap, err := metrics.QueryAttention(d, metrics.Filter{Window: "all"}, metrics.AttentionOptions{Now: now})
	if err != nil {
		t.Fatalf("QueryAttention: %v", err)
	}

	rules := map[string]int{}
	for _, item := range snap.Items {
		if item.Reason == "" {
			t.Errorf("item %s missing reason", item.ID)
		}
		if item.SourceType == "" || item.SourceID == "" {
			t.Errorf("item %s missing source: %#v", item.ID, item)
		}
		if item.RecommendedAction == "" {
			t.Errorf("item %s missing recommended action", item.ID)
		}
		rules[item.Rule]++
	}

	for _, rule := range []string{
		metrics.AttentionRuleStaleBlocker,
		metrics.AttentionRuleDecisionWithoutEvidence,
		metrics.AttentionRuleActiveTurnWithoutEvidence,
		metrics.AttentionRuleDriftAlert,
		metrics.AttentionRuleLongTurnWithoutOutcome,
		metrics.AttentionRuleAgentAttribution,
	} {
		if rules[rule] == 0 {
			t.Errorf("missing attention rule %s in %#v", rule, rules)
		}
	}
	if snap.Summary.Total != len(snap.Items) {
		t.Fatalf("summary total = %d, want %d", snap.Summary.Total, len(snap.Items))
	}
	if snap.Summary.High == 0 || snap.Summary.Medium == 0 || snap.Summary.Low == 0 {
		t.Fatalf("expected high/medium/low counts, got %#v", snap.Summary)
	}
}

func TestQueryAttentionAgentFilterSkipsGlobalAttribution(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = d.Close() }()
	seedAttentionFixture(t, d)

	snap, err := metrics.QueryAttention(d, metrics.Filter{Window: "all", Agent: "codex"}, metrics.AttentionOptions{
		Now: time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("QueryAttention filtered: %v", err)
	}
	for _, item := range snap.Items {
		if item.Rule == metrics.AttentionRuleAgentAttribution {
			t.Fatalf("agent-filtered attention should not include global attribution item: %#v", item)
		}
	}
}

func TestFormatAttentionJSON(t *testing.T) {
	snap := &metrics.AttentionSnapshot{
		Window: "all",
		Items: []metrics.AttentionItem{
			{
				ID:                "decision_without_evidence:e1",
				Severity:          metrics.AttentionSeverityHigh,
				Rule:              metrics.AttentionRuleDecisionWithoutEvidence,
				SourceType:        "decision",
				SourceID:          "e1",
				Title:             "근거 없는 결정",
				Reason:            "유효한 결정에 연결된 evidence가 없습니다.",
				RecommendedAction: "evidence를 연결하세요.",
			},
		},
	}
	raw, err := metrics.FormatAttentionJSON(snap, time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FormatAttentionJSON: %v", err)
	}
	var got struct {
		GeneratedAt string                   `json:"generated_at"`
		Summary     metrics.AttentionSummary `json:"summary"`
		Items       []metrics.AttentionItem  `json:"items"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("attention JSON invalid: %v\n%s", err, raw)
	}
	if got.GeneratedAt != "2026-05-21T12:00:00Z" {
		t.Fatalf("generated_at = %q", got.GeneratedAt)
	}
	if got.Summary.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("summary/items = %#v len=%d", got.Summary, len(got.Items))
	}
}

func TestAttentionSchemaFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(fixtureDir(t), "golden", "attention_schema.json"))
	if err != nil {
		t.Fatalf("read attention_schema.json: %v", err)
	}
	var schemaDoc map[string]any
	if err := json.Unmarshal(raw, &schemaDoc); err != nil {
		t.Fatalf("attention_schema.json invalid: %v", err)
	}
	if schemaDoc["$schema"] == nil {
		t.Fatal("attention_schema.json missing $schema")
	}
}

func seedAttentionFixture(t *testing.T, d *db.DB) {
	t.Helper()
	execMetricFixture(t, d, `INSERT INTO sessions
		(id, started_at, mode, agent_id, agent_detected, agent_override, title)
		VALUES
		('s1', '2026-05-21T08:00:00Z', 'solo', 'codex', 'codex', NULL, 'codex session'),
		('s2', '2026-05-21T08:10:00Z', 'solo', NULL, 'unknown', NULL, 'unknown session'),
		('s3', '2026-05-21T08:20:00Z', 'solo', 'codex', 'claude', 'codex', 'override session')`)
	execMetricFixture(t, d, `INSERT INTO turns
		(id, session_id, sequence_no, title, started_at, status, elapsed_ms, agent_id, intent, done_when, drift_alerts, outcome)
		VALUES
		('t1', 's1', 1, '증거 없는 진행 턴', '2026-05-21T09:00:00Z', 'active', NULL, 'codex', '주의 큐 구현', '테스트 통과', 0, NULL),
		('t2', 's1', 2, '긴 완료 턴', '2026-05-21T06:00:00Z', 'complete', 10800000, 'codex', NULL, NULL, 0, NULL),
		('t3', 's1', 3, '드리프트 턴', '2026-05-21T09:30:00Z', 'active', NULL, 'codex', NULL, NULL, 2, NULL)`)
	execMetricFixture(t, d, `INSERT INTO entries
		(id, session_id, turn_id, kind, title, detail, created_at, agent_id)
		VALUES
		('e1', 's1', 't1', 'entry', '조사', NULL, '2026-05-21T09:05:00Z', 'codex'),
		('e2', 's1', 't1', 'entry', '구현', NULL, '2026-05-21T09:10:00Z', 'codex'),
		('e3', 's1', 't1', 'entry', '검토', NULL, '2026-05-21T09:15:00Z', 'codex'),
		('d1', 's1', 't1', 'decision', '근거 없는 결정', NULL, '2026-05-21T09:20:00Z', 'codex'),
		('d2', 's1', 't1', 'decision', '근거 있는 결정', NULL, '2026-05-21T09:25:00Z', 'codex'),
		('v1', 's1', 't2', 'evidence', '완료 근거', NULL, '2026-05-21T09:30:00Z', 'codex')`)
	execMetricFixture(t, d, `INSERT INTO decision_evidence_links
		(decision_entry_id, evidence_entry_id, created_at)
		VALUES ('d2', 'v1', '2026-05-21T09:31:00Z')`)
	execMetricFixture(t, d, `INSERT INTO blockers
		(id, turn_id, entry_id, title, opened_at, status, accumulated_seconds)
		VALUES ('b1', 't1', NULL, '오래 열린 블로커', '2026-05-21T10:00:00Z', 'open', 0)`)
}

func execMetricFixture(t *testing.T, d *db.DB, query string, args ...any) {
	t.Helper()
	if _, err := d.Exec(query, args...); err != nil {
		t.Fatalf("exec fixture query: %v\n%s", err, query)
	}
}
