package cli

import (
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newModeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mode <모드명> [상세내용]",
		Short: "현재 작업 모드를 설정합니다",
		Long:  "작업 모드를 변경하고 main.md에 기록합니다 (예: 코딩, 검토, 회의).",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := args[0]
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

			// Persist mode file (v1 compat).
			if err := worklog.WriteFile(s.cfg.ModeFile, mode); err != nil {
				return err
			}

			// Update SQLite session mode.
			sessionID := s.cfg.ActiveSessionID()
			if sessionID != "" {
				if _, err := s.store.Exec(
					`UPDATE sessions SET mode = ? WHERE id = ?`, mode, sessionID,
				); err != nil {
					return err
				}
			}

			// Append mode entry to main.md + turn file.
			entryTitle := "작업 모드: " + mode
			if err := writeEntry(s, "mode", entryTitle, detail); err != nil {
				return err
			}

			return worklog.ReplaceStatus(s.cfg, "모드 설정",
				"현재 작업 모드: "+mode,
				"작업 진행 내용을 턴 단위로 기록하세요.")
		},
	}
	return cmd
}
