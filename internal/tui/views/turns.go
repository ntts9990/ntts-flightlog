package views

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RenderTurns returns the turns view: each turn rendered as its own section.
// Mirrors v1 awk per-turn markdown file rendering (RenderTurns in worklog/view.go).
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

	// Index sessions for turn context.
	sessionByID := make(map[string]Session)
	for _, s := range data.Sessions {
		sessionByID[s.ID] = s
	}

	for _, t := range data.Turns {
		title := "(제목 없음)"
		if t.Title.Valid && t.Title.String != "" {
			title = t.Title.String
		}

		// Turn file header (mirrors v1 per-turn .md: "# Turn N: title").
		fmt.Fprintf(&sb, "%s%s# Turn %d: %s%s\n", Bold, ColorTitle, t.SequenceNo, title, Reset)
		fmt.Fprintf(&sb, "%s시작: %s%s\n", Dim, t.StartedAt, Reset)
		sb.WriteString("\n")

		WriteTurnStart(&sb, t.StartedAt, t.SequenceNo, title, absTurns)
		WriteTurnAnchor(&sb, t)

		for _, e := range entriesByTurn[t.ID] {
			WriteEntry(&sb, e.CreatedAt, e.Kind, e.Title)
			if e.Detail.Valid && e.Detail.String != "" {
				WriteDetail(&sb, e.Detail.String)
			}
		}

		if t.EndedAt.Valid && t.EndedAt.String != "" {
			summary := "작업 턴 완료."
			if es := entriesByTurn[t.ID]; len(es) > 0 {
				last := es[len(es)-1]
				if last.Kind == "entry" {
					summary = last.Title
				}
			}
			WriteTurnEnd(&sb, t.EndedAt.String, t.SequenceNo, summary)
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
