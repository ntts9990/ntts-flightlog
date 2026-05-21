package views

import (
	"fmt"
	"sort"
	"strings"
)

const visualBarWidth = 24

// RenderVisualReport returns an ASCII-first progress visualization.
// Solid "=" means completed or satisfied work; dotted "." means remaining work.
func RenderVisualReport(data *WorklogData) string {
	var sb strings.Builder

	sb.WriteString(Bold + ColorSection + "## 리포트 시각화" + Reset + "\n\n")
	sb.WriteString(Dim + "실선(=)은 달성, 점선(.)은 남은 목표를 뜻합니다." + Reset + "\n\n")

	if data == nil {
		sb.WriteString(Dim + "데이터 없음" + Reset + "\n")
		return sb.String()
	}

	turns := summarizeTurns(data.Turns)
	decisions := summarizeDecisions(data)
	blockers := summarizeBlockers(data)
	goalsDone, goalsTotal := summarizeGoalTurns(data.Turns)
	if goalsTotal == 0 {
		goalsDone = turns.completed
		goalsTotal = len(data.Turns)
	}

	sb.WriteString(Bold + "목표 레일" + Reset + "\n")
	writeProgressLine(&sb, "턴 완료", goalsDone, goalsTotal)
	writeProgressLine(&sb, "결정 근거", decisions.linked, decisions.total)
	writeProgressLine(&sb, "블로커 소거", blockers.resolved, blockers.open+blockers.resolved+blockers.other)
	fmt.Fprintf(&sb, "%s진행 중 턴: %d · 대기/기타: %d%s\n\n", Dim, turns.active, turns.other, Reset)

	sb.WriteString(Bold + "엔트리 믹스" + Reset + "\n")
	writeEntryMix(&sb, data.Entries)

	sb.WriteString("\n" + Bold + "Lane 스택" + Reset + "\n")
	writeLaneStack(&sb, data.Turns)

	sb.WriteString("\n" + Bold + "템플릿 범례" + Reset + "\n")
	sb.WriteString(Dim + "[=========...............] 목표형 막대\n" + Reset)
	sb.WriteString(Dim + "entry/decision/evidence/blocker는 같은 폭 안에서 비율로 표시\n" + Reset)

	return sb.String()
}

func summarizeGoalTurns(turns []Turn) (done int, total int) {
	for _, t := range turns {
		if !turnHasGoal(t) {
			continue
		}
		total++
		if turnComplete(t) {
			done++
		}
	}
	return done, total
}

func turnHasGoal(t Turn) bool {
	return (t.DoneWhen.Valid && strings.TrimSpace(t.DoneWhen.String) != "") ||
		(t.Intent.Valid && strings.TrimSpace(t.Intent.String) != "")
}

func turnComplete(t Turn) bool {
	switch t.Status {
	case "complete", "completed", "closed":
		return true
	default:
		return t.EndedAt.Valid
	}
}

func writeProgressLine(sb *strings.Builder, label string, done, total int) {
	bar := progressBar(done, total, visualBarWidth)
	fmt.Fprintf(sb, "%-10s %s %d/%d %s\n", label, bar, clamp(done, 0, total), total, percent(done, total))
}

func progressBar(done, total, width int) string {
	if width <= 0 {
		return "[]"
	}
	if total <= 0 {
		return "[" + strings.Repeat(".", width) + "]"
	}
	done = clamp(done, 0, total)
	filled := int(float64(done) / float64(total) * float64(width))
	if done > 0 && filled == 0 {
		filled = 1
	}
	if done == total {
		filled = width
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat(".", width-filled) + "]"
}

func clamp(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func writeEntryMix(sb *strings.Builder, entries []Entry) {
	counts := map[string]int{
		"entry":    0,
		"decision": 0,
		"evidence": 0,
		"blocker":  0,
	}
	for _, e := range entries {
		if _, ok := counts[e.Kind]; ok {
			counts[e.Kind]++
		}
	}
	total := counts["entry"] + counts["decision"] + counts["evidence"] + counts["blocker"]
	if total == 0 {
		sb.WriteString(Dim + "기록된 엔트리 없음" + Reset + "\n")
		return
	}
	parts := []struct {
		label string
		char  string
		count int
	}{
		{"entry", "E", counts["entry"]},
		{"decision", "D", counts["decision"]},
		{"evidence", "V", counts["evidence"]},
		{"blocker", "B", counts["blocker"]},
	}
	for _, part := range parts {
		bar := scaledBar(part.char, part.count, total, visualBarWidth)
		fmt.Fprintf(sb, "%-10s %s %d\n", part.label, bar, part.count)
	}
}

func scaledBar(char string, count, total, width int) string {
	if total <= 0 || count <= 0 {
		return "[" + strings.Repeat(".", width) + "]"
	}
	filled := int(float64(count) / float64(total) * float64(width))
	if filled == 0 {
		filled = 1
	}
	return "[" + strings.Repeat(char, filled) + strings.Repeat(".", width-filled) + "]"
}

func writeLaneStack(sb *strings.Builder, turns []Turn) {
	counts := make(map[string]int)
	for _, t := range turns {
		lane := "default"
		if t.Lane.Valid && strings.TrimSpace(t.Lane.String) != "" {
			lane = t.Lane.String
		}
		counts[lane]++
	}
	if len(counts) == 0 {
		sb.WriteString(Dim + "lane 데이터 없음" + Reset + "\n")
		return
	}
	lanes := make([]string, 0, len(counts))
	total := 0
	for lane, count := range counts {
		lanes = append(lanes, lane)
		total += count
	}
	sort.Strings(lanes)
	for _, lane := range lanes {
		bar := scaledBar("#", counts[lane], total, visualBarWidth)
		fmt.Fprintf(sb, "%-10s %s %d\n", truncateLabel(lane, 10), bar, counts[lane])
	}
}

func truncateLabel(label string, width int) string {
	if len(label) <= width {
		return label
	}
	if width <= 1 {
		return label[:width]
	}
	return label[:width-1] + "."
}
