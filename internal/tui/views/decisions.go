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

	sb.WriteString(Bold + ColorSection + "## 결정 기록" + Reset + "\n")
	sb.WriteString(Dim + "되돌리기 비싼 선택과 그 근거를 모읍니다." + Reset + "\n\n")

	for _, e := range data.Entries {
		if e.Kind != "decision" {
			continue
		}
		found = true
		fmt.Fprintf(&sb, "%s%s◆ %s  [decision]%s\n", Bold, ColorDecision, e.CreatedAt, Reset)
		fmt.Fprintf(&sb, "%s%s  %s%s\n", Bold, ColorDecision, e.Title, Reset)
		if e.TurnID.Valid {
			if t, ok := turnByID[e.TurnID.String]; ok {
				title := "(제목 없음)"
				if t.Title.Valid && t.Title.String != "" {
					title = t.Title.String
				}
				fmt.Fprintf(&sb, "%s  turn-%d: %s%s\n", Dim, t.SequenceNo, title, Reset)
			}
		}
		fmt.Fprintf(&sb, "%s  근거: linked %d · same-turn %d%s\n",
			Dim, linkedEvidence[e.ID], sameTurnEvidenceCount(e, evidenceByTurn), Reset)
		if e.Detail.Valid && e.Detail.String != "" {
			WriteDetail(&sb, e.Detail.String)
		}
		sb.WriteString("\n")
	}

	if !found {
		return "(결정 사항이 아직 없습니다. flightlog decision으로 기록하세요.)\n"
	}
	return sb.String()
}

func sameTurnEvidenceCount(e Entry, evidenceByTurn map[string]int) int {
	if !e.TurnID.Valid || e.TurnID.String == "" {
		return 0
	}
	return evidenceByTurn[e.TurnID.String]
}
