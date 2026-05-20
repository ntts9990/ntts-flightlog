package cli

import (
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <상태> [초점] [다음]",
		Short: "현재 세션 상태를 업데이트합니다",
		Long:  "현재 상태 레이블과 선택적으로 초점/다음 단계를 설정합니다.",
		Args:  cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			label := args[0]
			focus := ""
			nextStep := ""
			if len(args) >= 2 {
				focus = args[1]
			}
			if len(args) >= 3 {
				nextStep = args[2]
			}

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			// Update SQLite session focus/next_step if session is active.
			sessionID := s.cfg.ActiveSessionID()
			if sessionID != "" {
				if _, err := s.store.Exec(
					`UPDATE sessions SET focus = ?, next_step = ? WHERE id = ?`,
					nullStr(focus), nullStr(nextStep), sessionID,
				); err != nil {
					return err
				}
			}

			return worklog.ReplaceStatus(s.cfg, label, focus, nextStep)
		},
	}
	return cmd
}
