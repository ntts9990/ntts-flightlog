package metrics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

const (
	AttentionRuleStaleBlocker              = "stale_blocker"
	AttentionRuleDecisionWithoutEvidence   = "decision_without_evidence"
	AttentionRuleActiveTurnWithoutEvidence = "active_turn_without_evidence"
	AttentionRuleDriftAlert                = "drift_alert"
	AttentionRuleAgentAttribution          = "agent_attribution_warning"
	AttentionRuleLongTurnWithoutOutcome    = "long_turn_without_outcome"
)

const (
	AttentionSeverityHigh   = "high"
	AttentionSeverityMedium = "medium"
	AttentionSeverityLow    = "low"
)

// AttentionOptions controls thresholds for operator-facing recommendations.
type AttentionOptions struct {
	Now                   time.Time
	OpenBlockerAge        time.Duration
	ActiveTurnAge         time.Duration
	ActiveTurnEntries     int
	LongTurnAge           time.Duration
	UnknownRateThreshold  float64
	MismatchRateThreshold float64
}

// AttentionSummary counts attention items by severity.
type AttentionSummary struct {
	Total  int `json:"total"`
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}

// AttentionItem is one concrete operator action with a source object.
type AttentionItem struct {
	ID                string `json:"id"`
	Severity          string `json:"severity"`
	Rule              string `json:"rule"`
	SourceType        string `json:"source_type"`
	SourceID          string `json:"source_id"`
	Title             string `json:"title"`
	Reason            string `json:"reason"`
	RecommendedAction string `json:"recommended_action"`
	AgeSeconds        *int64 `json:"age_seconds,omitempty"`
}

// AttentionSnapshot is the schema-stable output for `flightlog attention`.
type AttentionSnapshot struct {
	Window      string           `json:"window"`
	Agent       string           `json:"agent"`
	GeneratedAt string           `json:"generated_at"`
	Summary     AttentionSummary `json:"summary"`
	Items       []AttentionItem  `json:"items"`
}

func (o AttentionOptions) withDefaults() AttentionOptions {
	if o.Now.IsZero() {
		o.Now = time.Now().UTC()
	} else {
		o.Now = o.Now.UTC()
	}
	if o.OpenBlockerAge <= 0 {
		o.OpenBlockerAge = time.Hour
	}
	if o.ActiveTurnAge <= 0 {
		o.ActiveTurnAge = 30 * time.Minute
	}
	if o.ActiveTurnEntries <= 0 {
		o.ActiveTurnEntries = 3
	}
	if o.LongTurnAge <= 0 {
		o.LongTurnAge = 2 * time.Hour
	}
	if o.UnknownRateThreshold <= 0 {
		o.UnknownRateThreshold = 0.20
	}
	if o.MismatchRateThreshold < 0 {
		o.MismatchRateThreshold = 0
	}
	return o
}

// QueryAttention returns prioritized recommendations derived from current worklog state.
func QueryAttention(d *db.DB, f Filter, opts AttentionOptions) (*AttentionSnapshot, error) {
	opts = opts.withDefaults()

	var items []AttentionItem
	var err error
	if items, err = appendStaleBlockers(d, f, opts, items); err != nil {
		return nil, err
	}
	if items, err = appendDecisionsWithoutEvidence(d, f, items); err != nil {
		return nil, err
	}
	if items, err = appendActiveTurnsWithoutEvidence(d, f, opts, items); err != nil {
		return nil, err
	}
	if items, err = appendDriftAlerts(d, f, items); err != nil {
		return nil, err
	}
	if items, err = appendLongTurnsWithoutOutcome(d, f, opts, items); err != nil {
		return nil, err
	}
	if f.Agent == "" {
		if items, err = appendAgentAttributionWarnings(d, f, opts, items); err != nil {
			return nil, err
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		si, sj := severityRank(items[i].Severity), severityRank(items[j].Severity)
		if si != sj {
			return si < sj
		}
		if items[i].Rule != items[j].Rule {
			return items[i].Rule < items[j].Rule
		}
		return items[i].SourceID < items[j].SourceID
	})

	return &AttentionSnapshot{
		Window:  f.Window,
		Agent:   f.Agent,
		Summary: summarizeAttention(items),
		Items:   items,
	}, nil
}

func appendStaleBlockers(d *db.DB, f Filter, opts AttentionOptions, items []AttentionItem) ([]AttentionItem, error) {
	q := `SELECT b.id, b.title, b.opened_at, b.accumulated_seconds
	      FROM blockers b
	      LEFT JOIN entries e ON e.id = b.entry_id
	      LEFT JOIN turns t ON t.id = b.turn_id
	      WHERE b.status = 'open'`
	var args []any
	q, args = appendWindowFilter(q, args, f, "b.opened_at")
	q, args = appendAgentFilter(q, args, f, "COALESCE(e.agent_id, t.agent_id, '')")
	q += " ORDER BY b.opened_at"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("attention stale blockers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, openedAt string
		var accumulated int64
		if err := rows.Scan(&id, &title, &openedAt, &accumulated); err != nil {
			return nil, fmt.Errorf("attention stale blockers scan: %w", err)
		}
		age := accumulated
		if age <= 0 {
			age = secondsSince(openedAt, opts.Now)
		}
		if age < int64(opts.OpenBlockerAge.Seconds()) {
			continue
		}
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleStaleBlocker, id),
			Severity:          AttentionSeverityHigh,
			Rule:              AttentionRuleStaleBlocker,
			SourceType:        "blocker",
			SourceID:          id,
			Title:             title,
			Reason:            fmt.Sprintf("열린 지 %s 지난 블로커입니다.", formatAttentionDuration(age)),
			RecommendedAction: "blocker-resolve로 해결하거나 현재 막힌 원인과 다음 대기 조건을 갱신하세요.",
			AgeSeconds:        int64Ptr(age),
		})
	}
	return items, rows.Err()
}

func appendDecisionsWithoutEvidence(d *db.DB, f Filter, items []AttentionItem) ([]AttentionItem, error) {
	q := `SELECT e.id, e.title
	      FROM entries e
	      LEFT JOIN decision_evidence_links del ON del.decision_entry_id = e.id
	      LEFT JOIN decision_status ds ON ds.decision_entry_id = e.id
	      WHERE e.kind = 'decision'
	        AND del.decision_entry_id IS NULL
	        AND COALESCE(ds.status, 'accepted') NOT IN ('superseded', 'rejected')`
	var args []any
	q, args = appendWindowFilter(q, args, f, "e.created_at")
	q, args = appendAgentFilter(q, args, f, "COALESCE(e.agent_id, '')")
	q += " ORDER BY e.created_at"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("attention decisions without evidence: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("attention decisions without evidence scan: %w", err)
		}
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleDecisionWithoutEvidence, id),
			Severity:          AttentionSeverityHigh,
			Rule:              AttentionRuleDecisionWithoutEvidence,
			SourceType:        "decision",
			SourceID:          id,
			Title:             title,
			Reason:            "유효한 결정에 연결된 evidence가 없습니다.",
			RecommendedAction: "ntts-flightlog evidence --link로 근거를 연결하거나 decision-supersede로 결정을 정리하세요.",
		})
	}
	return items, rows.Err()
}

func appendActiveTurnsWithoutEvidence(d *db.DB, f Filter, opts AttentionOptions, items []AttentionItem) ([]AttentionItem, error) {
	q := `SELECT t.id, t.sequence_no, COALESCE(t.title, ''), t.started_at,
	             COUNT(e.id) AS entry_count,
	             COALESCE(SUM(CASE WHEN e.kind = 'evidence' THEN 1 ELSE 0 END), 0) AS evidence_count
	      FROM turns t
	      LEFT JOIN entries e ON e.turn_id = t.id
	      WHERE t.status = 'active'`
	var args []any
	q, args = appendWindowFilter(q, args, f, "t.started_at")
	q, args = appendAgentFilter(q, args, f, "COALESCE(t.agent_id, '')")
	q += " GROUP BY t.id, t.sequence_no, t.title, t.started_at ORDER BY t.started_at"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("attention active turns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, startedAt string
		var sequenceNo, entryCount, evidenceCount int
		if err := rows.Scan(&id, &sequenceNo, &title, &startedAt, &entryCount, &evidenceCount); err != nil {
			return nil, fmt.Errorf("attention active turns scan: %w", err)
		}
		age := secondsSince(startedAt, opts.Now)
		if evidenceCount > 0 || (entryCount < opts.ActiveTurnEntries && age < int64(opts.ActiveTurnAge.Seconds())) {
			continue
		}
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleActiveTurnWithoutEvidence, id),
			Severity:          AttentionSeverityMedium,
			Rule:              AttentionRuleActiveTurnWithoutEvidence,
			SourceType:        "turn",
			SourceID:          id,
			Title:             turnAttentionTitle(sequenceNo, title),
			Reason:            fmt.Sprintf("진행 중인 턴에 evidence가 없고 항목 %d개, 경과 %s 상태입니다.", entryCount, formatAttentionDuration(age)),
			RecommendedAction: "검증 근거를 evidence로 남기거나 turn-end로 결과를 닫으세요.",
			AgeSeconds:        int64Ptr(age),
		})
	}
	return items, rows.Err()
}

func appendDriftAlerts(d *db.DB, f Filter, items []AttentionItem) ([]AttentionItem, error) {
	q := `SELECT id, sequence_no, COALESCE(title, ''), COALESCE(drift_alerts, 0)
	      FROM turns
	      WHERE COALESCE(drift_alerts, 0) > 0`
	var args []any
	q, args = appendWindowFilter(q, args, f, "started_at")
	q, args = appendAgentFilter(q, args, f, "COALESCE(agent_id, '')")
	q += " ORDER BY started_at"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("attention drift alerts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title string
		var sequenceNo, driftAlerts int
		if err := rows.Scan(&id, &sequenceNo, &title, &driftAlerts); err != nil {
			return nil, fmt.Errorf("attention drift alerts scan: %w", err)
		}
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleDriftAlert, id),
			Severity:          AttentionSeverityMedium,
			Rule:              AttentionRuleDriftAlert,
			SourceType:        "turn",
			SourceID:          id,
			Title:             turnAttentionTitle(sequenceNo, title),
			Reason:            fmt.Sprintf("Turn Intent Anchor drift alert가 %d회 기록되었습니다.", driftAlerts),
			RecommendedAction: "refresh-anchor로 의도와 완료조건을 다시 확인하고 필요한 경우 현재 턴 범위를 갱신하세요.",
		})
	}
	return items, rows.Err()
}

func appendLongTurnsWithoutOutcome(d *db.DB, f Filter, opts AttentionOptions, items []AttentionItem) ([]AttentionItem, error) {
	q := `SELECT id, sequence_no, COALESCE(title, ''), started_at,
	             COALESCE(ended_at, ''), status, elapsed_ms
	      FROM turns
	      WHERE COALESCE(outcome, '') = ''`
	var args []any
	q, args = appendWindowFilter(q, args, f, "started_at")
	q, args = appendAgentFilter(q, args, f, "COALESCE(agent_id, '')")
	q += " ORDER BY started_at"

	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("attention long turns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, title, startedAt, endedAt, status string
		var sequenceNo int
		var elapsed sql.NullInt64
		if err := rows.Scan(&id, &sequenceNo, &title, &startedAt, &endedAt, &status, &elapsed); err != nil {
			return nil, fmt.Errorf("attention long turns scan: %w", err)
		}
		var age int64
		switch {
		case status == db.TurnStatusActive || endedAt == "":
			age = secondsSince(startedAt, opts.Now)
		case elapsed.Valid:
			age = elapsed.Int64 / 1000
		default:
			age = secondsBetween(startedAt, endedAt)
		}
		if age < int64(opts.LongTurnAge.Seconds()) {
			continue
		}
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleLongTurnWithoutOutcome, id),
			Severity:          AttentionSeverityLow,
			Rule:              AttentionRuleLongTurnWithoutOutcome,
			SourceType:        "turn",
			SourceID:          id,
			Title:             turnAttentionTitle(sequenceNo, title),
			Reason:            fmt.Sprintf("긴 턴(%s)에 명시적 outcome이 없습니다.", formatAttentionDuration(age)),
			RecommendedAction: "turn-end 요약을 보강해 결과와 검증 상태를 남기세요.",
			AgeSeconds:        int64Ptr(age),
		})
	}
	return items, rows.Err()
}

func appendAgentAttributionWarnings(d *db.DB, f Filter, opts AttentionOptions, items []AttentionItem) ([]AttentionItem, error) {
	snap, err := QueryAgentStats(d, Filter{Window: f.Window})
	if err != nil {
		return nil, err
	}
	if snap.Summary.TotalSessions == 0 {
		return items, nil
	}
	if snap.Summary.UnknownSessions > 0 && snap.Summary.UnknownRate >= opts.UnknownRateThreshold {
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleAgentAttribution, "unknown"),
			Severity:          AttentionSeverityMedium,
			Rule:              AttentionRuleAgentAttribution,
			SourceType:        "agent_stats",
			SourceID:          "unknown",
			Title:             "agent unknown 비율 높음",
			Reason:            fmt.Sprintf("agent 자동감지 unknown 세션이 %d/%d개(%.1f%%)입니다.", snap.Summary.UnknownSessions, snap.Summary.TotalSessions, snap.Summary.UnknownRate*100),
			RecommendedAction: "--agent 플래그 또는 agent-stats로 세션 attribution을 확인하세요.",
		})
	}
	if snap.Summary.MismatchSessions > 0 && snap.Summary.MismatchRate > opts.MismatchRateThreshold {
		items = append(items, AttentionItem{
			ID:                attentionID(AttentionRuleAgentAttribution, "mismatch"),
			Severity:          AttentionSeverityHigh,
			Rule:              AttentionRuleAgentAttribution,
			SourceType:        "agent_stats",
			SourceID:          "mismatch",
			Title:             "agent override mismatch",
			Reason:            fmt.Sprintf("agent override와 자동감지가 다른 세션이 %d/%d개(%.1f%%)입니다.", snap.Summary.MismatchSessions, snap.Summary.TotalSessions, snap.Summary.MismatchRate*100),
			RecommendedAction: "의도한 override인지 확인하고 잘못 기록된 세션은 다음 작업부터 명시적 --agent로 보정하세요.",
		})
	}
	return items, nil
}

func appendWindowFilter(q string, args []any, f Filter, column string) (string, []any) {
	if ws := f.windowExpr(); ws != "" {
		q += " AND " + column + " >= " + ws
	}
	return q, args
}

func appendAgentFilter(q string, args []any, f Filter, expr string) (string, []any) {
	if f.Agent != "" {
		q += " AND " + expr + " = ?"
		args = append(args, f.Agent)
	}
	return q, args
}

func summarizeAttention(items []AttentionItem) AttentionSummary {
	var summary AttentionSummary
	summary.Total = len(items)
	for _, item := range items {
		switch item.Severity {
		case AttentionSeverityHigh:
			summary.High++
		case AttentionSeverityMedium:
			summary.Medium++
		default:
			summary.Low++
		}
	}
	return summary
}

func severityRank(severity string) int {
	switch severity {
	case AttentionSeverityHigh:
		return 0
	case AttentionSeverityMedium:
		return 1
	default:
		return 2
	}
}

func secondsSince(ts string, now time.Time) int64 {
	t, err := parseMetricTime(ts)
	if err != nil {
		return 0
	}
	secs := int64(now.Sub(t).Seconds())
	if secs < 0 {
		return 0
	}
	return secs
}

func secondsBetween(start, end string) int64 {
	startTime, err := parseMetricTime(start)
	if err != nil {
		return 0
	}
	endTime, err := parseMetricTime(end)
	if err != nil {
		return 0
	}
	secs := int64(endTime.Sub(startTime).Seconds())
	if secs < 0 {
		return 0
	}
	return secs
}

func parseMetricTime(ts string) (time.Time, error) {
	if ts == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.UTC(), nil
	}
	return time.Parse("2006-01-02T15:04:05Z", ts)
}

func attentionID(rule, sourceID string) string {
	return rule + ":" + sourceID
}

func turnAttentionTitle(sequenceNo int, title string) string {
	if title == "" {
		title = "(제목 없음)"
	}
	if sequenceNo <= 0 {
		return title
	}
	return fmt.Sprintf("turn-%d: %s", sequenceNo, title)
}

func int64Ptr(v int64) *int64 {
	return &v
}

func formatAttentionDuration(secs int64) string {
	if secs <= 0 {
		return "0s"
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// FormatAttentionJSON serializes attention output with a stable timestamp hook.
func FormatAttentionJSON(snap *AttentionSnapshot, generatedAt time.Time) ([]byte, error) {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	out := *snap
	out.GeneratedAt = generatedAt.UTC().Format("2006-01-02T15:04:05Z")
	if out.Items == nil {
		out.Items = []AttentionItem{}
	}
	out.Summary = summarizeAttention(out.Items)
	return json.MarshalIndent(out, "", "  ")
}
