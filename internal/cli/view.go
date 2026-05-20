package cli

import (
	"fmt"
	"os"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view [flat|turns|decisions|blockers]",
		Short: "워크로그를 ANSI 색상으로 출력합니다",
		Long:  "워크로그를 지정한 뷰 모드로 ANSI 렌더링하여 stdout에 출력합니다. (전체 TUI는 Phase B)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			view := "flat"
			if len(args) > 0 {
				view = args[0]
			}
			cfg := worklog.DefaultConfig()
			w := os.Stdout
			switch view {
			case "flat":
				return worklog.RenderFlat(cfg, w)
			case "turns":
				return worklog.RenderTurns(cfg, w)
			case "decisions":
				return worklog.FilterByKind(cfg, "decision", w)
			case "blockers":
				return worklog.FilterByKind(cfg, "blocker", w)
			default:
				return fmt.Errorf("알 수 없는 view: %s (flat|turns|decisions|blockers)", view)
			}
		},
	}
	return cmd
}
