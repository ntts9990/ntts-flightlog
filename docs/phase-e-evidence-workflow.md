# Phase E Evidence Workflow

This workflow turns the Phase E retrospective gate into reproducible evidence. It separates scaffold checks from GA-blocking readiness so public contributors can see what is complete and what still needs real data.

## 1. Keep E0 Live Sanity Current

Run after installing or upgrading any agent CLI:

```bash
scripts/sanity_3_agents_tmux.sh
```

Expected result: `docs/e0-3-agent-tmux-sanity.md` shows `pass` for `claude`, `codex`, and `gemini`.

## 2. Write Real Persona Evidence

Use these source artifacts:

- Self-retro: `.omc/specs/alpha-dogfood-log.md`
- Agent-operator: `.omc/specs/v2-agent-operator-decisions.md`
- Team-share: `.omc/specs/v2-team-share-report.md`
- Final summary: `docs/v2-ga-acceptance-evidence.md`

Each persona must cite at least four of the five metrics with concrete values and interpretation:

- `turn_duration`
- `blocker_accumulation`
- `agent_completion`
- `agent_blocker_freq`
- `evidence_bound_decisions`

## 3. Run Adversarial Review

Use `docs/adversarial-review-framework.md` from a separate review session. Save the result as `.omc/specs/v2-adversarial-review.md` and link it from `docs/v2-ga-acceptance-evidence.md`.

## 4. Check Readiness

During evidence collection:

```bash
scripts/phase_e_readiness.sh
```

Before GA:

```bash
scripts/phase_e_readiness.sh --strict
```

Advisory mode allows placeholders while real four-week evidence is still being collected. Strict mode fails on placeholders, missing dated alpha entries, missing behavior-change evidence, missing external acknowledgement, or missing adversarial review evidence.
