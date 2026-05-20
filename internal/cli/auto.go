package cli

import (
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newAutoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto [제목]",
		Short: "세션을 자동으로 시작하거나 재시작합니다",
		Long:  "세션이 없으면 start, 이미 있으면 재연결합니다 (tmux 재접속 시 사용).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := "작업 기록"
			if len(args) > 0 {
				title = args[0]
			}
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, title); err != nil {
				return err
			}

			// If there's no active session, create one (like start).
			sessionID := s.cfg.ActiveSessionID()
			if sessionID == "" {
				sessionID, err = insertSession(s, title, "solo")
				if err != nil {
					return err
				}
				if err := worklog.WriteFile(s.cfg.SessionIDFile, sessionID); err != nil {
					return err
				}
				if err := worklog.WriteFile(s.cfg.SessionStart, worklog.EpochSeconds()); err != nil {
					return err
				}
			}

			if err := worklog.ReplaceStatus(s.cfg, "활성",
				"작업 기록 pane이 표시됩니다.",
				"turn-start로 시간 측정 작업 턴을 시작하세요."); err != nil {
				return err
			}

			return startPane(s.cfg, title, cmd)
		},
	}
	return cmd
}
