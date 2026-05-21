package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	tuiviews "github.com/ntts9990/ntts-flightlog/internal/tui/views"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

const handoffListLimit = 5

type handoffPacket struct {
	GeneratedAt              string            `json:"generated_at"`
	WorklogDir               string            `json:"worklog_dir"`
	Status                   handoffStatus     `json:"status"`
	ActiveTurn               *handoffTurn      `json:"active_turn,omitempty"`
	OpenBlockers             []handoffBlocker  `json:"open_blockers"`
	DecisionsNeedingEvidence []handoffDecision `json:"decisions_needing_evidence"`
	LatestEvidence           []handoffEvidence `json:"latest_evidence"`
	RecommendedNext          string            `json:"recommended_next"`
}

type handoffStatus struct {
	Label   string `json:"label"`
	Mode    string `json:"mode"`
	Focus   string `json:"focus,omitempty"`
	Next    string `json:"next,omitempty"`
	Elapsed string `json:"elapsed,omitempty"`
}

type handoffTurn struct {
	ID          string   `json:"id"`
	SequenceNo  int      `json:"sequence_no"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	StartedAt   string   `json:"started_at"`
	Elapsed     string   `json:"elapsed"`
	Intent      string   `json:"intent,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
	DoneWhen    string   `json:"done_when,omitempty"`
	Outcome     string   `json:"outcome,omitempty"`
	Lane        string   `json:"lane,omitempty"`
	ParentTurn  string   `json:"parent_turn,omitempty"`
}

type handoffBlocker struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	OpenedAt string `json:"opened_at"`
	Age      string `json:"age"`
	Detail   string `json:"detail,omitempty"`
}

type handoffDecision struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	Turn      string `json:"turn,omitempty"`
}

type handoffEvidence struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	Turn      string `json:"turn,omitempty"`
}

func newHandoffCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "새 에이전트에게 넘길 현재 작업 요약을 출력합니다",
		Long:  "현재 상태, 활성 턴, 열린 블로커, 근거 없는 결정, 최신 근거를 새 세션에 붙여 넣기 쉬운 패킷으로 출력합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, ""); err != nil {
				return err
			}

			data, err := tuiviews.LoadAll(s.store)
			if err != nil {
				return err
			}
			packet := buildHandoffPacket(s, data, time.Now().UTC())

			switch format {
			case "", "text":
				_, err = fmt.Fprint(cmd.OutOrStdout(), renderHandoffText(packet, false))
			case "md":
				_, err = fmt.Fprint(cmd.OutOrStdout(), renderHandoffText(packet, true))
			case "json":
				var b []byte
				b, err = json.MarshalIndent(packet, "", "  ")
				if err == nil {
					_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
				}
			default:
				err = fmt.Errorf("알 수 없는 handoff format: %s (text|md|json)", format)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "출력 형식 (text|md|json)")
	return cmd
}

func buildHandoffPacket(s *session, data *tuiviews.WorklogData, generatedAt time.Time) *handoffPacket {
	sessionID := s.cfg.ActiveSessionID()
	activeTurnID := s.activeTurnID()
	status := readHandoffStatus(s)
	currentSession := selectSession(data.Sessions, sessionID)
	if sessionID == "" && currentSession != nil {
		sessionID = currentSession.ID
	}
	if (status.Mode == "" || status.Mode == "미지정") && currentSession != nil {
		status.Mode = currentSession.Mode
	}

	packet := &handoffPacket{
		GeneratedAt: generatedAt.Format(time.RFC3339),
		WorklogDir:  s.cfg.Dir,
		Status:      status,
	}

	currentTurn := selectTurn(data.Turns, sessionID, activeTurnID)
	entryByID := entriesByID(data.Entries)
	turnByID := turnsByID(data.Turns)
	if currentTurn != nil {
		packet.ActiveTurn = formatHandoffTurn(*currentTurn, generatedAt)
	}

	packet.OpenBlockers = collectOpenBlockers(data.Blockers, entryByID, turnByID, sessionID, generatedAt)
	packet.DecisionsNeedingEvidence = collectDecisionsNeedingEvidence(data, turnByID, sessionID)
	packet.LatestEvidence = collectLatestEvidence(data.Entries, turnByID, sessionID)
	packet.RecommendedNext = recommendNext(packet)
	return packet
}

func readHandoffStatus(s *session) handoffStatus {
	status := handoffStatus{Mode: s.cfg.CurrentMode()}
	data, err := os.ReadFile(s.cfg.MainMd)
	if err != nil {
		return status
	}
	inStatus := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "## 현재 상태" || line == "## Current Status" {
			inStatus = true
			continue
		}
		if inStatus && strings.HasPrefix(line, "## ") {
			break
		}
		if !inStatus || !strings.HasPrefix(line, "- ") {
			continue
		}
		key, val, ok := strings.Cut(strings.TrimPrefix(line, "- "), ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "상태", "Status":
			status.Label = val
		case "모드", "Mode":
			status.Mode = val
		case "초점", "Focus":
			status.Focus = val
		case "다음", "Next":
			status.Next = val
		case "경과", "Elapsed":
			status.Elapsed = val
		}
	}
	return status
}

func selectSession(sessions []tuiviews.Session, activeID string) *tuiviews.Session {
	var latest *tuiviews.Session
	for i := range sessions {
		if sessions[i].ID == activeID {
			return &sessions[i]
		}
		latest = &sessions[i]
	}
	return latest
}

func selectTurn(turns []tuiviews.Turn, sessionID, activeTurnID string) *tuiviews.Turn {
	var latest *tuiviews.Turn
	for i := range turns {
		t := &turns[i]
		if t.ID == activeTurnID {
			return t
		}
		if sessionID == "" || t.SessionID == sessionID {
			latest = t
		}
	}
	return latest
}

func formatHandoffTurn(t tuiviews.Turn, now time.Time) *handoffTurn {
	title := nullString(t.Title, "(제목 없음)")
	elapsed := "0s"
	if t.ElapsedMs.Valid {
		elapsed = fmtDurationMS(t.ElapsedMs.Int64)
	} else if started, err := time.Parse(time.RFC3339, t.StartedAt); err == nil {
		end := now
		if t.EndedAt.Valid {
			if parsedEnd, err := time.Parse(time.RFC3339, t.EndedAt.String); err == nil {
				end = parsedEnd
			}
		}
		elapsed = fmtDurationSec(int64(end.Sub(started).Seconds()))
	}

	return &handoffTurn{
		ID:          t.ID,
		SequenceNo:  t.SequenceNo,
		Title:       title,
		Status:      t.Status,
		StartedAt:   t.StartedAt,
		Elapsed:     elapsed,
		Intent:      nullString(t.Intent, ""),
		Constraints: parseConstraints(t.ConstraintsJSON.String),
		DoneWhen:    nullString(t.DoneWhen, ""),
		Outcome:     nullString(t.Outcome, ""),
		Lane:        nullString(t.Lane, ""),
		ParentTurn:  nullString(t.ParentTurnID, ""),
	}
}

func parseConstraints(raw string) []string {
	if raw == "" {
		return nil
	}
	var constraints []string
	if err := json.Unmarshal([]byte(raw), &constraints); err == nil {
		return constraints
	}
	return []string{raw}
}

func entriesByID(entries []tuiviews.Entry) map[string]tuiviews.Entry {
	m := make(map[string]tuiviews.Entry, len(entries))
	for _, e := range entries {
		m[e.ID] = e
	}
	return m
}

func turnsByID(turns []tuiviews.Turn) map[string]tuiviews.Turn {
	m := make(map[string]tuiviews.Turn, len(turns))
	for _, t := range turns {
		m[t.ID] = t
	}
	return m
}

func collectOpenBlockers(blockers []tuiviews.Blocker, entryByID map[string]tuiviews.Entry, turnByID map[string]tuiviews.Turn, sessionID string, now time.Time) []handoffBlocker {
	var out []handoffBlocker
	for _, b := range blockers {
		if b.Status != "" && b.Status != "open" {
			continue
		}
		if !blockerMatchesSession(b, entryByID, turnByID, sessionID) {
			continue
		}
		ageSeconds := b.AccumulatedSeconds
		if ageSeconds <= 0 {
			if opened, err := time.Parse(time.RFC3339, b.OpenedAt); err == nil {
				ageSeconds = int64(now.Sub(opened).Seconds())
			}
		}
		item := handoffBlocker{
			ID:       b.ID,
			Title:    b.Title,
			OpenedAt: b.OpenedAt,
			Age:      fmtDurationSec(ageSeconds),
		}
		if b.EntryID.Valid {
			if e, ok := entryByID[b.EntryID.String]; ok {
				item.Detail = nullString(e.Detail, "")
			}
		}
		out = append(out, item)
	}
	return out
}

func blockerMatchesSession(b tuiviews.Blocker, entryByID map[string]tuiviews.Entry, turnByID map[string]tuiviews.Turn, sessionID string) bool {
	if sessionID == "" {
		return true
	}
	if b.TurnID.Valid {
		if t, ok := turnByID[b.TurnID.String]; ok {
			return t.SessionID == sessionID
		}
	}
	if b.EntryID.Valid {
		if e, ok := entryByID[b.EntryID.String]; ok {
			return e.SessionID == sessionID
		}
	}
	return false
}

func collectDecisionsNeedingEvidence(data *tuiviews.WorklogData, turnByID map[string]tuiviews.Turn, sessionID string) []handoffDecision {
	linked := make(map[string]bool, len(data.DecisionEvidenceLinks))
	for _, link := range data.DecisionEvidenceLinks {
		linked[link.DecisionEntryID] = true
	}
	stateByDecision := make(map[string]string, len(data.DecisionStates))
	for _, state := range data.DecisionStates {
		stateByDecision[state.DecisionEntryID] = state.Status
	}

	var out []handoffDecision
	for _, e := range data.Entries {
		if e.Kind != db.KindDecision || linked[e.ID] {
			continue
		}
		if sessionID != "" && e.SessionID != sessionID {
			continue
		}
		status := stateByDecision[e.ID]
		if status == db.DecisionStatusRejected || status == db.DecisionStatusSuperseded {
			continue
		}
		out = append(out, handoffDecision{
			ID:        e.ID,
			Title:     e.Title,
			CreatedAt: e.CreatedAt,
			Turn:      entryTurnLabel(e, turnByID),
		})
	}
	return out
}

func collectLatestEvidence(entries []tuiviews.Entry, turnByID map[string]tuiviews.Turn, sessionID string) []handoffEvidence {
	var out []handoffEvidence
	for i := len(entries) - 1; i >= 0 && len(out) < handoffListLimit; i-- {
		e := entries[i]
		if e.Kind != db.KindEvidence {
			continue
		}
		if sessionID != "" && e.SessionID != sessionID {
			continue
		}
		out = append(out, handoffEvidence{
			ID:        e.ID,
			Title:     e.Title,
			CreatedAt: e.CreatedAt,
			Turn:      entryTurnLabel(e, turnByID),
		})
	}
	return out
}

func entryTurnLabel(e tuiviews.Entry, turnByID map[string]tuiviews.Turn) string {
	if !e.TurnID.Valid {
		return ""
	}
	t, ok := turnByID[e.TurnID.String]
	if !ok {
		return ""
	}
	return fmt.Sprintf("turn-%d", t.SequenceNo)
}

func recommendNext(packet *handoffPacket) string {
	switch {
	case len(packet.OpenBlockers) > 0:
		return "가장 오래 열린 블로커를 해소하거나 해결 근거를 기록하세요."
	case len(packet.DecisionsNeedingEvidence) > 0:
		return "근거 없는 결정을 evidence --link로 보강하세요."
	case packet.ActiveTurn != nil && packet.ActiveTurn.DoneWhen != "":
		return "완료조건을 기준으로 검증한 뒤 turn-end에 결과를 남기세요."
	case packet.ActiveTurn != nil:
		return "현재 턴의 다음 검증 근거를 기록하세요."
	default:
		return "turn-start로 다음 작업 턴을 시작하세요."
	}
}

func renderHandoffText(packet *handoffPacket, markdown bool) string {
	var b strings.Builder
	if markdown {
		b.WriteString("# NTTS Flightlog Handoff\n\n")
	} else {
		b.WriteString("NTTS Flightlog handoff\n")
	}
	fmt.Fprintf(&b, "생성: %s\n", packet.GeneratedAt)
	fmt.Fprintf(&b, "작업로그: %s\n", packet.WorklogDir)
	fmt.Fprintf(&b, "상태: %s\n", fallback(packet.Status.Label, "unknown"))
	fmt.Fprintf(&b, "모드: %s\n", fallback(packet.Status.Mode, "unknown"))
	if packet.Status.Focus != "" {
		fmt.Fprintf(&b, "초점: %s\n", clip(packet.Status.Focus, 120))
	}
	if packet.Status.Next != "" {
		fmt.Fprintf(&b, "다음: %s\n", clip(packet.Status.Next, 120))
	}
	if packet.Status.Elapsed != "" {
		fmt.Fprintf(&b, "세션 경과: %s\n", packet.Status.Elapsed)
	}

	b.WriteString("\n현재 턴\n")
	if packet.ActiveTurn == nil {
		b.WriteString("- 활성 턴 없음\n")
	} else {
		t := packet.ActiveTurn
		fmt.Fprintf(&b, "- turn-%d: %s\n", t.SequenceNo, clip(t.Title, 100))
		fmt.Fprintf(&b, "- 상태/경과: %s / %s\n", fallback(t.Status, "unknown"), t.Elapsed)
		writeOptionalLine(&b, "의도", t.Intent)
		if len(t.Constraints) > 0 {
			writeOptionalLine(&b, "제약", strings.Join(t.Constraints, ", "))
		}
		writeOptionalLine(&b, "완료조건", t.DoneWhen)
		writeOptionalLine(&b, "마지막 결과", t.Outcome)
	}

	writeBlockers(&b, packet.OpenBlockers)
	writeDecisions(&b, packet.DecisionsNeedingEvidence)
	writeEvidence(&b, packet.LatestEvidence)

	b.WriteString("\n추천 다음 행동\n")
	fmt.Fprintf(&b, "- %s\n", packet.RecommendedNext)
	return b.String()
}

func writeBlockers(b *strings.Builder, blockers []handoffBlocker) {
	b.WriteString("\n열린 블로커\n")
	if len(blockers) == 0 {
		b.WriteString("- 없음\n")
		return
	}
	limit := min(len(blockers), handoffListLimit)
	for _, blocker := range blockers[:limit] {
		fmt.Fprintf(b, "- %s (%s): %s\n", clip(blocker.ID, 10), blocker.Age, clip(blocker.Title, 100))
	}
	if len(blockers) > limit {
		fmt.Fprintf(b, "- 외 %d개\n", len(blockers)-limit)
	}
}

func writeDecisions(b *strings.Builder, decisions []handoffDecision) {
	b.WriteString("\n근거 없는 결정\n")
	if len(decisions) == 0 {
		b.WriteString("- 없음\n")
		return
	}
	limit := min(len(decisions), handoffListLimit)
	for _, d := range decisions[:limit] {
		prefix := clip(d.ID, 10)
		if d.Turn != "" {
			prefix = d.Turn + " " + prefix
		}
		fmt.Fprintf(b, "- %s: %s\n", prefix, clip(d.Title, 100))
	}
	if len(decisions) > limit {
		fmt.Fprintf(b, "- 외 %d개\n", len(decisions)-limit)
	}
}

func writeEvidence(b *strings.Builder, evidence []handoffEvidence) {
	b.WriteString("\n최근 근거\n")
	if len(evidence) == 0 {
		b.WriteString("- 없음\n")
		return
	}
	for _, e := range evidence {
		prefix := clip(e.ID, 10)
		if e.Turn != "" {
			prefix = e.Turn + " " + prefix
		}
		fmt.Fprintf(b, "- %s: %s\n", prefix, clip(e.Title, 100))
	}
}

func writeOptionalLine(b *strings.Builder, label, value string) {
	value = clip(singleLine(value), 140)
	if value == "" {
		return
	}
	fmt.Fprintf(b, "- %s: %s\n", label, value)
}

func nullString(v sql.NullString, fallback string) string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return fallback
	}
	return v.String
}

func fallback(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func singleLine(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func clip(s string, limit int) string {
	s = singleLine(s)
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	if limit <= 1 {
		return string(r[:limit])
	}
	return string(r[:limit-1]) + "…"
}
