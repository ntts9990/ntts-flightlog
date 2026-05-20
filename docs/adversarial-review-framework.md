# Phase E Adversarial Review Framework

Use this framework for the Agent-Operator persona gate before GA. The review is separate from the authoring session and challenges whether metric citations were genuinely decision-useful.

## Inputs

- `.omc/specs/alpha-dogfood-log.md`
- `.omc/specs/v2-agent-operator-decisions.md`
- `.omc/specs/v2-team-share-report.md`
- `docs/v2-ga-acceptance-evidence.md`
- The five canonical metrics: `turn_duration`, `blocker_accumulation`, `agent_completion`, `agent_blocker_freq`, `evidence_bound_decisions`

## Reviewer Prompt

```text
Review the Phase E evidence for NTTS Flightlog v2.

For each persona section, decide whether each metric citation appears decision-useful or merely gate-driven.
Flag citations that lack a concrete number, interpretation, or resulting decision.
Return:
- pass/fail per persona
- questionable citations with reasons
- missing evidence needed before GA
- whether the external acknowledgement is sufficient
```

## Output

Save the review to `.omc/specs/v2-adversarial-review.md` and link it from `docs/v2-ga-acceptance-evidence.md`.

## Pass Rule

GA requires all high-severity reviewer objections to be answered in the evidence document. Medium objections may ship only with an explicit rationale and follow-up.
