package metrics

import "fmt"

// MetricHighlight is a concise interpretation of one report metric for sharing.
type MetricHighlight struct {
	Metric         string `json:"metric"`
	Label          string `json:"label"`
	Value          string `json:"value"`
	Interpretation string `json:"interpretation"`
}

// BuildMetricHighlights turns raw report metrics into shareable status bullets.
func BuildMetricHighlights(snap *Snapshot) []MetricHighlight {
	if snap == nil {
		return nil
	}
	return []MetricHighlight{
		turnDurationHighlight(snap.TurnDurations),
		blockerAccumulationHighlight(snap.BlockerAccumulations),
		agentCompletionHighlight(snap.AgentCompletion),
		agentBlockerFreqHighlight(snap.AgentBlockerFreq),
		evidenceBoundHighlight(snap.EvidenceBound),
	}
}

func turnDurationHighlight(rows []TurnDuration) MetricHighlight {
	var count int
	var total int64
	for _, row := range rows {
		if row.ElapsedMS == nil {
			continue
		}
		count++
		total += *row.ElapsedMS
	}
	if count == 0 {
		return MetricHighlight{
			Metric:         "turn_duration",
			Label:          "Turn 소요시간",
			Value:          "완료 턴 없음",
			Interpretation: "아직 완료된 턴이 없어 소요시간 경향을 판단할 수 없습니다.",
		}
	}
	avgSeconds := total / int64(count) / 1000
	return MetricHighlight{
		Metric:         "turn_duration",
		Label:          "Turn 소요시간",
		Value:          fmt.Sprintf("완료 턴 %d개, 평균 %s", count, formatAttentionDuration(avgSeconds)),
		Interpretation: "턴 단위 작업 크기와 종료 리듬을 확인하세요.",
	}
}

func blockerAccumulationHighlight(rows []BlockerAccumulation) MetricHighlight {
	var openCount int
	var maxSeconds int64
	for _, row := range rows {
		if row.ClosedAt == "" {
			openCount++
		}
		if row.AccumulatedSeconds > maxSeconds {
			maxSeconds = row.AccumulatedSeconds
		}
	}
	if len(rows) == 0 {
		return MetricHighlight{
			Metric:         "blocker_accumulation",
			Label:          "Blocker 누적시간",
			Value:          "블로커 없음",
			Interpretation: "현재 기록된 블로커가 없습니다.",
		}
	}
	return MetricHighlight{
		Metric:         "blocker_accumulation",
		Label:          "Blocker 누적시간",
		Value:          fmt.Sprintf("열림 %d개, 최대 누적 %s", openCount, formatAttentionDuration(maxSeconds)),
		Interpretation: "열린 블로커가 있으면 공유 대상에게 필요한 도움을 명확히 요청하세요.",
	}
}

func agentCompletionHighlight(rows []AgentCompletion) MetricHighlight {
	if len(rows) == 0 {
		return MetricHighlight{
			Metric:         "agent_completion",
			Label:          "Agent 완료율",
			Value:          "agent별 턴 없음",
			Interpretation: "agent별 완료율을 판단할 수 있는 턴 데이터가 없습니다.",
		}
	}
	worst := rows[0]
	for _, row := range rows[1:] {
		if row.CompletionRate < worst.CompletionRate {
			worst = row
		}
	}
	return MetricHighlight{
		Metric:         "agent_completion",
		Label:          "Agent 완료율",
		Value:          fmt.Sprintf("%s %.1f%% (%d/%d)", metricAgentLabel(worst.AgentID), worst.CompletionRate*100, worst.CompleteCount, worst.TotalCount),
		Interpretation: "완료율이 낮은 agent 흐름은 범위 분할이나 handoff 보강이 필요한지 확인하세요.",
	}
}

func agentBlockerFreqHighlight(rows []AgentBlockerFreq) MetricHighlight {
	if len(rows) == 0 {
		return MetricHighlight{
			Metric:         "agent_blocker_freq",
			Label:          "Agent blocker 빈도",
			Value:          "agent별 블로커 없음",
			Interpretation: "agent별 blocker 빈도를 판단할 수 있는 턴 데이터가 없습니다.",
		}
	}
	worst := rows[0]
	for _, row := range rows[1:] {
		if row.BlockerFreq > worst.BlockerFreq {
			worst = row
		}
	}
	return MetricHighlight{
		Metric:         "agent_blocker_freq",
		Label:          "Agent blocker 빈도",
		Value:          fmt.Sprintf("%s %.3f (%d blockers/%d turns)", metricAgentLabel(worst.AgentID), worst.BlockerFreq, worst.BlockerCount, worst.TurnCount),
		Interpretation: "blocker 빈도가 높은 흐름은 외부 의존성이나 작업 분해 문제를 먼저 점검하세요.",
	}
}

func evidenceBoundHighlight(row EvidenceBoundDecisions) MetricHighlight {
	if row.TotalCount == 0 {
		return MetricHighlight{
			Metric:         "evidence_bound_decisions",
			Label:          "Evidence-bound decision",
			Value:          "결정 없음",
			Interpretation: "공유 가능한 결정 근거가 아직 없습니다.",
		}
	}
	return MetricHighlight{
		Metric:         "evidence_bound_decisions",
		Label:          "Evidence-bound decision",
		Value:          fmt.Sprintf("%d/%d decisions (%.1f%%)", row.LinkedCount, row.TotalCount, row.Ratio*100),
		Interpretation: "근거가 없는 결정은 공유 전에 evidence 링크를 보강하세요.",
	}
}

func metricAgentLabel(agentID string) string {
	if agentID == "" {
		return "unknown"
	}
	return agentID
}
