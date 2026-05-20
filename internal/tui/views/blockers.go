package views

import (
	"fmt"
	"strings"
)

// RenderBlockers returns an open-risk board. It prioritizes blocker state rows
// and falls back to blocker entries for older/imported data.
func RenderBlockers(data *WorklogData) string {
	var sb strings.Builder

	if data == nil {
		return "(블로커가 없습니다.)\n"
	}

	blockerEntries := make([]Entry, 0)
	entryByID := make(map[string]Entry)
	for _, e := range data.Entries {
		entryByID[e.ID] = e
		if e.Kind == "blocker" {
			blockerEntries = append(blockerEntries, e)
		}
	}
	turnByID := make(map[string]Turn)
	for _, t := range data.Turns {
		turnByID[t.ID] = t
	}

	if len(data.Blockers) == 0 && len(blockerEntries) == 0 {
		return "(블로커가 없습니다.)\n"
	}

	sb.WriteString(Bold + ColorSection + "## 블로커 / 리스크" + Reset + "\n")
	sb.WriteString(Dim + "현재 진행을 막는 항목을 open 우선으로 보여줍니다." + Reset + "\n\n")

	openCount := renderBlockerGroup(&sb, "열림", "open", data.Blockers, entryByID, turnByID)
	resolvedCount := renderBlockerGroup(&sb, "해결됨", "resolved", data.Blockers, entryByID, turnByID)

	if len(data.Blockers) == 0 {
		sb.WriteString(Bold + ColorBlocker + "열림" + Reset + "\n")
		for _, e := range blockerEntries {
			writeLegacyBlocker(&sb, e, turnByID)
			openCount++
		}
	}

	if openCount == 0 && resolvedCount == 0 {
		return "(블로커가 없습니다.)\n"
	}
	return sb.String()
}

func renderBlockerGroup(sb *strings.Builder, label, status string, blockers []Blocker, entryByID map[string]Entry, turnByID map[string]Turn) int {
	count := 0
	for _, b := range blockers {
		if b.Status != status {
			continue
		}
		if count == 0 {
			sb.WriteString(Bold + ColorBlocker + label + Reset + "\n")
		}
		writeBlockerRow(sb, b, entryByID, turnByID)
		count++
	}
	if count > 0 {
		sb.WriteString("\n")
	}
	return count
}

func writeBlockerRow(sb *strings.Builder, b Blocker, entryByID map[string]Entry, turnByID map[string]Turn) {
	fmt.Fprintf(sb, "%s%s!! %s  [%s]%s\n", Bold, ColorBlocker, b.OpenedAt, b.Status, Reset)
	fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorBlocker, b.Title, Reset)
	if b.TurnID.Valid {
		writeTurnContext(sb, b.TurnID.String, turnByID)
	}
	if b.ClosedAt.Valid && b.ClosedAt.String != "" {
		fmt.Fprintf(sb, "%s  해결: %s%s\n", Dim, b.ClosedAt.String, Reset)
	}
	if b.ResolutionNote.Valid && b.ResolutionNote.String != "" {
		fmt.Fprintf(sb, "%s  해결내용: %s%s\n", Dim, b.ResolutionNote.String, Reset)
	}
	if b.EntryID.Valid {
		if e, ok := entryByID[b.EntryID.String]; ok && e.Detail.Valid && e.Detail.String != "" {
			WriteDetail(sb, e.Detail.String)
		}
	}
}

func writeLegacyBlocker(sb *strings.Builder, e Entry, turnByID map[string]Turn) {
	fmt.Fprintf(sb, "%s%s!! %s  [open]%s\n", Bold, ColorBlocker, e.CreatedAt, Reset)
	fmt.Fprintf(sb, "%s%s  %s%s\n", Bold, ColorBlocker, e.Title, Reset)
	if e.TurnID.Valid {
		writeTurnContext(sb, e.TurnID.String, turnByID)
	}
	if e.Detail.Valid && e.Detail.String != "" {
		WriteDetail(sb, e.Detail.String)
	}
}

func writeTurnContext(sb *strings.Builder, turnID string, turnByID map[string]Turn) {
	t, ok := turnByID[turnID]
	if !ok {
		return
	}
	title := "(제목 없음)"
	if t.Title.Valid && t.Title.String != "" {
		title = t.Title.String
	}
	fmt.Fprintf(sb, "%s  turn-%d: %s%s\n", Dim, t.SequenceNo, title, Reset)
}
