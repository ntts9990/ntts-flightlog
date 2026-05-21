package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
	"github.com/spf13/cobra"
)

func newAttentionCmd() *cobra.Command {
	var format string
	var window string
	var attentionAgent string

	cmd := &cobra.Command{
		Use:   "attention",
		Short: "주의가 필요한 작업 항목을 추천합니다",
		Long:  "블로커, 근거 없는 결정, 오래 열린 턴, drift, agent attribution 문제를 operator action으로 요약합니다.",
		Args:  cobra.NoArgs,
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

			snap, err := metrics.QueryAttention(sess.store, metrics.Filter{
				Window: window,
				Agent:  attentionAgent,
			}, metrics.AttentionOptions{})
			if err != nil {
				return fmt.Errorf("attention: %w", err)
			}

			switch format {
			case "json":
				data, err := metrics.FormatAttentionJSON(snap, time.Time{})
				if err != nil {
					return fmt.Errorf("attention json: %w", err)
				}
				cmd.Println(string(data))
			default:
				cmd.Print(formatAttentionText(snap))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	cmd.Flags().StringVar(&window, "window", "all", "기간 필터: day|week|all")
	cmd.Flags().StringVar(&attentionAgent, "agent", "", "에이전트 필터 (예: claude, codex)")
	return cmd
}

func formatAttentionText(snap *metrics.AttentionSnapshot) string {
	var b strings.Builder
	windowLabel := cliWindowLabel(snap.Window)
	agentLabel := "전체"
	if snap.Agent != "" {
		agentLabel = snap.Agent
	}

	b.WriteString("NTTS Flightlog attention\n")
	fmt.Fprintf(&b, "[창: %s] [에이전트: %s]\n", windowLabel, agentLabel)
	fmt.Fprintf(&b, "주의 필요: %d개 (높음 %d · 중간 %d · 낮음 %d)\n\n",
		snap.Summary.Total, snap.Summary.High, snap.Summary.Medium, snap.Summary.Low)

	if len(snap.Items) == 0 {
		b.WriteString("(주의 필요 없음)\n")
		return b.String()
	}

	for _, item := range snap.Items {
		fmt.Fprintf(&b, "[%s] %s\n", severityLabel(item.Severity), item.Title)
		fmt.Fprintf(&b, "  대상: %s %s\n", item.SourceType, shortCLIID(item.SourceID))
		fmt.Fprintf(&b, "  이유: %s\n", item.Reason)
		fmt.Fprintf(&b, "  다음: %s\n\n", item.RecommendedAction)
	}
	return b.String()
}

func cliWindowLabel(window string) string {
	switch window {
	case "day":
		return "오늘 (24h)"
	case "week":
		return "이번 주 (7d)"
	default:
		return "전체"
	}
}

func severityLabel(severity string) string {
	switch severity {
	case metrics.AttentionSeverityHigh:
		return "높음"
	case metrics.AttentionSeverityMedium:
		return "중간"
	default:
		return "낮음"
	}
}
