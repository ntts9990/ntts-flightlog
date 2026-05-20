package metrics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/db"
)

// AgentStat summarizes Phase E agent-attribution health and per-agent work.
type AgentStat struct {
	AgentID        string  `json:"agent_id"`
	Sessions       int64   `json:"sessions"`
	Turns          int64   `json:"turns"`
	CompleteTurns  int64   `json:"complete_turns"`
	Blockers       int64   `json:"blockers"`
	CompletionRate float64 `json:"completion_rate"`
	BlockerFreq    float64 `json:"blocker_freq"`
}

// AgentStatsSummary reports detection quality gates used in Phase E4.
type AgentStatsSummary struct {
	TotalSessions    int64   `json:"total_sessions"`
	UnknownSessions  int64   `json:"unknown_sessions"`
	OverrideSessions int64   `json:"override_sessions"`
	UnknownRate      float64 `json:"auto_detect_unknown_rate"`
	OverrideRate     float64 `json:"override_rate"`
}

// AgentStatsSnapshot is the schema-stable output for `flightlog agent-stats`.
type AgentStatsSnapshot struct {
	Window      string            `json:"window"`
	GeneratedAt string            `json:"generated_at"`
	Summary     AgentStatsSummary `json:"summary"`
	Agents      []AgentStat       `json:"agents"`
}

// QueryAgentStats returns per-agent stats plus auto-detection health.
func QueryAgentStats(d *db.DB, f Filter) (*AgentStatsSnapshot, error) {
	windowPredicate := ""
	if ws := f.windowExpr(); ws != "" {
		windowPredicate = " AND started_at >= " + ws
	}

	summaryQ := `SELECT
		COUNT(*) AS total_sessions,
		SUM(CASE WHEN agent_detected IS NULL OR agent_detected = '' OR agent_detected = 'unknown' THEN 1 ELSE 0 END) AS unknown_sessions,
		SUM(CASE WHEN agent_override IS NOT NULL AND agent_override != '' THEN 1 ELSE 0 END) AS override_sessions
		FROM sessions WHERE 1=1` + windowPredicate
	var summary AgentStatsSummary
	if err := d.QueryRow(summaryQ).Scan(&summary.TotalSessions, &summary.UnknownSessions, &summary.OverrideSessions); err != nil {
		return nil, fmt.Errorf("agent stats summary: %w", err)
	}
	if summary.TotalSessions > 0 {
		summary.UnknownRate = float64(summary.UnknownSessions) / float64(summary.TotalSessions)
		summary.OverrideRate = float64(summary.OverrideSessions) / float64(summary.TotalSessions)
	}

	q := `WITH session_agents AS (
			SELECT id,
			       COALESCE(NULLIF(agent_override, ''), NULLIF(agent_detected, ''), NULLIF(agent_id, ''), 'unknown') AS agent_id
			FROM sessions
			WHERE 1=1` + windowPredicate + `
		),
		turn_stats AS (
			SELECT sa.agent_id,
			       COUNT(DISTINCT t.id) AS turns,
			       COUNT(DISTINCT CASE WHEN t.status = 'complete' THEN t.id END) AS complete_turns,
			       COUNT(b.id) AS blockers
			FROM session_agents sa
			LEFT JOIN turns t ON t.session_id = sa.id
			LEFT JOIN blockers b ON b.turn_id = t.id
			GROUP BY sa.agent_id
		)
		SELECT sa.agent_id,
		       COUNT(DISTINCT sa.id) AS sessions,
		       COALESCE(ts.turns, 0) AS turns,
		       COALESCE(ts.complete_turns, 0) AS complete_turns,
		       COALESCE(ts.blockers, 0) AS blockers
		FROM session_agents sa
		LEFT JOIN turn_stats ts ON ts.agent_id = sa.agent_id
		GROUP BY sa.agent_id
		ORDER BY sa.agent_id`

	rows, err := d.Query(q)
	if err != nil {
		return nil, fmt.Errorf("agent stats query: %w", err)
	}
	defer rows.Close()

	var agents []AgentStat
	for rows.Next() {
		var stat AgentStat
		if err := rows.Scan(&stat.AgentID, &stat.Sessions, &stat.Turns, &stat.CompleteTurns, &stat.Blockers); err != nil {
			return nil, fmt.Errorf("agent stats scan: %w", err)
		}
		if stat.Turns > 0 {
			stat.CompletionRate = float64(stat.CompleteTurns) / float64(stat.Turns)
			stat.BlockerFreq = float64(stat.Blockers) / float64(stat.Turns)
		}
		if f.Agent == "" || f.Agent == stat.AgentID {
			agents = append(agents, stat)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &AgentStatsSnapshot{
		Window:  f.Window,
		Summary: summary,
		Agents:  agents,
	}, nil
}

// FormatAgentStatsJSON serializes agent stats with a stable timestamp hook.
func FormatAgentStatsJSON(snap *AgentStatsSnapshot, generatedAt time.Time) ([]byte, error) {
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	out := *snap
	out.GeneratedAt = generatedAt.Format("2006-01-02T15:04:05Z")
	if out.Agents == nil {
		out.Agents = []AgentStat{}
	}
	return json.MarshalIndent(out, "", "  ")
}
