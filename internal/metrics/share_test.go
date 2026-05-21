package metrics_test

import (
	"strings"
	"testing"

	"github.com/ntts9990/ntts-flightlog/internal/metrics"
)

func TestBuildMetricHighlights(t *testing.T) {
	ms := int64(120000)
	snap := &metrics.Snapshot{
		TurnDurations: []metrics.TurnDuration{
			{TurnID: "t1", AgentID: "codex", ElapsedMS: &ms},
		},
		BlockerAccumulations: []metrics.BlockerAccumulation{
			{BlockerID: "b1", OpenedAt: "2026-05-21T10:00:00Z", AccumulatedSeconds: 3600},
		},
		AgentCompletion: []metrics.AgentCompletion{
			{AgentID: "codex", CompletionRate: 0.5, CompleteCount: 1, TotalCount: 2},
		},
		AgentBlockerFreq: []metrics.AgentBlockerFreq{
			{AgentID: "codex", BlockerFreq: 0.5, BlockerCount: 1, TurnCount: 2},
		},
		EvidenceBound: metrics.EvidenceBoundDecisions{Ratio: 0.5, LinkedCount: 1, TotalCount: 2},
	}
	got := metrics.BuildMetricHighlights(snap)
	if len(got) != 5 {
		t.Fatalf("BuildMetricHighlights len = %d, want 5", len(got))
	}
	for _, h := range got {
		if h.Metric == "" || h.Label == "" || h.Value == "" || h.Interpretation == "" {
			t.Fatalf("highlight missing fields: %#v", h)
		}
	}
	if !strings.Contains(got[4].Value, "1/2") {
		t.Fatalf("evidence-bound highlight value = %q", got[4].Value)
	}
}
