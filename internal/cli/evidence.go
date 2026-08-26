package cli

import (
	"fmt"
	"strings"

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

			// Optionally link to a decision entry by ID or unique title fragment.
			if linkDecision != "" {
				decisionID, err := findDecisionEntryID(s, linkDecision)
				if err != nil {
					return err
				}
				const q = `INSERT OR IGNORE INTO decision_evidence_links
					(decision_entry_id, evidence_entry_id, created_at)
					VALUES (?, ?, ?)`
				if _, err := s.store.Exec(q, decisionID, evidenceID, now()); err != nil {
					return fmt.Errorf("link evidence to decision: %w", err)
				}
			}

			return worklog.AppendEntryForLane(s.cfg, s.lane, db.KindEvidence, title, detail)
		},
	}
	cmd.Flags().StringVar(&linkDecision, "link", "", "연결할 결정 항목 ID 또는 고유한 제목 일부")
	return cmd
}

func findDecisionEntryID(s *session, query string) (string, error) {
	const q = `SELECT id, title
		FROM entries
		WHERE kind = 'decision'
		  AND (id = ? OR title = ? OR lower(title) LIKE '%' || lower(?) || '%')
		ORDER BY created_at`
	rows, err := s.store.Query(q, query, query, query)
	if err != nil {
		return "", fmt.Errorf("find decision: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type match struct {
		id    string
		title string
	}
	var matches []match
	for rows.Next() {
		var m match
		if err := rows.Scan(&m.id, &m.title); err != nil {
			return "", fmt.Errorf("find decision: scan: %w", err)
		}
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("find decision: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("결정 항목을 찾을 수 없습니다: %s", query)
	}
	if len(matches) > 1 {
		var titles []string
		for _, m := range matches {
			titles = append(titles, m.title)
		}
		return "", fmt.Errorf("결정 항목이 여러 개 일치합니다: %s", strings.Join(titles, ", "))
	}
	return matches[0].id, nil
}
