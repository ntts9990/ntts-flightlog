package cli

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newTurnCloseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "turn-close <turn-id-or-number> [요약]",
		Short: "지정한 턴을 종료하거나 outcome을 보강합니다",
		Long:  "현재 포인터가 아닌 오래 열린 active 턴을 ID prefix 또는 턴 번호로 종료하고, 이미 완료된 턴은 빈 outcome을 보강합니다.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary := "작업 턴 완료."
			if len(args) > 1 {
				summary = args[1]
			}

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			target, err := findClosableTurn(s, args[0])
			if err != nil {
				return err
			}
			if target.status != db.TurnStatusActive && !(target.status == db.TurnStatusComplete && target.outcome == "") {
				return fmt.Errorf("turn-close: turn %d is %s, not active", target.sequenceNo, target.status)
			}

			endedAt := time.Now().UTC()
			elapsed := secondsBetweenTimes(target.startedAt, endedAt)
			kindKey := fmt.Sprintf("turn-%d-close", target.sequenceNo)
			detail := fmt.Sprintf("소요 시간: %s.", worklog.FormatDuration(elapsed))
			if target.status == db.TurnStatusActive {
				if _, err := s.store.Exec(
					`UPDATE turns SET ended_at = ?, elapsed_ms = ?, status = 'complete', outcome = ? WHERE id = ?`,
					endedAt.Format("2006-01-02T15:04:05Z"), elapsed*1000, nullStr(summary), target.id,
				); err != nil {
					return fmt.Errorf("turn-close: update turn: %w", err)
				}
			} else {
				kindKey = fmt.Sprintf("turn-%d-outcome", target.sequenceNo)
				detail = "완료된 턴의 outcome을 보강했습니다."
				if _, err := s.store.Exec(`UPDATE turns SET outcome = ? WHERE id = ?`, nullStr(summary), target.id); err != nil {
					return fmt.Errorf("turn-close: update turn outcome: %w", err)
				}
			}

			if s.activeTurnID() == target.id || s.activeTurnNumber() == target.sequenceNo {
				s.clearActiveTurn()
			}

			block := fmt.Sprintf("\n### %s [%s] %s\n%s\n", worklog.Timestamp(), kindKey, summary, detail)
			if err := worklog.AppendToTurnFile(s.cfg, target.sequenceNo, block); err != nil {
				return fmt.Errorf("turn-close: append turn file: %w", err)
			}
			if err := worklog.AppendEntryToMain(s.cfg, kindKey, summary, detail); err != nil {
				return err
			}

			return worklog.ReplaceStatus(s.cfg, "대기",
				fmt.Sprintf("turn-%d 완료: %s.", target.sequenceNo, worklog.FormatDuration(elapsed)),
				"다음 작업 턴을 기다립니다.")
		},
	}
	return cmd
}

type closableTurn struct {
	id         string
	sequenceNo int
	status     string
	startedAt  time.Time
	outcome    string
}

func findClosableTurn(s *session, query string) (closableTurn, error) {
	if n, err := strconv.Atoi(query); err == nil {
		return queryClosableTurn(s, "sequence_no = ?", n)
	}
	return queryClosableTurn(s, "id = ? OR id LIKE ?", query, query+"%")
}

func queryClosableTurn(s *session, where string, args ...any) (closableTurn, error) {
	rows, err := s.store.Query(`SELECT id, sequence_no, status, started_at, COALESCE(outcome, '') FROM turns WHERE `+where+` ORDER BY started_at`, args...)
	if err != nil {
		return closableTurn{}, fmt.Errorf("turn-close: find turn: %w", err)
	}
	defer rows.Close()

	var matches []closableTurn
	for rows.Next() {
		var t closableTurn
		var startedRaw string
		if err := rows.Scan(&t.id, &t.sequenceNo, &t.status, &startedRaw, &t.outcome); err != nil {
			return closableTurn{}, fmt.Errorf("turn-close: scan turn: %w", err)
		}
		startedAt, err := time.Parse("2006-01-02T15:04:05Z", startedRaw)
		if err != nil {
			return closableTurn{}, fmt.Errorf("turn-close: parse turn %d start time: %w", t.sequenceNo, err)
		}
		t.startedAt = startedAt
		matches = append(matches, t)
	}
	if err := rows.Err(); err != nil {
		return closableTurn{}, fmt.Errorf("turn-close: find turn: %w", err)
	}
	if len(matches) == 0 {
		return closableTurn{}, sql.ErrNoRows
	}
	if len(matches) > 1 {
		var labels []string
		for _, t := range matches {
			labels = append(labels, fmt.Sprintf("turn-%d %s", t.sequenceNo, t.id[:8]))
		}
		return closableTurn{}, fmt.Errorf("turn-close: turn identifier is ambiguous: %s", strings.Join(labels, ", "))
	}
	return matches[0], nil
}

func secondsBetweenTimes(start, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Seconds())
}
