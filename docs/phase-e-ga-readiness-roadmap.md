# Phase E / GA Readiness Roadmap

Status date: 2026-05-22

This roadmap keeps the remaining work focused on product readiness instead of
small implementation drift. The current CLI can record, render, share, and check
evidence. GA now depends on proving that the workflow works across real agents
and real evidence artifacts.

## Current Readiness

- `evidence-check` advisory mode passes required artifact presence checks.
- The Agent-Operator evidence scaffold now has dated local metric evidence.
- 3-agent tmux version smoke passes for Claude Code, Codex, and Gemini, and
  hook starter output has been reviewed as non-mutating/redacted.
- Strict readiness still fails by design because real Phase E artifacts remain:
  self-retro journal, team-share external acknowledgement, and adversarial
  review.
- `agent-stats` shows attribution is not ready for cross-agent comparison:
  75.0% unknown sessions in the latest local report.

## GA-Blocking Workstreams

### 1. Three-Agent Attachment

Goal: Codex, Claude Code, and Gemini can each start Flightlog, write turns, and
produce evidence without manual repo-specific setup.

Required artifacts:

- one sanity log per agent showing `auto`, `turn-start`, `entry`, `evidence`,
  `turn-end`, and `handoff`
- hook starter output reviewed against each agent's native payload shape
- `agent-stats` unknown rate trending below the GA threshold once explicit
  `--agent` flags or hooks are used

Stop condition:

- all three agents have dated evidence in the repo and the remaining attribution
  risk is documented.

### 2. Real Persona Evidence

Goal: replace scaffolds with evidence that metrics changed or informed behavior.

Required artifacts:

- Self-Retro: four-week journal entries in `.omc/specs/alpha-dogfood-log.md`
  with at least four distinct metric citations and one real
  `[CHANGED-BY-METRIC: metric_id]` behavior change
- Agent-Operator: at least one operator decision citing at least four metrics
  and acknowledging attribution limits
- Team-Share: one external-facing weekly report plus dated acknowledgement from
  a real recipient

Stop condition:

- `ntts-flightlog evidence-check --strict` has no placeholder failures except
  any explicitly documented non-GA deferral.

### 3. Adversarial Evidence Review

Goal: prevent the evidence gate from becoming self-graded checklist theater.

Required artifacts:

- `.omc/specs/v2-adversarial-review.md` filled by a separate review pass
- high-severity objections answered in `docs/v2-ga-acceptance-evidence.md`
- medium objections either fixed or accepted with an explicit follow-up

Stop condition:

- the review recommendation is `APPROVE` or all blocking objections are closed.

### 4. Metric Interpretation Guide

Goal: users can understand what the five metrics mean before using them for
retrospective decisions.

Required artifacts:

- `docs/metric-interpretation-guide.md`, covering turn duration, blocker
  accumulation, agent completion, agent blocker frequency, and evidence-bound
  decision ratio
- examples of good and bad decisions based on each metric
- explicit warning that same-turn evidence is a signal, while evidence-bound
  decisions require links

Stop condition:

- README and skill docs link the guide, and Phase E reviewers can use it without
  reading source code.

## Non-Goals Before GA

- cloud dashboard
- raw transcript archive
- all-tool-call replay
- PR runner
- broad long-term memory
- ranking agents before attribution is reliable

## Recommended Next Slices

1. Close stale local attention items that are already known bookkeeping noise:
   old active turns and old unlinked product-direction decision.
2. Draft the metric interpretation guide, because it supports every persona.
3. Run a three-agent attachment rehearsal using explicit `--agent` flags.
4. Fill Team-Share only after a real external recipient exists.
5. Run the adversarial review after Self-Retro and Team-Share are no longer
   placeholders.

## Readiness Commands

```bash
ntts-flightlog evidence-check --format text
ntts-flightlog evidence-check --strict
ntts-flightlog agent-stats --window all --format text
ntts-flightlog share --window week --format md
go test ./...
```
