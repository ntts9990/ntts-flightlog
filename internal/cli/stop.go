package cli

import (
	"os"
	"os/exec"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "현재 세션을 종료합니다",
		Long:  "현재 진행 중인 세션을 종료하고 경과 시간을 기록합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			sessionID := s.cfg.ActiveSessionID()
			if sessionID != "" {
				if _, err := s.store.Exec(
					`UPDATE sessions SET ended_at = ? WHERE id = ?`, now(), sessionID,
				); err != nil {
					return err
				}
			}

			// Kill tmux pane if alive.
			if paneAlive(s.cfg) {
				paneID := worklog.ReadFile(s.cfg.PaneFile)
				_ = exec.Command("tmux", "kill-pane", "-t", paneID).Run()
			}
			_ = os.Remove(s.cfg.PaneFile)
			_ = os.Remove(s.cfg.SessionIDFile)

			if err := worklog.ReplaceStatus(s.cfg, "중지됨",
				"작업 기록 pane을 닫았습니다.",
				"필요하면 auto로 다시 시작하세요."); err != nil {
				return err
			}
			cmd.Println("작업 기록 pane 중지됨.")
			return nil
		},
	}
	return cmd
}
