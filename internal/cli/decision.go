package cli

import (
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newDecisionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decision <제목> [상세내용]",
		Short: "결정 사항을 기록합니다",
		Long:  "의사결정 항목을 현재 턴에 추가합니다. 증거와 연결 가능합니다.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			detail := ""
			if len(args) >= 2 {
				detail = args[1]
			}
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()
			return writeDecision(s, title, detail)
		},
	}
	return cmd
}

func writeDecision(s *session, title, detail string) error {
	if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
		return err
	}
	decisionID, err := insertEntry(s, db.KindDecision, title, detail)
	if err != nil {
		return fmt.Errorf("writeDecision: %w", err)
	}
	if err := insertDecisionStatus(s, decisionID, db.DecisionStatusAccepted, "", "", ""); err != nil {
		return err
	}
	if err := worklog.AppendEntryForLane(s.cfg, s.lane, db.KindDecision, title, detail); err != nil {
		return err
	}
	maybeReminderAnchor(s)
	return nil
}

func insertDecisionStatus(s *session, decisionID, status, supersededBy, supersededAt, rationale string) error {
	const q = `INSERT INTO decision_status
		(decision_entry_id, status, superseded_by_entry_id, superseded_at, rationale)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(decision_entry_id) DO UPDATE SET
			status = excluded.status,
			superseded_by_entry_id = excluded.superseded_by_entry_id,
			superseded_at = excluded.superseded_at,
			rationale = excluded.rationale`
	if _, err := s.store.Exec(q,
		decisionID, status, nullStr(supersededBy), nullStr(supersededAt), nullStr(rationale),
	); err != nil {
		return fmt.Errorf("insert decision status: %w", err)
	}
	return nil
}
