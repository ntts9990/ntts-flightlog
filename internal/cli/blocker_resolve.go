package cli

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

type blockerMatch struct {
	id       string
	entryID  sql.NullString
	title    string
	openedAt string
}

func newBlockerResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocker-resolve <id-or-title> [해결내용]",
		Short: "열린 블로커를 해결 처리합니다",
		Long:  "열린 블로커를 id, entry id, 또는 고유한 제목 일부로 찾아 resolved 상태로 전환합니다.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			note := ""
			if len(args) >= 2 {
				note = args[1]
			}

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			match, err := findOpenBlocker(s, query)
			if err != nil {
				return err
			}

			closedAt := now()
			accumulatedSeconds := blockerElapsedSeconds(match.openedAt, closedAt)
			const updateQ = `UPDATE blockers
				SET status = 'resolved', closed_at = ?, accumulated_seconds = ?, resolution_note = ?
				WHERE id = ? AND status = 'open'`
			result, err := s.store.Exec(updateQ, closedAt, accumulatedSeconds, nullStr(note), match.id)
			if err != nil {
				return fmt.Errorf("resolve blocker: %w", err)
			}
			if rows, _ := result.RowsAffected(); rows == 0 {
				return fmt.Errorf("resolve blocker: blocker is not open: %s", query)
			}

			detail := fmt.Sprintf("해결된 블로커: %s", match.id)
			if note != "" {
				detail += "\n" + note
			}
			title := "블로커 해결: " + match.title
			if _, err := insertEntry(s, db.KindEvidence, title, detail); err != nil {
				return err
			}
			return worklog.AppendEntryForLane(s.cfg, s.lane, db.KindEvidence, title, detail)
		},
	}
	return cmd
}

func findOpenBlocker(s *session, query string) (blockerMatch, error) {
	const q = `SELECT id, entry_id, title, opened_at
		FROM blockers
		WHERE status = 'open'
		  AND (id = ? OR entry_id = ? OR title = ? OR lower(title) LIKE '%' || lower(?) || '%')
		ORDER BY opened_at`
	rows, err := s.store.Query(q, query, query, query, query)
	if err != nil {
		return blockerMatch{}, fmt.Errorf("find blocker: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var matches []blockerMatch
	for rows.Next() {
		var m blockerMatch
		if err := rows.Scan(&m.id, &m.entryID, &m.title, &m.openedAt); err != nil {
			return blockerMatch{}, fmt.Errorf("find blocker: scan: %w", err)
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return blockerMatch{}, fmt.Errorf("find blocker: %w", err)
	}
	if len(matches) == 0 {
		return blockerMatch{}, fmt.Errorf("열린 블로커를 찾을 수 없습니다: %s", query)
	}
	if len(matches) > 1 {
		var titles []string
		for _, m := range matches {
			titles = append(titles, m.title)
		}
		return blockerMatch{}, fmt.Errorf("블로커가 여러 개 일치합니다: %s", strings.Join(titles, ", "))
	}
	return matches[0], nil
}

func blockerElapsedSeconds(openedAt, closedAt string) int64 {
	opened, err := time.Parse("2006-01-02T15:04:05Z", openedAt)
	if err != nil {
		return 0
	}
	closed, err := time.Parse("2006-01-02T15:04:05Z", closedAt)
	if err != nil {
		return 0
	}
	elapsed := closed.Sub(opened).Seconds()
	if elapsed < 0 {
		return 0
	}
	return int64(elapsed)
}
