package cli

import (
	"github.com/ntts9990/ntts-flightlog/internal/db"
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
			return writeEntry(s, db.KindDecision, title, detail)
		},
	}
	return cmd
}
