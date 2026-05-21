package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ntts9990/ntts-flightlog/internal/agent"
	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

type ingestResponse struct {
	OK                bool   `json:"ok"`
	EventID           string `json:"event_id,omitempty"`
	Duplicate         bool   `json:"duplicate"`
	PromotionStatus   string `json:"promotion_status"`
	PromotedEntryID   string `json:"promoted_entry_id,omitempty"`
	RedactionVersion  string `json:"redaction_version"`
	DroppedFieldCount int    `json:"dropped_field_count"`
}

func newIngestCmd() *cobra.Command {
	var source string
	var eventName string

	cmd := &cobra.Command{
		Use:   "ingest --source <agent> --event <name>",
		Short: "redacted hook/event JSON을 기록합니다",
		Long:  "agent hook/event JSON을 stdin에서 읽고 raw payload 없이 redacted audit record와 후보 evidence/blocker를 기록합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read event stdin: %w", err)
			}
			input, err := agent.ParseEventJSON(raw, source, eventName)
			if err != nil {
				return fmt.Errorf("ingest: %w", err)
			}
			event := agent.SanitizeEvent(input, now())

			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			response, err := ingestSanitizedEvent(s, event)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(response)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "event source override (codex, claude, gemini, generic)")
	cmd.Flags().StringVar(&eventName, "event", "", "event name override")
	return cmd
}

func ingestSanitizedEvent(s *session, event agent.SanitizedEvent) (ingestResponse, error) {
	sessionID, err := ensureActiveSession(s, "작업 기록", "solo")
	if err != nil {
		return ingestResponse{}, err
	}
	status, kind := classifyIngestPromotion(event)
	const q = `INSERT OR IGNORE INTO agent_events
		(session_id, turn_id, source, event_name, event_time, summary, severity, dedupe_key,
		 promotion_status, redaction_version, dropped_field_count, command_summary, exit_code, duration_ms, lane)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := s.store.Exec(q,
		sessionID,
		nullStr(s.activeTurnID()),
		event.Source,
		event.EventName,
		event.EventTime,
		event.Summary,
		nullStr(event.Severity),
		event.DedupeKey,
		status,
		event.RedactionVersion,
		event.DroppedFieldCount,
		nullStr(event.CommandSummary),
		event.ExitCode,
		event.DurationMS,
		nullStr(s.lane),
	)
	if err != nil {
		return ingestResponse{}, fmt.Errorf("insert agent event: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ingestResponse{}, fmt.Errorf("insert agent event affected rows: %w", err)
	}
	if affected == 0 {
		eventID, existingStatus, promotedEntryID, err := findAgentEventByDedupeKey(s, event.DedupeKey)
		if err != nil {
			return ingestResponse{}, err
		}
		return ingestResponse{
			OK:                true,
			EventID:           eventID,
			Duplicate:         true,
			PromotionStatus:   duplicatePromotionStatus(existingStatus),
			PromotedEntryID:   promotedEntryID,
			RedactionVersion:  event.RedactionVersion,
			DroppedFieldCount: event.DroppedFieldCount,
		}, nil
	}

	eventID, _, _, err := findAgentEventByDedupeKey(s, event.DedupeKey)
	if err != nil {
		return ingestResponse{}, err
	}
	promotedEntryID := ""
	if status == "candidate" {
		entryID, err := promoteIngestCandidate(s, kind, event)
		if err != nil {
			return ingestResponse{}, err
		}
		promotedEntryID = entryID
		if _, err := s.store.Exec(`UPDATE agent_events SET promotion_status = 'promoted', promoted_entry_id = ? WHERE id = ?`, entryID, eventID); err != nil {
			return ingestResponse{}, fmt.Errorf("update agent event promotion: %w", err)
		}
		status = "promoted"
	}
	return ingestResponse{
		OK:                true,
		EventID:           eventID,
		Duplicate:         false,
		PromotionStatus:   status,
		PromotedEntryID:   promotedEntryID,
		RedactionVersion:  event.RedactionVersion,
		DroppedFieldCount: event.DroppedFieldCount,
	}, nil
}

func findAgentEventByDedupeKey(s *session, dedupeKey string) (eventID, status, promotedEntryID string, err error) {
	var promoted *string
	const q = `SELECT id, promotion_status, promoted_entry_id FROM agent_events WHERE dedupe_key = ?`
	if err := s.store.QueryRow(q, dedupeKey).Scan(&eventID, &status, &promoted); err != nil {
		return "", "", "", fmt.Errorf("find agent event: %w", err)
	}
	if promoted != nil {
		promotedEntryID = *promoted
	}
	return eventID, status, promotedEntryID, nil
}

func classifyIngestPromotion(event agent.SanitizedEvent) (status, kind string) {
	text := strings.ToLower(event.EventName + " " + event.Summary)
	isTest := strings.Contains(text, "test")
	failed := strings.Contains(text, "fail") || strings.Contains(text, "failed") || strings.Contains(text, "permission denied") || strings.Contains(text, "denied")
	passed := strings.Contains(text, "pass") || strings.Contains(text, "passed")
	if event.ExitCode != nil {
		if *event.ExitCode == 0 && isTest {
			passed = true
		}
		if *event.ExitCode != 0 && isTest {
			failed = true
		}
	}
	switch {
	case failed:
		return "candidate", db.KindBlocker
	case isTest && passed:
		return "candidate", db.KindEvidence
	default:
		return "none", ""
	}
}

func promoteIngestCandidate(s *session, kind string, event agent.SanitizedEvent) (string, error) {
	titlePrefix := "Hook evidence candidate"
	if kind == db.KindBlocker {
		titlePrefix = "Hook blocker candidate"
	}
	title := fmt.Sprintf("%s: %s", titlePrefix, event.EventName)
	detail := ingestCandidateDetail(event)
	if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
		return "", err
	}
	entryID, err := insertEntry(s, kind, title, detail)
	if err != nil {
		return "", err
	}
	if kind == db.KindBlocker {
		const bq = `INSERT INTO blockers (turn_id, entry_id, title, opened_at, status)
			VALUES (?, ?, ?, ?, 'open')`
		if _, err := s.store.Exec(bq, nullStr(s.activeTurnID()), entryID, title, now()); err != nil {
			return "", fmt.Errorf("insert blocker candidate: %w", err)
		}
	}
	if err := worklog.AppendEntryForLane(s.cfg, s.lane, kind, title, detail); err != nil {
		return "", err
	}
	return entryID, nil
}

func ingestCandidateDetail(event agent.SanitizedEvent) string {
	var parts []string
	if event.Summary != "" {
		parts = append(parts, event.Summary)
	}
	if event.CommandSummary != "" {
		parts = append(parts, "command: "+event.CommandSummary)
	}
	if event.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("exit_code: %d", *event.ExitCode))
	}
	if event.DurationMS != nil {
		parts = append(parts, fmt.Sprintf("duration_ms: %d", *event.DurationMS))
	}
	parts = append(parts, "redaction: "+event.RedactionVersion)
	return strings.Join(parts, "\n")
}

func duplicatePromotionStatus(existing string) string {
	if existing == "promoted" || existing == "candidate" || existing == "none" || existing == "rejected" {
		return "duplicate"
	}
	return existing
}
