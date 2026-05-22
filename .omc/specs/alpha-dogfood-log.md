# NTTS Flightlog v2 Alpha Dogfood Log

**Status**: SEEDED (Phase D milestone). Active dogfooding begins Phase E (Month 4 week 2 per plan).
**Persona**: Self-retro (1 of 3 — agent-operator and team-share covered in separate docs at Phase E)
**Goal**: 4 weeks of daily journal with ≥3 daily entries/week, each entry citing ≥1 metric.
**Phase E gate**: ≥4 of 5 distinct metrics cited across the journal + ≥1 `[CHANGED-BY-METRIC: X]` entry.

## How to use this log

After each work day where v2 was used, append a dated entry below following this template:

```markdown
### YYYY-MM-DD (Day N / Week W)

**What I did**: <1-2 sentences>

**Metrics consulted**: <list of metric names actually opened via `flightlog report` — e.g., turn_duration, blocker_accumulation>

**Insight / decision change**: <what you noticed; if a metric triggered a behavior change, tag it>

[CHANGED-BY-METRIC: metric_name]  ← only when a real change was triggered
```

## Citation conventions (for post-hoc citation_extractor.sh)

Use these canonical phrases to ensure the extractor counts citations:
- **turn 소요시간** or **turn duration** → matches `metric_turn_duration`
- **blocker 누적시간** or **blocker accumulation** or **차단 시간** → matches `metric_blocker_accumulation`
- **agent 완료율** or **agent completion** or **완료율** → matches `metric_agent_completion`
- **agent blocker 빈도** or **blocker 빈도** → matches `metric_agent_blocker_freq`
- **evidence-bound decision** or **evidence가 붙은** → matches `metric_evidence_bound_decisions`

## Hard rules (Phase E gate enforcement)

1. Write each daily entry **without consulting the metric list above first**. Open `flightlog report` only to see numbers, not to plan which metrics to mention.
2. Run `scripts/extract_citations.sh .omc/specs/alpha-dogfood-log.md` only **after writing** each daily entry. Do not pre-tune entries to clear the gate.
3. The `[CHANGED-BY-METRIC: X]` tag must reflect a real decision change, not filler. Acceptable: "I shortened lunch breaks because turn_duration showed me afternoon entries were 40% longer than morning ones." Unacceptable: "I noticed the metric." (no behavior change).

## Weeks

### Week 1 — Foundation use
Started: 2026-05-22

### 2026-05-22 (Day 1 / Week 1)

**What I did**: Continued Phase E readiness work under Ralph. I refreshed
three-agent attachment evidence, added a metric interpretation guide, and then
attempted the full 1-5 GA-readiness sequence without fabricating external or
four-week evidence.

**Metrics consulted**: `turn_duration`, `blocker_accumulation`,
`agent_completion`, `agent_blocker_freq`, `evidence_bound_decisions`.

**Insight / decision change**: The all-window share showed 24 completed turns
with an average turn duration of 2m 14s, no blocker accumulation, and 0/1
evidence-bound decisions. `agent-stats` still showed 75.0% unknown sessions,
so agent completion and agent blocker frequency are not yet reliable for
ranking agents. I changed the next-step behavior from "try to make strict pass"
to "record honest blockers and require real external/self-retro evidence before
strict GA."

[CHANGED-BY-METRIC: evidence_bound_decisions]

### Week 2

### Week 3

### Week 4

## Phase E gate self-check (run at week 4 end)

```bash
scripts/extract_citations.sh .omc/specs/alpha-dogfood-log.md
# Expected output: ≥4 distinct metrics cited + ≥1 [CHANGED-BY-METRIC] tag
```

If <4 metrics cited or 0 behavior changes triggered, this is a **gate miss signal** — escalate to plan iter 3 to redesign metric prominence in `flightlog report` text output (advisory from Phase B.5 rehearsal already flagged self-retro persona missing `agent_blocker_freq`).
