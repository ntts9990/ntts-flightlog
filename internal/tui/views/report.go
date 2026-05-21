package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// RenderReport returns the report view content.
// It summarizes the loaded worklog into operator-facing signals for the sidecar.
func RenderReport(data *WorklogData) string {
	var sb strings.Builder

	sb.WriteString(Bold + ColorSection + "## 리포트" + Reset + "\n\n")
	sb.WriteString(Dim + "현재 세션의 진행률, 결정 근거, 블로커 상태를 요약합니다." + Reset + "\n\n")

	if data == nil {
		sb.WriteString(Dim + "데이터 없음" + Reset + "\n")
		return sb.String()
	}

	entryCounts := make(map[string]int)
	for _, e := range data.Entries {
		entryCounts[e.Kind]++
	}
	turns := summarizeTurns(data.Turns)
	lanes := summarizeLanes(data.Turns)
	decisions := summarizeDecisions(data)
	blockers := summarizeBlockers(data)

	sb.WriteString(Bold + "작업량" + Reset + "\n")
	sb.WriteString(Dim + formatCount("세션", len(data.Sessions)) + Reset + "\n")
	sb.WriteString(Dim + formatCount("턴", len(data.Turns)) + Reset + "\n")
	sb.WriteString(Dim + formatCount("엔트리", len(data.Entries)) + Reset + "\n")
	fmt.Fprintf(&sb, "%s  entry %d · decision %d · evidence %d · blocker %d%s\n\n",
		Dim, entryCounts["entry"], entryCounts["decision"], entryCounts["evidence"], entryCounts["blocker"], Reset)

	writeAttentionSection(&sb, data.Attention)

	sb.WriteString(Bold + "턴 진행" + Reset + "\n")
	fmt.Fprintf(&sb, "%s완료 턴: %d · 진행 중: %d · 기타: %d%s\n", Dim, turns.completed, turns.active, turns.other, Reset)
	if turns.completedWithDuration > 0 {
		fmt.Fprintf(&sb, "%s평균 완료 시간: %s%s\n", Dim, formatDurationMS(turns.totalElapsedMS/int64(turns.completedWithDuration)), Reset)
	}
	if len(lanes) > 0 {
		fmt.Fprintf(&sb, "%sLane: %s%s\n", Dim, strings.Join(lanes, " · "), Reset)
	}
	sb.WriteString("\n")

	sb.WriteString(Bold + "결정 품질" + Reset + "\n")
	fmt.Fprintf(&sb, "%s유효 %d · 대체됨 %d · 거절 %d · 기타 %d%s\n",
		Dim, decisions.accepted, decisions.superseded, decisions.rejected, decisions.other, Reset)
	fmt.Fprintf(&sb, "%s근거 연결 결정: %d/%d (%s)%s\n\n",
		Dim, decisions.linked, decisions.total, percent(decisions.linked, decisions.total), Reset)

	sb.WriteString(Bold + "블로커" + Reset + "\n")
	fmt.Fprintf(&sb, "%s열린 블로커: %d · 해결됨: %d · 기타: %d%s\n",
		Dim, blockers.open, blockers.resolved, blockers.other, Reset)
	if blockers.untrackedEntries > 0 {
		fmt.Fprintf(&sb, "%s상태 미분류 blocker 엔트리: %d%s\n", Dim, blockers.untrackedEntries, Reset)
	}

	return sb.String()
}

func summarizeLanes(turns []Turn) []string {
	counts := make(map[string]int)
	for _, t := range turns {
		if t.Lane.Valid && t.Lane.String != "" {
			counts[t.Lane.String]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	out := make([]string, 0, len(counts))
	for lane, count := range counts {
		out = append(out, fmt.Sprintf("%s %d", lane, count))
	}
	sort.Strings(out)
	return out
}

func writeAttentionSection(sb *strings.Builder, items []metrics.AttentionItem) {
	sb.WriteString(Bold + "주의 필요" + Reset + "\n")
	if len(items) == 0 {
		fmt.Fprintf(sb, "%s현재 주의가 필요한 항목 없음%s\n\n", Dim, Reset)
		return
	}
	limit := len(items)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		item := items[i]
		fmt.Fprintf(sb, "%s[%s] %s%s\n", ColorBlocker, reportSeverityLabel(item.Severity), item.Title, Reset)
		fmt.Fprintf(sb, "%s  이유: %s%s\n", Dim, item.Reason, Reset)
		fmt.Fprintf(sb, "%s  다음: %s%s\n", Dim, item.RecommendedAction, Reset)
	}
	if hidden := len(items) - limit; hidden > 0 {
		fmt.Fprintf(sb, "%s  외 %d개%s\n", Dim, hidden, Reset)
	}
	sb.WriteString("\n")
}

func reportSeverityLabel(severity string) string {
	switch severity {
	case metrics.AttentionSeverityHigh:
		return "높음"
	case metrics.AttentionSeverityMedium:
		return "중간"
	default:
		return "낮음"
	}
}

func formatCount(label string, n int) string {
	return label + ": " + itoa(n) + "개"
}

type turnReportSummary struct {
	active                int
	completed             int
	other                 int
	completedWithDuration int
	totalElapsedMS        int64
}

func summarizeTurns(turns []Turn) turnReportSummary {
	var summary turnReportSummary
	for _, t := range turns {
		switch t.Status {
		case "active", "":
			summary.active++
		case "complete", "completed", "closed":
			summary.completed++
			if t.ElapsedMs.Valid {
				summary.completedWithDuration++
				summary.totalElapsedMS += t.ElapsedMs.Int64
			}
		default:
			if t.EndedAt.Valid {
				summary.completed++
				if t.ElapsedMs.Valid {
					summary.completedWithDuration++
					summary.totalElapsedMS += t.ElapsedMs.Int64
				}
			} else {
				summary.other++
			}
		}
	}
	return summary
}

type decisionReportSummary struct {
	total      int
	linked     int
	accepted   int
	superseded int
	rejected   int
	other      int
}

func summarizeDecisions(data *WorklogData) decisionReportSummary {
	linked := make(map[string]bool)
	for _, link := range data.DecisionEvidenceLinks {
		linked[link.DecisionEntryID] = true
	}
	states := make(map[string]DecisionState)
	for _, state := range data.DecisionStates {
		states[state.DecisionEntryID] = state
	}

	var summary decisionReportSummary
	for _, e := range data.Entries {
		if e.Kind != "decision" {
			continue
		}
		summary.total++
		if linked[e.ID] {
			summary.linked++
		}
		switch decisionStatus(e, states) {
		case "accepted":
			summary.accepted++
		case "superseded":
			summary.superseded++
		case "rejected":
			summary.rejected++
		default:
			summary.other++
		}
	}
	return summary
}

type blockerReportSummary struct {
	open             int
	resolved         int
	other            int
	untrackedEntries int
}

func summarizeBlockers(data *WorklogData) blockerReportSummary {
	var summary blockerReportSummary
	for _, b := range data.Blockers {
		switch b.Status {
		case "open", "":
			summary.open++
		case "resolved", "closed":
			summary.resolved++
		default:
			summary.other++
		}
	}
	if len(data.Blockers) == 0 {
		for _, e := range data.Entries {
			if e.Kind == "blocker" {
				summary.untrackedEntries++
			}
		}
	}
	return summary
}

func percent(n, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(n)*100/float64(total))
}

// itoa converts an int to a decimal string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
