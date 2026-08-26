package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDriftCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift-check [TURN_ID]",
		Short: "현재 턴 항목이 제약 조건과 일치하는지 확인합니다",
		Long: `현재 활성 턴(또는 지정한 TURN_ID)의 entries를 제약 조건(constraints)과 비교합니다.
키워드 매칭(substring) 방식으로 드리프트를 감지합니다. (NLP는 v2.1+)

드리프트 감지 시:
  - blockers 테이블에 auto-drift 항목 삽입
  - turns.drift_alerts 카운터 증가`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			// Resolve turn ID.
			turnID := ""
			if len(args) > 0 {
				turnID = args[0]
			} else {
				turnID = s.activeTurnID()
			}
			if turnID == "" {
				return fmt.Errorf("활성 턴이 없습니다. turn-start로 시작하세요")
			}

			// Fetch turn anchor.
			const tq = `SELECT id, intent, constraints_json, done_when, started_at FROM turns WHERE id = ?`
			var row anchorRow
			var startedAt string
			if err := s.store.QueryRow(tq, turnID).Scan(
				&row.id, &row.intent, &row.constraintsJSON, &row.doneWhen, &startedAt,
			); err != nil {
				return fmt.Errorf("drift-check: fetch turn: %w", err)
			}

			// Parse constraints.
			if !row.constraintsJSON.Valid || row.constraintsJSON.String == "" {
				cmd.Println("이 턴에 제약 조건이 없습니다. drift-check를 건너뜁니다.")
				return nil
			}
			var constraints []string
			if err := json.Unmarshal([]byte(row.constraintsJSON.String), &constraints); err != nil {
				return fmt.Errorf("drift-check: parse constraints: %w", err)
			}
			if len(constraints) == 0 {
				cmd.Println("제약 조건 목록이 비어 있습니다.")
				return nil
			}

			// Fetch entries for this turn.
			rows, err := s.store.Query(
				`SELECT id, title, detail FROM entries WHERE turn_id = ? AND created_at >= ?`,
				turnID, startedAt,
			)
			if err != nil {
				return fmt.Errorf("drift-check: query entries: %w", err)
			}
			defer func() { _ = rows.Close() }()

			type entryCandidate struct {
				id     string
				title  string
				detail sql.NullString
			}
			var entries []entryCandidate
			for rows.Next() {
				var e entryCandidate
				if err := rows.Scan(&e.id, &e.title, &e.detail); err != nil {
					return err
				}
				entries = append(entries, e)
			}
			if err := rows.Err(); err != nil {
				return err
			}

			// Keyword overlap check: for each entry, at least one constraint keyword
			// must appear in title or detail. Entries with NO constraint keyword match
			// are flagged as drift.
			driftCount := 0
			var driftReasons []string

			for _, e := range entries {
				text := strings.ToLower(e.title)
				if e.detail.Valid {
					text += " " + strings.ToLower(e.detail.String)
				}
				matched := false
				for _, c := range constraints {
					if strings.Contains(text, strings.ToLower(c)) {
						matched = true
						break
					}
				}
				if !matched {
					reason := fmt.Sprintf("항목 \"%s\" — 제약 키워드 없음", e.title)
					driftReasons = append(driftReasons, reason)
					driftCount++

					// Insert auto-drift blocker.
					blockerTitle := fmt.Sprintf("auto-drift: %s", e.title)
					const bq = `INSERT INTO blockers (turn_id, entry_id, title, opened_at, status)
						VALUES (?, ?, ?, ?, 'open')`
					if _, err := s.store.Exec(bq, turnID, e.id, blockerTitle, now()); err != nil {
						return fmt.Errorf("drift-check: insert blocker: %w", err)
					}
				}
			}

			// Increment drift_alerts counter if any drift found.
			if driftCount > 0 {
				if _, err := s.store.Exec(
					`UPDATE turns SET drift_alerts = drift_alerts + ? WHERE id = ?`,
					driftCount, turnID,
				); err != nil {
					return fmt.Errorf("drift-check: update drift_alerts: %w", err)
				}
				cmd.Printf("⚠️  드리프트 감지: %d개 항목\n", driftCount)
				for _, r := range driftReasons {
					cmd.Printf("  • %s\n", r)
				}
				cmd.Printf("제약 조건: %s\n", strings.Join(constraints, " | "))
			} else {
				cmd.Printf("✅ 드리프트 없음 — 모든 항목이 제약 조건과 일치합니다 (%d개 검사)\n", len(entries))
			}
			return nil
		},
	}
	return cmd
}
