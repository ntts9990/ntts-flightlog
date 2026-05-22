# Metric Interpretation Guide

Status date: 2026-05-22

This guide is GA-blocking because Phase E evidence depends on people using the
metrics to make or challenge decisions. A metric citation is useful only when it
connects a concrete value to an operator choice, a product decision, or a review
objection.

## How To Use Metrics

Use metrics as signals, not verdicts. A good retrospective sentence has three
parts:

1. the metric and value observed
2. the interpretation in the current work context
3. the decision, behavior change, or objection that follows

Example:

```text
turn_duration showed the last three completed turns were all under three
minutes, so I kept the next slice small and did not split it across agents.
```

Weak citation:

```text
I looked at turn_duration.
```

## The Five Metrics

### 1. `turn_duration`

What it measures: elapsed time for completed turns.

Use it to answer:

- Are work slices small enough to review and recover?
- Are some tasks consistently too large for one turn?
- Did a change make work more interruptible or more sprawling?

Good decisions:

- Split a future task when several turns exceed the team's review comfort zone.
- Keep a solo loop when recent turns are short and finish cleanly.
- Investigate long completed turns that lack clear outcomes.

Bad decisions:

- Treating shorter duration as automatically better.
- Comparing agents by duration before task type and attribution are reliable.
- Ignoring whether a short turn produced evidence or only chat.

Evidence example:

```text
turn_duration: 21 completed turns averaged 2m 9s; this supports keeping the
next roadmap slice solo rather than launching a team.
```

### 2. `blocker_accumulation`

What it measures: how long blockers remain open.

Use it to answer:

- Is the session accumulating unresolved risk?
- Do blockers need owner/action updates?
- Is a handoff blocked by external dependency rather than implementation work?

Good decisions:

- Stop feature work and resolve stale blockers first.
- Convert an ambiguous blocker into a concrete waiting condition.
- Escalate when a blocker has aged beyond the expected response window.

Bad decisions:

- Counting absence of blockers as proof that nothing is risky.
- Closing blockers only to make the metric look clean.
- Treating blocker age without reading the blocker title and latest detail.

Evidence example:

```text
blocker_accumulation: no open blockers were recorded, so the next readiness
risk is evidence quality rather than blocked implementation.
```

### 3. `agent_completion`

What it measures: completion rate by detected or overridden agent.

Use it to answer:

- Which agent lanes are finishing recorded turns?
- Is an agent integration producing complete turns or abandoned ones?
- Is attribution good enough to compare agents?

Good decisions:

- Require explicit `--agent` flags when unknown attribution is high.
- Rehearse a three-agent flow before ranking agents by completion.
- Compare agents only after task type and attribution are controlled.

Bad decisions:

- Ranking agents while most sessions are `unknown`.
- Treating completion rate as quality without evidence and review results.
- Penalizing an agent for turns that were actually operator-aborted.

Evidence example:

```text
agent_completion: Codex had 5/6 complete turns, but 12/16 sessions were
unknown, so the decision is to improve attribution before comparing agents.
```

### 4. `agent_blocker_freq`

What it measures: blocker frequency by agent or lane.

Use it to answer:

- Which agent lanes encounter blockers often?
- Are blocker-heavy lanes missing setup, permissions, or context?
- Does a specific workflow need better preflight checks?

Good decisions:

- Improve setup documentation for an agent with repeated permission blockers.
- Add preflight checks when blocker frequency clusters around one integration.
- Review blocked sessions before claiming an agent is unreliable.

Bad decisions:

- Treating zero blockers as proof of high quality.
- Comparing agents when blockers are not consistently recorded.
- Ignoring the severity and type of blockers.

Evidence example:

```text
agent_blocker_freq: Codex and unknown lanes both showed 0.000 blocker frequency;
this does not distinguish agents, so no agent-ranking decision should be made.
```

### 5. `evidence_bound_decisions`

What it measures: the ratio of decision entries explicitly linked to evidence.

Use it to answer:

- Are important decisions reviewable after the session?
- Can another operator see why a decision was made?
- Are retrospectives relying on linked evidence or only nearby context?

Good decisions:

- Link evidence before using a decision in a report or GA artifact.
- Supersede stale decisions when the rationale has changed.
- Treat same-turn evidence as a useful signal, not as a replacement for links.

Bad decisions:

- Claiming a decision is evidence-bound because evidence appears in the same
  turn but is not linked.
- Linking irrelevant evidence just to improve the ratio.
- Ignoring rejected or superseded decisions when interpreting the metric.

Evidence example:

```text
evidence_bound_decisions: 0/1 decisions were explicitly linked, so the next
operator action is to link evidence or supersede the product-direction decision.
```

## Phase E Evidence Rules

For Self-Retro, Agent-Operator, and Team-Share evidence:

- cite at least four of the five metrics with concrete values
- explain what each value means in context
- state the decision, behavior change, or review objection that follows
- avoid metric shopping after writing the evidence
- keep external acknowledgements dated and attributable

For adversarial review:

- challenge whether each metric changed behavior or only satisfied the gate
- flag missing concrete values
- flag decisions that should have linked evidence but only cite same-turn
  context
- request changes when attribution is too weak for agent comparison

## GA Readiness Checklist

- [ ] Self-Retro cites at least four metrics and includes a real
      `[CHANGED-BY-METRIC: metric_id]` behavior change.
- [ ] Agent-Operator cites at least four metrics and explains attribution
      limits before comparing agents.
- [ ] Team-Share cites at least four metrics in plain language and includes a
      dated external acknowledgement.
- [ ] Adversarial review has no open high-severity objections.
- [ ] `ntts-flightlog evidence-check --strict` passes.
