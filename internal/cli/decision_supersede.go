package cli

import (
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newDecisionSupersedeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decision-supersede <old-id-or-title> <new-title> [reason]",
		Short: "기존 결정을 새 결정으로 대체합니다",
		Long:  "기존 결정을 id 또는 고유한 제목 일부로 찾아 superseded 처리하고 새 accepted 결정을 기록합니다.",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldQuery := args[0]
			newTitle := args[1]
			reason := ""
			if len(args) >= 3 {
				reason = args[2]
			}

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			oldID, err := findDecisionEntryID(s, oldQuery)
			if err != nil {
				return err
			}

			detail := reason
			if detail == "" {
				detail = fmt.Sprintf("이전 결정 %s 대체.", shortCLIID(oldID))
			}
			newID, err := insertEntry(s, db.KindDecision, newTitle, detail)
			if err != nil {
				return err
			}
			if err := insertDecisionStatus(s, newID, db.DecisionStatusAccepted, "", "", ""); err != nil {
				return err
			}
			if err := insertDecisionStatus(s, oldID, db.DecisionStatusSuperseded, newID, now(), reason); err != nil {
				return err
			}

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}
			if err := worklog.AppendEntryForLane(s.cfg, s.lane, db.KindDecision, newTitle, detail); err != nil {
				return err
			}
			maybeReminderAnchor(s)
			return nil
		},
	}
	return cmd
}

func shortCLIID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
