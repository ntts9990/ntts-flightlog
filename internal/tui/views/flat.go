package views

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RenderFlat returns the flat worklog view as an ANSI-colored string.
// Data source: SQLite (via WorklogData) — NOT main.md file.
// Rendering: byte-identical to v1 awk render_markdown_ansi (B2 exit criterion).
func RenderFlat(data *WorklogData, turnsDir string) string {
	if data == nil || (len(data.Sessions) == 0 && len(data.Entries) == 0) {
		return "(워크로그가 비어 있습니다. flightlog start로 세션을 시작하세요.)\n"
	}

	absTurns, _ := filepath.Abs(turnsDir)
	var sb strings.Builder

	// Index: entries by turn ID, session-level entries by session ID.
	entriesByTurn := make(map[string][]Entry)
	sessionEntries := make(map[string][]Entry)
	for _, e := range data.Entries {
		if e.TurnID.Valid && e.TurnID.String != "" {
			entriesByTurn[e.TurnID.String] = append(entriesByTurn[e.TurnID.String], e)
		} else {
			sessionEntries[e.SessionID] = append(sessionEntries[e.SessionID], e)
		}
	}

	// Index: turns by session ID.
	turnsBySession := make(map[string][]Turn)
	for i := range data.Turns {
		t := data.Turns[i]
		turnsBySession[t.SessionID] = append(turnsBySession[t.SessionID], t)
	}

	for _, s := range data.Sessions {
		// Session header (mirrors v1 awk "# Title" → bold colorTitle).
		sessionTitle := "NTTS Flightlog"
		if s.Title.Valid && s.Title.String != "" {
			sessionTitle = s.Title.String
		}
		fmt.Fprintf(&sb, "%s%s# %s%s\n", Bold, ColorTitle, sessionTitle, Reset)
		fmt.Fprintf(&sb, "%s시작: %s%s\n", Dim, s.StartedAt, Reset)
		sb.WriteString("\n")

		// Session-level entries (no turn_id).
		for _, e := range sessionEntries[s.ID] {
			WriteEntry(&sb, e.CreatedAt, e.Kind, e.Title)
			if e.Detail.Valid && e.Detail.String != "" {
				WriteDetail(&sb, e.Detail.String)
			}
		}

		// Turns in sequence order (already ordered by loadTurns).
		for _, t := range turnsBySession[s.ID] {
			title := "(제목 없음)"
			if t.Title.Valid && t.Title.String != "" {
				title = t.Title.String
			}
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
				// Use the last entry's title in the turn as the summary if available.
				if es := entriesByTurn[t.ID]; len(es) > 0 {
					last := es[len(es)-1]
					if last.Kind == "entry" {
						summary = last.Title
					}
				}
				WriteTurnEnd(&sb, t.EndedAt.String, t.SequenceNo, summary)
			}
		}
	}

	return sb.String()
}
