package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/metrics"
	tuiviews "github.com/ntts9990/ntts-flightlog/internal/tui/views"
	"github.com/spf13/cobra"
)

const shareListLimit = 12

type sharePacket struct {
	GeneratedAt      string                    `json:"generated_at"`
	Window           string                    `json:"window"`
	Summary          shareSummary              `json:"summary"`
	CompletedTurns   []shareTurn               `json:"completed_turns"`
	ActiveBlockers   []shareBlocker            `json:"active_blockers"`
	Decisions        []shareDecision           `json:"decisions"`
	MetricHighlights []metrics.MetricHighlight `json:"metric_highlights"`
	RequestedReview  []shareReviewItem         `json:"requested_review"`
}

type shareSummary struct {
	Sessions       int `json:"sessions"`
	Turns          int `json:"turns"`
	CompletedTurns int `json:"completed_turns"`
	ActiveTurns    int `json:"active_turns"`
	Entries        int `json:"entries"`
	Decisions      int `json:"decisions"`
	Evidence       int `json:"evidence"`
	ActiveBlockers int `json:"active_blockers"`
}

type shareTurn struct {
	ID        string `json:"id"`
	Sequence  int    `json:"sequence"`
	Title     string `json:"title"`
	Agent     string `json:"agent,omitempty"`
	Lane      string `json:"lane,omitempty"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	Elapsed   string `json:"elapsed,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
}

type shareBlocker struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	OpenedAt string `json:"opened_at"`
	Age      string `json:"age"`
	Detail   string `json:"detail,omitempty"`
}

type shareDecision struct {
	ID                    string `json:"id"`
	Title                 string `json:"title"`
	Status                string `json:"status"`
	LinkedEvidenceCount   int    `json:"linked_evidence_count"`
	SameTurnEvidenceCount int    `json:"same_turn_evidence_count"`
}

type shareReviewItem struct {
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Title    string `json:"title"`
	Action   string `json:"action"`
}

func newShareCmd() *cobra.Command {
	var format string
	var window string

	cmd := &cobra.Command{
		Use:   "share",
		Short: "팀 공유용 상태 리포트를 출력합니다",
		Long:  "완료 턴, 열린 블로커, 결정/근거, 메트릭 하이라이트, 리뷰 요청을 Markdown 또는 JSON으로 출력합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "md" && format != "json" {
				return fmt.Errorf("--format must be md or json, got %q", format)
			}
			if window != "day" && window != "week" && window != "all" {
				return fmt.Errorf("--window must be day, week, or all, got %q", window)
			}

			sess, err := openSession()
			if err != nil {
				return err
			}
			defer sess.close()

			data, err := tuiviews.LoadAll(sess.store)
			if err != nil {
				return fmt.Errorf("share load: %w", err)
			}
			filter := metrics.Filter{Window: window}
			snap, err := metrics.QueryAll(sess.store, filter)
			if err != nil {
				return fmt.Errorf("share metrics: %w", err)
			}
			attention, err := metrics.QueryAttention(sess.store, filter, metrics.AttentionOptions{})
			if err != nil {
				return fmt.Errorf("share attention: %w", err)
			}

			packet := buildSharePacket(data, snap, attention, window, time.Now().UTC())
			switch format {
			case "json":
				raw, err := json.MarshalIndent(packet, "", "  ")
				if err != nil {
					return fmt.Errorf("share json: %w", err)
				}
				cmd.Println(string(raw))
			default:
				cmd.Print(renderShareMarkdown(packet))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "md", "출력 형식: md|json")
	cmd.Flags().StringVar(&window, "window", "week", "기간 필터: day|week|all")
	return cmd
}

func buildSharePacket(data *tuiviews.WorklogData, snap *metrics.Snapshot, attention *metrics.AttentionSnapshot, window string, generatedAt time.Time) sharePacket {
	if data == nil {
		data = &tuiviews.WorklogData{}
	}
	data = filterShareData(data, window, generatedAt)

	linkedEvidence := make(map[string]int)
	for _, link := range data.DecisionEvidenceLinks {
		linkedEvidence[link.DecisionEntryID]++
	}
	evidenceByTurn := make(map[string]int)
	entryByID := make(map[string]tuiviews.Entry)
	for _, e := range data.Entries {
		entryByID[e.ID] = e
		if e.Kind == db.KindEvidence && e.TurnID.Valid {
			evidenceByTurn[e.TurnID.String]++
		}
	}
	stateByDecision := make(map[string]tuiviews.DecisionState)
	for _, state := range data.DecisionStates {
		stateByDecision[state.DecisionEntryID] = state
	}

	packet := sharePacket{
		GeneratedAt:      generatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Window:           window,
		Summary:          summarizeShare(data),
		MetricHighlights: metrics.BuildMetricHighlights(snap),
	}

	for _, t := range data.Turns {
		if !shareTurnCompleted(t) {
			continue
		}
		packet.CompletedTurns = append(packet.CompletedTurns, shareTurn{
			ID:        t.ID,
			Sequence:  t.SequenceNo,
			Title:     nullString(t.Title, "(제목 없음)"),
			Agent:     nullString(t.AgentID, ""),
			Lane:      nullString(t.Lane, ""),
			StartedAt: t.StartedAt,
			EndedAt:   nullString(t.EndedAt, ""),
			Elapsed:   shareElapsed(t),
			Outcome:   nullString(t.Outcome, ""),
		})
		if len(packet.CompletedTurns) >= shareListLimit {
			break
		}
	}

	for _, b := range data.Blockers {
		if b.Status != db.BlockerStatusOpen && b.Status != "" {
			continue
		}
		item := shareBlocker{
			ID:       b.ID,
			Title:    b.Title,
			OpenedAt: b.OpenedAt,
			Age:      shareBlockerAge(b, generatedAt),
		}
		if b.EntryID.Valid {
			if e, ok := entryByID[b.EntryID.String]; ok {
				item.Detail = nullString(e.Detail, "")
			}
		}
		packet.ActiveBlockers = append(packet.ActiveBlockers, item)
		if len(packet.ActiveBlockers) >= shareListLimit {
			break
		}
	}

	for _, e := range data.Entries {
		if e.Kind != db.KindDecision {
			continue
		}
		packet.Decisions = append(packet.Decisions, shareDecision{
			ID:                    e.ID,
			Title:                 e.Title,
			Status:                shareDecisionStatus(e.ID, stateByDecision),
			LinkedEvidenceCount:   linkedEvidence[e.ID],
			SameTurnEvidenceCount: shareSameTurnEvidence(e, evidenceByTurn),
		})
		if len(packet.Decisions) >= shareListLimit {
			break
		}
	}

	if attention != nil {
		for _, item := range attention.Items {
			if item.Severity == metrics.AttentionSeverityLow {
				continue
			}
			packet.RequestedReview = append(packet.RequestedReview, shareReviewItem{
				Severity: severityLabel(item.Severity),
				Source:   item.SourceType + " " + shortCLIID(item.SourceID),
				Title:    item.Title,
				Action:   item.RecommendedAction,
			})
			if len(packet.RequestedReview) >= shareListLimit {
				break
			}
		}
	}

	ensureShareSlices(&packet)
	return packet
}

func renderShareMarkdown(packet sharePacket) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# NTTS Flightlog Share\n\n")
	fmt.Fprintf(&b, "- Generated: %s\n", packet.GeneratedAt)
	fmt.Fprintf(&b, "- Window: %s\n\n", cliWindowLabel(packet.Window))

	b.WriteString("## Summary / 요약\n\n")
	fmt.Fprintf(&b, "- Sessions: %d\n", packet.Summary.Sessions)
	fmt.Fprintf(&b, "- Turns: %d total · %d completed · %d active\n", packet.Summary.Turns, packet.Summary.CompletedTurns, packet.Summary.ActiveTurns)
	fmt.Fprintf(&b, "- Entries: %d · decisions %d · evidence %d · active blockers %d\n\n",
		packet.Summary.Entries, packet.Summary.Decisions, packet.Summary.Evidence, packet.Summary.ActiveBlockers)

	b.WriteString("## Completed Turns / 완료 턴\n\n")
	if len(packet.CompletedTurns) == 0 {
		b.WriteString("- 완료 턴 없음\n\n")
	} else {
		for _, t := range packet.CompletedTurns {
			line := fmt.Sprintf("- turn-%d %s", t.Sequence, clip(t.Title, 90))
			if t.Outcome != "" {
				line += " — " + clip(t.Outcome, 100)
			}
			if t.Elapsed != "" {
				line += " (" + t.Elapsed + ")"
			}
			if t.Lane != "" {
				line += " [lane: " + t.Lane + "]"
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Active Blockers / 열린 블로커\n\n")
	if len(packet.ActiveBlockers) == 0 {
		b.WriteString("- 열린 블로커 없음\n\n")
	} else {
		for _, blocker := range packet.ActiveBlockers {
			fmt.Fprintf(&b, "- %s — %s", clip(blocker.Title, 90), blocker.Age)
			if blocker.Detail != "" {
				fmt.Fprintf(&b, ": %s", clip(blocker.Detail, 120))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Decisions And Evidence / 결정과 근거\n\n")
	if len(packet.Decisions) == 0 {
		b.WriteString("- 결정 없음\n\n")
	} else {
		for _, decision := range packet.Decisions {
			fmt.Fprintf(&b, "- %s — %s, linked evidence %d, same-turn evidence %d\n",
				clip(decision.Title, 90), decision.Status, decision.LinkedEvidenceCount, decision.SameTurnEvidenceCount)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Metric Highlights / 메트릭 하이라이트\n\n")
	if len(packet.MetricHighlights) == 0 {
		b.WriteString("- 메트릭 없음\n\n")
	} else {
		for _, h := range packet.MetricHighlights {
			fmt.Fprintf(&b, "- %s: %s — %s\n", h.Label, h.Value, h.Interpretation)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Requested Review/Help / 요청 사항\n\n")
	if len(packet.RequestedReview) == 0 {
		b.WriteString("- 요청 사항 없음\n")
	} else {
		for _, item := range packet.RequestedReview {
			fmt.Fprintf(&b, "- [%s] %s (%s): %s\n", item.Severity, clip(item.Title, 90), item.Source, clip(item.Action, 140))
		}
	}
	return b.String()
}

func summarizeShare(data *tuiviews.WorklogData) shareSummary {
	var summary shareSummary
	summary.Sessions = len(data.Sessions)
	summary.Turns = len(data.Turns)
	summary.Entries = len(data.Entries)
	for _, t := range data.Turns {
		if shareTurnCompleted(t) {
			summary.CompletedTurns++
		}
		if t.Status == db.TurnStatusActive || t.Status == "" {
			summary.ActiveTurns++
		}
	}
	for _, e := range data.Entries {
		switch e.Kind {
		case db.KindDecision:
			summary.Decisions++
		case db.KindEvidence:
			summary.Evidence++
		}
	}
	for _, b := range data.Blockers {
		if b.Status == db.BlockerStatusOpen || b.Status == "" {
			summary.ActiveBlockers++
		}
	}
	return summary
}

func filterShareData(data *tuiviews.WorklogData, window string, now time.Time) *tuiviews.WorklogData {
	cutoff := shareWindowCutoff(window, now)
	if cutoff.IsZero() {
		return data
	}

	out := &tuiviews.WorklogData{}
	keptEntries := make(map[string]bool)
	for _, session := range data.Sessions {
		if includeShareTime(session.StartedAt, cutoff) {
			out.Sessions = append(out.Sessions, session)
		}
	}
	for _, turn := range data.Turns {
		if includeShareTime(turn.StartedAt, cutoff) {
			out.Turns = append(out.Turns, turn)
		}
	}
	for _, entry := range data.Entries {
		if includeShareTime(entry.CreatedAt, cutoff) {
			out.Entries = append(out.Entries, entry)
			keptEntries[entry.ID] = true
		}
	}
	for _, blocker := range data.Blockers {
		if includeShareTime(blocker.OpenedAt, cutoff) {
			out.Blockers = append(out.Blockers, blocker)
		}
	}
	for _, link := range data.DecisionEvidenceLinks {
		if keptEntries[link.DecisionEntryID] && keptEntries[link.EvidenceEntryID] {
			out.DecisionEvidenceLinks = append(out.DecisionEvidenceLinks, link)
		}
	}
	for _, state := range data.DecisionStates {
		if keptEntries[state.DecisionEntryID] {
			out.DecisionStates = append(out.DecisionStates, state)
		}
	}
	out.Attention = data.Attention
	return out
}

func shareWindowCutoff(window string, now time.Time) time.Time {
	switch window {
	case "day":
		return now.Add(-24 * time.Hour)
	case "week":
		return now.Add(-7 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

func includeShareTime(ts string, cutoff time.Time) bool {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return true
	}
	return !t.Before(cutoff)
}

func shareTurnCompleted(t tuiviews.Turn) bool {
	return t.Status == db.TurnStatusComplete || t.EndedAt.Valid
}

func shareElapsed(t tuiviews.Turn) string {
	if !t.ElapsedMs.Valid {
		return ""
	}
	return fmtDurationMS(t.ElapsedMs.Int64)
}

func shareBlockerAge(b tuiviews.Blocker, now time.Time) string {
	seconds := b.AccumulatedSeconds
	if seconds <= 0 {
		opened, err := time.Parse(time.RFC3339, b.OpenedAt)
		if err == nil {
			seconds = int64(now.Sub(opened).Seconds())
		}
	}
	if seconds < 0 {
		seconds = 0
	}
	return fmtDurationSec(seconds)
}

func shareDecisionStatus(decisionID string, states map[string]tuiviews.DecisionState) string {
	if state, ok := states[decisionID]; ok && state.Status != "" {
		return state.Status
	}
	return db.DecisionStatusAccepted
}

func shareSameTurnEvidence(e tuiviews.Entry, evidenceByTurn map[string]int) int {
	if !e.TurnID.Valid || e.TurnID.String == "" {
		return 0
	}
	return evidenceByTurn[e.TurnID.String]
}

func ensureShareSlices(packet *sharePacket) {
	if packet.CompletedTurns == nil {
		packet.CompletedTurns = []shareTurn{}
	}
	if packet.ActiveBlockers == nil {
		packet.ActiveBlockers = []shareBlocker{}
	}
	if packet.Decisions == nil {
		packet.Decisions = []shareDecision{}
	}
	if packet.MetricHighlights == nil {
		packet.MetricHighlights = []metrics.MetricHighlight{}
	}
	if packet.RequestedReview == nil {
		packet.RequestedReview = []shareReviewItem{}
	}
}
