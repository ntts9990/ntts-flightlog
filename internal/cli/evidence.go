package cli

import (
	"fmt"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newEvidenceCmd() *cobra.Command {
	var linkDecision string

	cmd := &cobra.Command{
		Use:   "evidence <제목> [상세내용]",
		Short: "근거/증거를 기록합니다",
		Long:  "결정의 근거가 되는 증거 항목을 현재 턴에 추가합니다.",
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

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			evidenceID, err := insertEntry(s, db.KindEvidence, title, detail)
			if err != nil {
				return err
			}

			// Optionally link to a decision entry by ID.
			if linkDecision != "" {
				const q = `INSERT OR IGNORE INTO decision_evidence_links
					(decision_entry_id, evidence_entry_id, created_at)
					VALUES (?, ?, ?)`
				if _, err := s.store.Exec(q, linkDecision, evidenceID, now()); err != nil {
					return fmt.Errorf("link evidence to decision: %w", err)
				}
			}

			return worklog.AppendEntry(s.cfg, db.KindEvidence, title, detail)
		},
	}
	cmd.Flags().StringVar(&linkDecision, "link", "", "연결할 결정 항목 ID (decision entry ID)")
	return cmd
}
