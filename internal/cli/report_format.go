package cli

import (
	"fmt"
	"strings"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

// formatText renders the snapshot as a Korean-labelled text report.
// Korean default labels are required per plan spec (Korean default labels).
func formatText(snap *metrics.Snapshot, window, agent string) string {
	var b strings.Builder

	windowLabel := map[string]string{
		"day":  "오늘 (24h)",
		"week": "이번 주 (7d)",
		"all":  "전체",
		"":     "전체",
	}[window]
	agentLabel := "전체"
	if agent != "" {
		agentLabel = agent
	}

	const sep = "═══════════════════════════════════════════════════════"
	const thin = "───────────────────────────────────────────────────────"

	b.WriteString(sep + "\n")
	b.WriteString("  NTTS Flightlog 메트릭 리포트\n")
	fmt.Fprintf(&b, "  [창: %s]  [에이전트: %s]\n", windowLabel, agentLabel)
	b.WriteString(sep + "\n\n")

	// ── 1. Turn 소요시간 분포 ─────────────────────────────────────────────────
	b.WriteString("■ 1. Turn 소요시간 분포 (metric_turn_duration)\n")
	b.WriteString(thin + "\n")
	if len(snap.TurnDurations) == 0 {
		b.WriteString("  (데이터 없음)\n")
	} else {
		for _, td := range snap.TurnDurations {
			elapsed := "—"
			if td.ElapsedMS != nil {
				elapsed = fmtDurationMS(*td.ElapsedMS)
			}
			agentStr := td.AgentID
			if agentStr == "" {
				agentStr = "unknown"
			}
			fmt.Fprintf(&b, "  %-20s  %-10s  %s\n", td.TurnID, agentStr, elapsed)
		}
	}
	b.WriteString("\n")

	// ── 2. Blocker 누적시간 ───────────────────────────────────────────────────
	b.WriteString("■ 2. Blocker 누적시간 (metric_blocker_accumulation)\n")
	b.WriteString(thin + "\n")
	if len(snap.BlockerAccumulations) == 0 {
		b.WriteString("  (데이터 없음)\n")
	} else {
		for _, ba := range snap.BlockerAccumulations {
			closedStr := "열림 중"
			if ba.ClosedAt != "" {
				closedStr = "닫힘: " + ba.ClosedAt
			}
			fmt.Fprintf(&b, "  %-8s  열림: %-24s  %s  누적: %s\n",
				ba.BlockerID, ba.OpenedAt, closedStr, fmtDurationSec(ba.AccumulatedSeconds))
		}
	}
	b.WriteString("\n")

	// ── 3. Agent별 Turn 완료율 ────────────────────────────────────────────────
	b.WriteString("■ 3. Agent별 Turn 완료율 (metric_agent_completion)\n")
	b.WriteString(thin + "\n")
	if len(snap.AgentCompletion) == 0 {
		b.WriteString("  (데이터 없음)\n")
	} else {
		for _, ac := range snap.AgentCompletion {
			fmt.Fprintf(&b, "  %-12s  %d / %d  (%.1f%%)\n",
				ac.AgentID, ac.CompleteCount, ac.TotalCount, ac.CompletionRate*100)
		}
	}
	b.WriteString("\n")

	// ── 4. Agent별 Blocker 빈도 ───────────────────────────────────────────────
	b.WriteString("■ 4. Agent별 Blocker 빈도 (metric_agent_blocker_freq)\n")
	b.WriteString(thin + "\n")
	if len(snap.AgentBlockerFreq) == 0 {
		b.WriteString("  (데이터 없음)\n")
	} else {
		for _, af := range snap.AgentBlockerFreq {
			fmt.Fprintf(&b, "  %-12s  %.3f  (%d blockers / %d turns)\n",
				af.AgentID, af.BlockerFreq, af.BlockerCount, af.TurnCount)
		}
	}
	b.WriteString("\n")

	// ── 5. Evidence-Bound Decision 비율 ───────────────────────────────────────
	b.WriteString("■ 5. Evidence가 붙은 Decision 비율 (metric_evidence_bound_decisions)\n")
	b.WriteString(thin + "\n")
	ebd := snap.EvidenceBound
	if ebd.TotalCount == 0 {
		b.WriteString("  (결정 없음)\n")
	} else {
		fmt.Fprintf(&b, "  %d / %d decisions에 근거 자료 연결  (%.1f%%)\n",
			ebd.LinkedCount, ebd.TotalCount, ebd.Ratio*100)
	}
	b.WriteString("\n")

	b.WriteString(sep + "\n")
	return b.String()
}

// fmtDurationMS formats milliseconds as a human-readable duration string.
func fmtDurationMS(ms int64) string {
	return fmtDurationSec(ms / 1000)
}

// fmtDurationSec formats a second count as a human-readable duration string.
func fmtDurationSec(secs int64) string {
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
