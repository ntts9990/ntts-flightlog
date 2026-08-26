package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newTurnPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "turn-path [N]",
		Short: "턴 N(또는 현재 턴)의 절대 경로를 출력합니다",
		Long:  "지정한 턴 번호(또는 현재 턴)의 마크다운 파일 절대 경로를 출력합니다.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := worklog.DefaultConfig()
			var n int
			if len(args) > 0 {
				var err error
				n, err = strconv.Atoi(args[0])
				if err != nil || n < 1 {
					return fmt.Errorf("turn-path: invalid turn number %q", args[0])
				}
			} else {
				n = cfg.CurrentTurnNumber()
			}
			if n == 0 {
				return fmt.Errorf("활성 turn이 없습니다. turn-start로 시작하세요")
			}
			abs, err := filepath.Abs(cfg.TurnFilePath(n))
			if err != nil {
				return err
			}
			fmt.Println(abs)
			return nil
		},
	}
	return cmd
}
