package views

import (
	"fmt"
	"strings"
)

// RenderBlockers returns all blocker entries as an ANSI-colored string.
// Mirrors v1 awk filter_entries_by_kind("blocker").
func RenderBlockers(data *WorklogData) string {
	var sb strings.Builder
	found := false

	if data == nil {
		return "(블로커가 없습니다.)\n"
	}

	for _, e := range data.Entries {
		if e.Kind != "blocker" {
			continue
		}
		found = true
		fmt.Fprintf(&sb, "%s%s!! %s%s\n", Bold, ColorBlocker, e.CreatedAt, Reset)
		fmt.Fprintf(&sb, "%s%s  %s%s\n", Bold, ColorBlocker, e.Title, Reset)
		if e.Detail.Valid && e.Detail.String != "" {
			WriteDetail(&sb, e.Detail.String)
		}
	}

	if !found {
		return "(블로커가 없습니다.)\n"
	}
	return sb.String()
}
