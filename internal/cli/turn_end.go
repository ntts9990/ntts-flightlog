package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newTurnEndCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "turn-end [요약]",
		Short: "현재 턴을 종료합니다",
		Long:  "현재 작업 턴을 종료하고 경과 시간을 기록합니다.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary := "작업 턴 완료."
			if len(args) > 0 {
				summary = args[0]
			}

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			n := s.activeTurnNumber()
			turnLabel := "?"
			if n > 0 {
				turnLabel = strconv.Itoa(n)
			}

			// Compute elapsed from turn-start-epoch.
			var elapsedStr string
			var elapsedMS int64
			if raw := worklog.ReadFile(s.cfg.LaneTurnStartFile(s.lane)); raw != "" {
				var startEpoch int64
				fmt.Sscanf(raw, "%d", &startEpoch)
				elapsed := time.Now().Unix() - startEpoch
				elapsedMS = elapsed * 1000
				elapsedStr = worklog.FormatDuration(elapsed)
			} else {
				elapsedStr = "unknown"
			}

			// Update turn in SQLite.
			turnID := s.activeTurnID()
			if turnID != "" {
				if _, err := s.store.Exec(
					`UPDATE turns SET ended_at = ?, elapsed_ms = ?, status = 'complete', outcome = ? WHERE id = ?`,
					now(), elapsedMS, nullStr(summary), turnID,
				); err != nil {
					return err
				}
			}

			// Clean up v1 compat + v2 state files.
			s.clearActiveTurn()

			// Append turn-end entry.
			kindKey := fmt.Sprintf("turn-%s-end", turnLabel)
			detail := fmt.Sprintf("소요 시간: %s.", elapsedStr)
			if err := worklog.AppendEntryForLane(s.cfg, s.lane, kindKey, summary, detail); err != nil {
				return err
			}

			return worklog.ReplaceStatus(s.cfg, "대기",
				fmt.Sprintf("마지막 턴 %s 완료: %s.", turnLabel, elapsedStr),
				"다음 작업 턴을 기다립니다.")
		},
	}
	return cmd
}
