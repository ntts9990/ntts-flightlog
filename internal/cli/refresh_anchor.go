package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// anchorRow holds the TIA fields fetched from a turns row.
type anchorRow struct {
	id              string
	intent          sql.NullString
	constraintsJSON sql.NullString
	doneWhen        sql.NullString
}

func newRefreshAnchorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh-anchor [TURN_ID]",
		Short: "현재 턴의 Intent Anchor를 stdout에 출력하고 갱신 시각을 기록합니다",
		Long: `현재 활성 턴(또는 지정한 TURN_ID)의 Turn Intent Anchor를 출력합니다.
에이전트가 컨텍스트를 잃었을 때 의도·제약·완료조건을 복원하는 데 사용합니다.
anchor_last_shown_at 컬럼이 현재 시각으로 갱신됩니다.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			// Resolve turn ID: explicit arg or active turn.
			turnID := ""
			if len(args) > 0 {
				turnID = args[0]
			} else {
				turnID = s.activeTurnID()
			}
			if turnID == "" {
				return fmt.Errorf("활성 턴이 없습니다. turn-start로 시작하세요")
			}

			// Fetch anchor fields.
			const q = `SELECT id, intent, constraints_json, done_when FROM turns WHERE id = ?`
			var row anchorRow
			if err := s.store.QueryRow(q, turnID).Scan(
				&row.id, &row.intent, &row.constraintsJSON, &row.doneWhen,
			); err != nil {
				return fmt.Errorf("refresh-anchor: fetch turn: %w", err)
			}

			// Render anchor block.
			block := renderAnchorBlock(row)
			if block == "" {
				cmd.Printf("턴 %s에 Intent Anchor가 설정되어 있지 않습니다.\n", turnID[:8])
				cmd.Println("turn-start --intent <의도> 로 설정하세요.")
				return nil
			}
			fmt.Print(block)

			// Update anchor_last_shown_at.
			if _, err := s.store.Exec(
				`UPDATE turns SET anchor_last_shown_at = ? WHERE id = ?`, now(), turnID,
			); err != nil {
				return fmt.Errorf("refresh-anchor: update anchor_last_shown_at: %w", err)
			}
			return nil
		},
	}
	return cmd
}

// renderAnchorBlock formats the Korean anchor block from a DB row.
func renderAnchorBlock(row anchorRow) string {
	if !row.intent.Valid && !row.constraintsJSON.Valid && !row.doneWhen.Valid {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("─── ⚓ Turn Intent Anchor ───────────────────\n")
	if row.intent.Valid && row.intent.String != "" {
		fmt.Fprintf(&sb, "⚓ 의도: %s\n", row.intent.String)
	}
	if row.constraintsJSON.Valid && row.constraintsJSON.String != "" {
		var constraints []string
		if err := json.Unmarshal([]byte(row.constraintsJSON.String), &constraints); err == nil && len(constraints) > 0 {
			fmt.Fprintf(&sb, "📐 제약: %s\n", strings.Join(constraints, " | "))
		}
	}
	if row.doneWhen.Valid && row.doneWhen.String != "" {
		fmt.Fprintf(&sb, "✅ 완료조건: %s\n", row.doneWhen.String)
	}
	sb.WriteString("────────────────────────────────────────────\n")
	return sb.String()
}
