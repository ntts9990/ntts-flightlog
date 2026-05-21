# v2 GA Acceptance Evidence

Status: draft evidence scaffold for Phase E. Replace placeholder links with real dated artifacts before GA.

## Self-Retro

Source: `.omc/specs/alpha-dogfood-log.md`

- turn_duration / turn 소요시간: TODO quote from week 4 journal.
- blocker_accumulation / blocker 누적시간: TODO quote from week 4 journal.
- agent_completion / agent 완료율: TODO quote from week 4 journal.
- agent_blocker_freq / agent blocker 빈도: TODO quote from week 4 journal.
- evidence_bound_decisions / evidence-bound decision: TODO quote from week 4 journal.
- Behavior change: TODO `[CHANGED-BY-METRIC: metric_id]` quote.

## Agent-Operator

Source: `.omc/specs/v2-agent-operator-decisions.md`

- turn_duration / turn 소요시간: 2026-05-21 agent-operator decision cites
  21 completed turns, average 2m 9s, with completed turns ranging from 0s to
  10m 14s.
- blocker_accumulation / blocker 누적시간: 2026-05-21 agent-operator decision
  records no blocker rows in the all-window report.
- agent_completion / agent 완료율: 2026-05-21 agent-operator decision records
  codex sessions=4, turns=6, complete=5, completion=83.3%, and unknown
  sessions=12, turns=18, complete=16, completion=88.9%.
- agent_blocker_freq / agent blocker 빈도: 2026-05-21 agent-operator
  decision records codex blocker_freq=0.000 and unknown blocker_freq=0.000.
- evidence_bound_decisions / evidence-bound decision: 2026-05-21
  agent-operator decision records 0/1 decisions linked to evidence, 0.0%.
- Adversarial review: TODO link reviewer log challenging whether citations were useful or gate-driven.

## Team-Share

Source: `.omc/specs/v2-team-share-report.md`

- turn_duration / turn 소요시간: TODO quote from weekly report.
- blocker_accumulation / blocker 누적시간: TODO quote from weekly report.
- agent_completion / agent 완료율: TODO quote from weekly report.
- agent_blocker_freq / agent blocker 빈도: TODO quote from weekly report.
- evidence_bound_decisions / evidence-bound decision: TODO quote from weekly report.
- External recipient ack: TODO dated acknowledgement link or excerpt.

## Gate Summary

- Citation extractor: `scripts/extract_citations.sh`
- Evidence lint: `scripts/lint_evidence_doc.sh docs/v2-ga-acceptance-evidence.md`
- Required before GA: each persona section cites at least 4 of 5 metrics, adversarial review is linked, and external recipient acknowledgement is present.
