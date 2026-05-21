package cli

import (
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newEntryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entry <제목> [상세내용]",
		Short: "작업 기록을 추가합니다",
		Long:  "일반 작업 기록 항목을 현재 턴에 추가합니다.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			detail := ""
			if len(args) >= 2 {
				detail = args[1]
			}
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()
			return writeEntry(s, db.KindEntry, title, detail)
		},
	}
	return cmd
}

// writeEntry inserts an entry into SQLite and mirrors it to main.md.
// This is shared by entry, decision, evidence, mode commands.
// After writing, checks if anchor reminder should be printed (A.5: ≥5 entries since last shown).
func writeEntry(s *session, kind, title, detail string) error {
	if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
		return err
	}
	entryID, err := insertEntry(s, kind, title, detail)
	if err != nil {
		return fmt.Errorf("writeEntry: %w", err)
	}
	_ = entryID // may be used by evidence --link flag in A4 extension
	if err := worklog.AppendEntryForLane(s.cfg, s.lane, kind, title, detail); err != nil {
		return err
	}
	// A.5: print anchor reminder if turn has intent AND ≥5 entries since last shown.
	maybeReminderAnchor(s)
	return nil
}

// maybeReminderAnchor prints "⚓ ANCHOR REMINDER" to stdout if the active turn
// has a non-NULL intent and ≥5 entries have been written since anchor_last_shown_at.
func maybeReminderAnchor(s *session) {
	turnID := s.activeTurnID()
	if turnID == "" {
		return
	}
	const q = `SELECT intent, anchor_last_shown_at,
		(SELECT COUNT(*) FROM entries
		 WHERE turn_id = t.id
		   AND (t.anchor_last_shown_at IS NULL OR created_at > t.anchor_last_shown_at)
		) AS entries_since
		FROM turns t WHERE t.id = ?`
	var intent, lastShown *string
	var entriesSince int
	row := s.store.QueryRow(q, turnID)
	if err := row.Scan(&intent, &lastShown, &entriesSince); err != nil {
		return // non-fatal
	}
	if intent == nil || *intent == "" {
		return
	}
	if entriesSince >= 5 {
		fmt.Printf("\n⚓ ANCHOR REMINDER: %s\n\n", *intent)
		_, _ = s.store.Exec(`UPDATE turns SET anchor_last_shown_at = ? WHERE id = ?`, now(), turnID)
	}
}

// insertEntry writes a single entry row to SQLite and returns its ID.
func insertEntry(s *session, kind, title, detail string) (string, error) {
	sessionID, err := ensureActiveSession(s, "작업 기록", "solo")
	if err != nil {
		return "", err
	}
	const q = `INSERT INTO entries (session_id, turn_id, kind, title, detail, created_at, agent_id, lane)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	var id string
	err = s.store.QueryRow(q,
		sessionID,
		nullStr(s.activeTurnID()),
		kind, title, nullStr(detail), now(), nullStr(s.agentID), nullStr(s.lane),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert entry (%s): %w", kind, err)
	}
	return id, nil
}
