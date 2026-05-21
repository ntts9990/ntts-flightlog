package cli

import (
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newBlockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocker <제목> [상세내용]",
		Short: "블로커를 기록합니다",
		Long:  "작업을 막는 블로커 항목을 현재 턴에 추가합니다.",
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

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			entryID, err := insertEntry(s, db.KindBlocker, title, detail)
			if err != nil {
				return err
			}

			// Also insert a blockers row (tracks resolution state + timing).
			const bq = `INSERT INTO blockers (turn_id, entry_id, title, opened_at, status)
				VALUES (?, ?, ?, ?, 'open') RETURNING id`
			var blockerID string
			if err := s.store.QueryRow(bq,
				nullStr(s.activeTurnID()), entryID, title, now(),
			).Scan(&blockerID); err != nil {
				return fmt.Errorf("insert blocker: %w", err)
			}

			return worklog.AppendEntryForLane(s.cfg, s.lane, db.KindBlocker, title, detail)
		},
	}
	return cmd
}
