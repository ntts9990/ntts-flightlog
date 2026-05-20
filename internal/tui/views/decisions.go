package views

import (
	"fmt"
	"strings"
)

// RenderDecisions returns all decision entries as an ANSI-colored string.
// Mirrors v1 awk filter_entries_by_kind("decision").
func RenderDecisions(data *WorklogData) string {
	var sb strings.Builder
	found := false

	if data == nil {
		return "(결정 사항이 아직 없습니다. flightlog decision으로 기록하세요.)\n"
	}

	for _, e := range data.Entries {
		if e.Kind != "decision" {
			continue
		}
		found = true
		fmt.Fprintf(&sb, "%s%s◆ %s%s\n", Bold, ColorDecision, e.CreatedAt, Reset)
		fmt.Fprintf(&sb, "%s%s  %s%s\n", Bold, ColorDecision, e.Title, Reset)
		if e.Detail.Valid && e.Detail.String != "" {
			WriteDetail(&sb, e.Detail.String)
		}
	}

	if !found {
		return "(결정 사항이 아직 없습니다. flightlog decision으로 기록하세요.)\n"
	}
	return sb.String()
}
