package cli

import (
	"fmt"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var format string
	var window string
	var reportAgent string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "5개 메트릭 분석 리포트를 출력합니다",
		Long: `세션 데이터를 기반으로 5개 핵심 메트릭 분석 리포트를 출력합니다.

메트릭:
  1. Turn 소요시간 분포         (metric_turn_duration)
  2. Blocker 누적시간           (metric_blocker_accumulation)
  3. Agent별 Turn 완료율        (metric_agent_completion)
  4. Agent별 Blocker 빈도       (metric_agent_blocker_freq)
  5. Evidence가 붙은 Decision   (metric_evidence_bound_decisions)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("--format must be text or json, got %q", format)
			}
			if window != "day" && window != "week" && window != "all" {
				return fmt.Errorf("--window must be day, week, or all, got %q", window)
			}

			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.close()

			f := metrics.Filter{
				Window: window,
				Agent:  reportAgent,
			}

			snap, err := metrics.QueryAll(sess.store, f)
			if err != nil {
				return fmt.Errorf("report: %w", err)
			}

			switch format {
			case "json":
				data, err := metrics.FormatJSON(snap, window, reportAgent, time.Time{})
				if err != nil {
					return fmt.Errorf("report json: %w", err)
				}
				cmd.Println(string(data))
			default: // "text"
				cmd.Print(formatText(snap, window, reportAgent))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	cmd.Flags().StringVar(&window, "window", "all", "기간 필터: day|week|all")
	cmd.Flags().StringVar(&reportAgent, "agent", "", "에이전트 필터 (예: claude, codex)")
	return cmd
}
