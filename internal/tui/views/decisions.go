package views

import (
	"fmt"
	"strings"
)

// RenderDecisions returns an ADR-lite decision log with turn context and
// evidence signals.
func RenderDecisions(data *WorklogData) string {
	var sb strings.Builder
	found := false

	if data == nil {
		return "(결정 사항이 아직 없습니다. flightlog decision으로 기록하세요.)\n"
	}

	turnByID := make(map[string]Turn)
	for _, t := range data.Turns {
		turnByID[t.ID] = t
	}
	evidenceByTurn := make(map[string]int)
	for _, e := range data.Entries {
		if e.Kind == "evidence" && e.TurnID.Valid {
			evidenceByTurn[e.TurnID.String]++
		}
	}
	linkedEvidence := make(map[string]int)
	for _, link := range data.DecisionEvidenceLinks {
		linkedEvidence[link.DecisionEntryID]++
	}
	stateByDecision := make(map[string]DecisionState)
	for _, state := range data.DecisionStates {
		stateByDecision[state.DecisionEntryID] = state
	}

	sb.WriteString(Bold + ColorSection + "## 결정 기록" + Reset + "\n")
	sb.WriteString(Dim + "되돌리기 비싼 선택과 그 근거를 모읍니다." + Reset + "\n\n")

	activeDecisions := make([]Entry, 0)
	supersededDecisions := make([]Entry, 0)
	for _, e := range data.Entries {
		if e.Kind != "decision" {
			continue
		}
		found = true
		if decisionStatus(e, stateByDecision) == "superseded" {
			supersededDecisions = append(supersededDecisions, e)
		} else {
			activeDecisions = append(activeDecisions, e)
		}
	}

	writeDecisionGroup(&sb, "유효한 결정", activeDecisions, turnByID, evidenceByTurn, linkedEvidence, stateByDecision)
	writeDecisionGroup(&sb, "대체된 결정", supersededDecisions, turnByID, evidenceByTurn, linkedEvidence, stateByDecision)

	if !found {
		return "(결정 사항이 아직 없습니다. flightlog decision으로 기록하세요.)\n"
	}
	return sb.String()
}

func writeDecisionGroup(sb *strings.Builder, label string, entries []Entry, turnByID map[string]Turn, evidenceByTurn map[string]int, linkedEvidence map[string]int, stateByDecision map[string]DecisionState) {
	if len(entries) == 0 {
		return
	}
	sb.WriteString(Bold + ColorDecision + label + Reset + "\n")
	for _, e := range entries {
		state := stateByDecision[e.ID]
		status := decisionStatus(e, stateByDecision)
		fmt.Fprintf(sb, "%s%s◆ %s  [decision]%s\n", Bold, ColorDecision, e.CreatedAt, Reset)
		fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorDecision, e.Title, Reset)
		fmt.Fprintf(sb, "%s  id: %s%s\n", Dim, shortID(e.ID), Reset)
		fmt.Fprintf(sb, "%s  상태: %s%s\n", Dim, status, Reset)
		if e.TurnID.Valid {
			if t, ok := turnByID[e.TurnID.String]; ok {
				title := "(제목 없음)"
				if t.Title.Valid && t.Title.String != "" {
					title = t.Title.String
				}
				fmt.Fprintf(sb, "%s  turn-%d: %s%s\n", Dim, t.SequenceNo, title, Reset)
			}
		}
		fmt.Fprintf(sb, "%s  근거: linked %d · same-turn %d%s\n",
			Dim, linkedEvidence[e.ID], sameTurnEvidenceCount(e, evidenceByTurn), Reset)
		if state.SupersededByEntryID.Valid && state.SupersededByEntryID.String != "" {
			fmt.Fprintf(sb, "%s  대체됨: %s%s\n", Dim, shortID(state.SupersededByEntryID.String), Reset)
		}
		if state.Rationale.Valid && state.Rationale.String != "" {
			fmt.Fprintf(sb, "%s  사유: %s%s\n", Dim, state.Rationale.String, Reset)
		}
		if e.Detail.Valid && e.Detail.String != "" {
			WriteDetail(sb, e.Detail.String)
		}
		sb.WriteString("\n")
	}
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func sameTurnEvidenceCount(e Entry, evidenceByTurn map[string]int) int {
	if !e.TurnID.Valid || e.TurnID.String == "" {
		return 0
	}
	return evidenceByTurn[e.TurnID.String]
}

func decisionStatus(e Entry, stateByDecision map[string]DecisionState) string {
	if state, ok := stateByDecision[e.ID]; ok && state.Status != "" {
		return state.Status
	}
	return "accepted"
}
