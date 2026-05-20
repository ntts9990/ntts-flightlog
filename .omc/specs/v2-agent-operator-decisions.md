# v2 Agent-Operator Decisions

Status: scaffold. Replace every TODO before Phase E strict readiness can pass.

Purpose: record at least one Claude Code / Codex / Gemini operating decision based on concrete `ntts-flightlog report` and `ntts-flightlog agent-stats` values.

## Decision Template

### YYYY-MM-DD — TODO agent comparison decision title

**Context**: TODO describe the work class being assigned or compared.

**Data window**: TODO command and date range, for example `ntts-flightlog report --window week --format text`.

**Metrics cited**:

- `turn_duration`: TODO concrete value and interpretation.
- `blocker_accumulation`: TODO concrete value and interpretation.
- `agent_completion`: TODO concrete value and interpretation.
- `agent_blocker_freq`: TODO concrete value and interpretation.
- `evidence_bound_decisions`: TODO concrete value and interpretation.

**Detection health**:

- `auto_detect_correct_rate`: TODO value from `ntts-flightlog agent-stats`.
- `auto_detect_unknown_rate`: TODO value from `ntts-flightlog agent-stats`.
- `auto_detect_mismatch_rate`: TODO value from `ntts-flightlog agent-stats`.
- `override_rate`: TODO value from `ntts-flightlog agent-stats`.

**Decision**: TODO state which agent or workflow changes because of the metrics.

**Follow-up**: TODO state when this decision will be rechecked.
