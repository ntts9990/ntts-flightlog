# v2 Agent-Operator Decisions

Status: seeded with local operator evidence from 2026-05-21.

Purpose: record at least one Claude Code / Codex / Gemini operating decision based on concrete `ntts-flightlog report` and `ntts-flightlog agent-stats` values.

## Decision

### 2026-05-21 — Keep Codex as the primary operator until attribution improves

**Context**: Visual-report implementation, skill synchronization, docs drift cleanup,
and commit/push work were handled in the current local repo session. The next
agent-routing decision is whether to compare Codex, Claude Code, and Gemini on
throughput now, or first fix attribution quality so comparison data is useful.

**Data window**: `ntts-flightlog report --window all --format text`,
`ntts-flightlog report --window all --format json`, and
`ntts-flightlog agent-stats --window all --format text`, generated
2026-05-21T13:22:53Z.

**Metrics cited**:

- `turn_duration`: 21 completed turns, average 2m 9s in the share summary;
  individual completed turns ranged from 0s to 10m 14s in the all-window report.
  Interpretation: the current work is already sliced small enough for one
  operator loop, so parallel agent comparison is not the immediate bottleneck.
- `blocker_accumulation`: no blocker rows in the all-window report. Interpretation:
  there is no recorded blocker pressure requiring a handoff to another agent.
- `agent_completion`: report metrics had no agent-attributed completion rows,
  while `agent-stats` showed codex sessions=4, turns=6, complete=5,
  completion=83.3%, and unknown sessions=12, turns=18, complete=16,
  completion=88.9%. Interpretation: completion exists, but attribution is too
  sparse for a fair cross-agent productivity decision.
- `agent_blocker_freq`: report metrics had no agent-attributed blocker rows;
  `agent-stats` showed codex blocker_freq=0.000 and unknown blocker_freq=0.000.
  Interpretation: blocker frequency does not distinguish agents yet.
- `evidence_bound_decisions`: 0/1 decisions linked to evidence, 0.0%.
  Interpretation: before using reports for external comparison, decisions need
  explicit evidence links rather than only same-turn evidence signals.

**Detection health**:

- `auto_detect_correct_rate`: 25.0% (4/16 sessions).
- `auto_detect_unknown_rate`: 75.0% (12/16 sessions).
- `auto_detect_mismatch_rate`: 0.0% (0/16 sessions).
- `override_rate`: 0.0% (0/16 sessions).

**Decision**: Keep Codex as the primary operator for near-term repo maintenance,
and do not use the current metrics to rank Claude Code or Gemini yet. The next
operator improvement is to use explicit `--agent` flags or hook starter kits so
future sessions produce agent-attributed completion and blocker rows.

**Follow-up**: Recheck after at least one Codex, Claude Code, and Gemini session
has explicit agent attribution and at least one linked evidence-bound decision.
