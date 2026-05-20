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

- turn_duration / turn 소요시간: TODO quote from agent comparison decision.
- blocker_accumulation / blocker 누적시간: TODO quote from agent comparison decision.
- agent_completion / agent 완료율: TODO quote from agent comparison decision.
- agent_blocker_freq / agent blocker 빈도: TODO quote from agent comparison decision.
- evidence_bound_decisions / evidence-bound decision: TODO quote from agent comparison decision.
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
