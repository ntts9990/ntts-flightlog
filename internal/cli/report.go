package cli

import (
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var format string
	var window string
	var reportAgent string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "5개 메트릭 분석 리포트를 출력합니다",
		Long:  "세션 데이터를 기반으로 5개 핵심 메트릭 분석 리포트를 출력합니다. (Phase B 구현 예정)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Full implementation is Phase B (B3 metric SQL views + B4 report command).
			cmd.Printf("report: Phase B에서 구현 예정\n  format=%s window=%s agent=%s\n",
				format, window, reportAgent)
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	cmd.Flags().StringVar(&window, "window", "all", "기간 필터: day|week|all")
	cmd.Flags().StringVar(&reportAgent, "agent", "", "에이전트 필터")
	return cmd
}
