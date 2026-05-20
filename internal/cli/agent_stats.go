package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
	"github.com/spf13/cobra"
)

func newAgentStatsCmd() *cobra.Command {
	var format string
	var window string
	var statsAgent string

	cmd := &cobra.Command{
		Use:   "agent-stats",
		Short: "에이전트별 작업 통계와 자동감지 상태를 출력합니다",
		Long:  "Phase E 검증용으로 agent별 완료율, blocker 빈도, unknown/override 비율을 출력합니다.",
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

			snap, err := metrics.QueryAgentStats(sess.store, metrics.Filter{Window: window, Agent: statsAgent})
			if err != nil {
				return fmt.Errorf("agent-stats: %w", err)
			}

			switch format {
			case "json":
				data, err := metrics.FormatAgentStatsJSON(snap, time.Time{})
				if err != nil {
					return err
				}
				cmd.Println(string(data))
			default:
				cmd.Print(formatAgentStatsText(snap, statsAgent))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	cmd.Flags().StringVar(&window, "window", "all", "기간 필터: day|week|all")
	cmd.Flags().StringVar(&statsAgent, "agent", "", "에이전트 필터 (예: claude, codex, gemini)")
	return cmd
}

func formatAgentStatsText(snap *metrics.AgentStatsSnapshot, agent string) string {
	var b strings.Builder
	filter := "전체"
	if agent != "" {
		filter = agent
	}
	fmt.Fprintf(&b, "NTTS Flightlog agent-stats [window=%s] [agent=%s]\n", snap.Window, filter)
	fmt.Fprintf(&b, "auto_detect_correct_rate: %.1f%% (%d/%d sessions)\n",
		snap.Summary.CorrectRate*100, snap.Summary.CorrectSessions, snap.Summary.TotalSessions)
	fmt.Fprintf(&b, "auto_detect_unknown_rate: %.1f%% (%d/%d sessions)\n",
		snap.Summary.UnknownRate*100, snap.Summary.UnknownSessions, snap.Summary.TotalSessions)
	fmt.Fprintf(&b, "auto_detect_mismatch_rate: %.1f%% (%d/%d sessions)\n",
		snap.Summary.MismatchRate*100, snap.Summary.MismatchSessions, snap.Summary.TotalSessions)
	fmt.Fprintf(&b, "override_rate: %.1f%% (%d/%d sessions)\n\n",
		snap.Summary.OverrideRate*100, snap.Summary.OverrideSessions, snap.Summary.TotalSessions)
	if len(snap.Agents) == 0 {
		b.WriteString("(agent data 없음)\n")
		return b.String()
	}
	for _, stat := range snap.Agents {
		fmt.Fprintf(&b, "%-10s sessions=%d turns=%d complete=%d completion=%.1f%% blockers=%d blocker_freq=%.3f\n",
			stat.AgentID, stat.Sessions, stat.Turns, stat.CompleteTurns,
			stat.CompletionRate*100, stat.Blockers, stat.BlockerFreq)
	}
	return b.String()
}
