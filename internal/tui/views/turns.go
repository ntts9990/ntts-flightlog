package views

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RenderTurns returns a compact turn index. Unlike the flat live log, this view
// summarizes each turn as a bounded work unit instead of repeating full entries.
func RenderTurns(data *WorklogData, turnsDir string) string {
	if data == nil || len(data.Turns) == 0 {
		return "(turn 파일이 아직 없습니다. turn-start로 첫 턴을 시작하세요.)\n"
	}

	absTurns, _ := filepath.Abs(turnsDir)
	var sb strings.Builder

	// Index entries by turn ID.
	entriesByTurn := make(map[string][]Entry)
	for _, e := range data.Entries {
		if e.TurnID.Valid && e.TurnID.String != "" {
			entriesByTurn[e.TurnID.String] = append(entriesByTurn[e.TurnID.String], e)
		}
	}

	sb.WriteString(Bold + ColorSection + "## 턴 인덱스" + Reset + "\n")
	sb.WriteString(Dim + "작업 단위별 상태, 신호, 마지막 결과를 요약합니다." + Reset + "\n\n")

	for _, t := range data.Turns {
		title := "(제목 없음)"
		if t.Title.Valid && t.Title.String != "" {
			title = t.Title.String
		}

		entries := entriesByTurn[t.ID]
		counts := countEntriesByKind(entries)
		result := turnResult(t, entries)

		color := TurnColorFor(t.SequenceNo)
		url := fmt.Sprintf("file://%s/turn-%d.md", absTurns, t.SequenceNo)
		status := t.Status
		if status == "" {
			status = "active"
		}
		elapsed := "진행 중"
		if t.ElapsedMs.Valid {
			elapsed = formatDurationMS(t.ElapsedMs.Int64)
		} else if t.EndedAt.Valid {
			elapsed = "완료"
		}

		fmt.Fprintf(&sb, "%s%s# Turn %d · %s · %s%s\n", Bold, color, t.SequenceNo, status, elapsed, Reset)
		fmt.Fprintf(&sb, "%s%s  %s%s\n", Bold, color, osc8Link(url, title), Reset)
		fmt.Fprintf(&sb, "%s  시작: %s%s\n", Dim, t.StartedAt, Reset)
		fmt.Fprintf(&sb, "%s  신호: entry %d · decision %d · evidence %d · blocker %d%s\n",
			Dim, counts["entry"], counts["decision"], counts["evidence"], counts["blocker"], Reset)
		if t.Intent.Valid && t.Intent.String != "" {
			fmt.Fprintf(&sb, "%s  의도: %s%s\n", Dim, t.Intent.String, Reset)
		}
		fmt.Fprintf(&sb, "%s  결과: %s%s\n", Dim, result, Reset)
		sb.WriteString("\n")
	}

	return sb.String()
}

func turnResult(t Turn, entries []Entry) string {
	if t.Outcome.Valid && t.Outcome.String != "" {
		return t.Outcome.String
	}
	if len(entries) > 0 {
		return entries[len(entries)-1].Title
	}
	if t.Status == "active" {
		return "진행 중"
	}
	return "기록 없음"
}

func countEntriesByKind(entries []Entry) map[string]int {
	counts := map[string]int{
		"entry":    0,
		"decision": 0,
		"evidence": 0,
		"blocker":  0,
		"mode":     0,
	}
	for _, e := range entries {
		counts[e.Kind]++
	}
	return counts
}

func formatDurationMS(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	totalSeconds := ms / 1000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %02ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
