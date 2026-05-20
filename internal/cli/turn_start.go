package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newTurnStartCmd() *cobra.Command {
	var intent string
	var constraintsRaw string
	var doneWhen string

	cmd := &cobra.Command{
		Use:   "turn-start <제목>",
		Short: "새 턴을 시작합니다",
		Long: `새 작업 턴을 시작하고 SQLite + main.md에 기록합니다.

선택적으로 Turn Intent Anchor(TIA)를 설정해 에이전트 컨텍스트 드리프트를 방지할 수 있습니다.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			// Increment turn counter (v1 compat file).
			n, err := s.cfg.NextTurnNumber()
			if err != nil {
				return err
			}

			// Record turn start epoch (v1 compat).
			if err := worklog.WriteFile(s.cfg.TurnStart, worklog.EpochSeconds()); err != nil {
				return err
			}

			// Parse --constraints into JSON array.
			var constraintsJSON string
			var constraints []string
			if constraintsRaw != "" {
				for _, c := range strings.Split(constraintsRaw, ",") {
					if t := strings.TrimSpace(c); t != "" {
						constraints = append(constraints, t)
					}
				}
				if len(constraints) > 0 {
					b, _ := json.Marshal(constraints)
					constraintsJSON = string(b)
				}
			}

			// Insert turn into SQLite (with anchor columns).
			sessionID := s.cfg.ActiveSessionID()
			turnID, err := insertTurnWithAnchor(s, sessionID, n, title, intent, constraintsJSON, doneWhen)
			if err != nil {
				return err
			}
			if err := worklog.WriteFile(s.cfg.TurnIDFile, turnID); err != nil {
				return err
			}

			// Create per-turn markdown file.
			if err := worklog.CreateTurnFile(s.cfg, n, title); err != nil {
				return err
			}

			// Build anchor block for main.md mirror (appended after turn-start entry).
			anchorBlock := buildAnchorBlock(intent, constraints, doneWhen)

			// Append turn-start entry to main.md (with anchor block as detail).
			kindKey := fmt.Sprintf("turn-%d-start", n)
			entryDetail := fmt.Sprintf("시작 시각: %s.", worklog.Timestamp())
			if anchorBlock != "" {
				entryDetail += "\n" + anchorBlock
			}
			if err := worklog.AppendEntry(s.cfg, kindKey, title, entryDetail); err != nil {
				return err
			}

			// Echo anchor to stdout so the agent sees it immediately.
			if anchorBlock != "" {
				fmt.Print(anchorBlock)
			}

			return worklog.ReplaceStatus(s.cfg,
				fmt.Sprintf("turn-%d", n),
				title,
				"결정과 검증 근거를 진행 중에 기록하세요.")
		},
	}

	cmd.Flags().StringVar(&intent, "intent", "", "턴 의도 (에이전트 컨텍스트 유지용)")
	cmd.Flags().StringVar(&constraintsRaw, "constraints", "", "쉼표 구분 제약 조건 목록")
	cmd.Flags().StringVar(&doneWhen, "done-when", "", "완료 조건 설명")
	return cmd
}

// insertTurnWithAnchor inserts a turn row with optional TIA fields and returns its ID.
func insertTurnWithAnchor(s *session, sessionID string, seqNo int, title, intent, constraintsJSON, doneWhen string) (string, error) {
	const q = `INSERT INTO turns
		(session_id, sequence_no, title, started_at, status, agent_id,
		 intent, constraints_json, done_when)
		VALUES (?, ?, ?, ?, 'active', ?, ?, ?, ?) RETURNING id`
	var id string
	err := s.store.QueryRow(q,
		nullStr(sessionID), seqNo, nullStr(title), now(), nullStr(s.agentID),
		nullStr(intent), nullStr(constraintsJSON), nullStr(doneWhen),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert turn: %w", err)
	}
	return id, nil
}

// insertTurn is kept for callers that don't need anchor fields.
func insertTurn(s *session, sessionID string, seqNo int, title string) (string, error) {
	return insertTurnWithAnchor(s, sessionID, seqNo, title, "", "", "")
}

// buildAnchorBlock returns the Korean anchor block string for main.md mirroring.
// Returns "" when no anchor fields are set.
func buildAnchorBlock(intent string, constraints []string, doneWhen string) string {
	if intent == "" && len(constraints) == 0 && doneWhen == "" {
		return ""
	}
	var sb strings.Builder
	if intent != "" {
		sb.WriteString(fmt.Sprintf("⚓ 의도: %s\n", intent))
	}
	if len(constraints) > 0 {
		sb.WriteString(fmt.Sprintf("📐 제약: %s\n", strings.Join(constraints, " | ")))
	}
	if doneWhen != "" {
		sb.WriteString(fmt.Sprintf("✅ 완료조건: %s\n", doneWhen))
	}
	return sb.String()
}
