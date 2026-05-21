# User And Investor Question Decisions

Date: 2026-05-21

Purpose: convert questions a user, buyer, or investor would naturally ask into
explicit product decisions for `ntts-flightlog`.

This document is not a backlog dump. It decides what belongs in the project,
what stays out, and what we have already attached to the product.

```text
+---------------------+      asks       +----------------------+
| user / operator    | --------------> | can I stay oriented? |
+---------------------+                 +----------------------+
          |
          | asks
          v
+---------------------+                 +----------------------+
| manager / reviewer  | --------------> | can I trust/share it?|
+---------------------+                 +----------------------+
          |
          | asks
          v
+---------------------+                 +----------------------+
| investor / buyer    | --------------> | is this a category?  |
+---------------------+                 +----------------------+

Decision filter:

  helps human control in a terminal sidecar?        keep
  improves context survival or evidence quality?    keep
  requires cloud/dashboard/PR-agent ownership?      reject for now
  captures raw everything without distillation?      reject
```

## Product Boundary

`ntts-flightlog` is a local-first control surface for long-running AI coding
sessions. It turns messy agent activity into a turn-bounded operating picture:
what is happening, why decisions were made, what evidence supports them, what is
blocked, and what should happen next.

It should not become:

- a cloud observability dashboard
- a PR automation agent
- a raw transcript archive
- a general project manager
- OS-wide memory
- a replacement for Codex, Claude Code, Gemini, Aider, OpenHands, or Copilot

## Decision Legend

```text
ATTACHED   already in the project
NEXT       should be implemented next
LATER      belongs, but only after the core sidecar is stronger
REJECT     does not fit the product boundary now
```

## Questions And Decisions

| Audience | Question They Will Ask | Decision | Product Decision |
| --- | --- | --- | --- |
| User | Can I see what the agent is doing without reading the whole chat? | ATTACHED | Keep the tmux side pane with flat, turns, decisions, blockers, and report views. |
| User | Can I recover after compaction, restart, or switching agents? | ATTACHED | Keep `handoff` as a first-class context packet. |
| User | Can it tell me what needs attention instead of only showing metrics? | ATTACHED | Keep `attention` and the report view's `주의 필요` section. |
| User | Can I share status with a teammate or reviewer? | ATTACHED | Keep `share --format md|json` as the portable status artifact. |
| User | Can it track subagents and team workers? | NEXT | Add explicit lane/team tracking instead of relying only on one global active turn. |
| User | Can it log automatically so agents do not forget? | NEXT | Add bounded hook/event ingest and hook starter kits. |
| User | Can it avoid leaking secrets from automated hooks? | NEXT | Add redaction before storing hook payloads or summaries. |
| User | Can it show every tool call and file read? | REJECT | Do not show raw traces by default. Store bounded events only when they improve orientation. |
| User | Can it work without tmux? | ATTACHED | Keep CLI/markdown/SQLite behavior independent of pane rendering. |
| Manager | Can I tell whether work is actually progressing? | ATTACHED | Use turns, outcomes, blockers, attention, and share summaries. |
| Manager | Can I see decisions and the evidence behind them? | ATTACHED | Keep decision/evidence links and evidence-bound decision metrics. |
| Manager | Can I compare agents or workers? | LATER | Extend `agent-stats` after lane tracking exists; avoid premature scorecards. |
| Manager | Can I generate a weekly status report? | ATTACHED | Use `share --window week --format md`. |
| Manager | Can I enforce evidence before GA? | NEXT | Add `evidence-check` / `evidence-report` wrappers over Phase E scripts and docs. |
| Investor | What is the wedge? | ATTACHED | The wedge is live terminal-side human control during long-running agent work. |
| Investor | Why not Langfuse or LangSmith? | ATTACHED | They trace LLM apps; this orients a human operating coding agents locally. |
| Investor | Why not AgentLogs or raw transcript search? | ATTACHED | Those preserve transcripts; this distills decisions, blockers, evidence, and next actions. |
| Investor | Is there network effect or team value? | NEXT | Team-share and lane tracking create portable evidence artifacts without requiring a cloud account. |
| Investor | Is this a cloud SaaS? | REJECT | Not now. Local-first is the trust boundary and differentiator. |
| Investor | Will it open PRs and run agents? | REJECT | The product monitors and disciplines agents; it should not become the agent runner. |
| Investor | Can this become a category? | LATER | Category proof depends on Phase E evidence that metrics change operator behavior. |
| Buyer | Can this run in private repos without sending data out? | ATTACHED | Repo-local SQLite + markdown remains a core constraint. |
| Buyer | Can security teams audit what it stores? | NEXT | Document storage, redaction, and hook payload policy before deeper ingest. |

## What We Have Attached Already

```text
attached now

  turn files + SQLite state
        |
        +-- decisions + evidence links
        +-- blockers + blocker age/resolution
        +-- report metrics
        +-- attention queue
        +-- handoff packet
        +-- share export
        +-- doctor/preflight
```

Current attached surfaces:

- `ntts-flightlog view flat|turns|decisions|blockers|report`
- `ntts-flightlog handoff --format text|md|json`
- `ntts-flightlog attention --format text|json`
- `ntts-flightlog share --window day|week|all --format md|json`
- `ntts-flightlog --lane <name> turn-start|entry|decision|evidence|blocker|turn-end`
- `ntts-flightlog agent-stats`
- `ntts-flightlog doctor`

## What We Should Attach Next

These NEXT items are ordered by dependency, not just desirability:

```text
storage/redaction policy
        |
        v
lane/team attribution
        |
        v
bounded hook/event ingest
        |
        v
hook starter kits
        |
        v
evidence automation wrappers
```

Storage and redaction policy is a prerequisite gate for any automated ingest.
Hook/event work must not store raw payloads first and defer redaction.
Lane attribution should land before agent comparison or team scorecards, because
the data is not trustworthy until parallel work has explicit ownership.

### 1. Lane / Team Tracking

Decision: NEXT.

Reason: users will ask whether subagents and team workers are tracked. Today,
multiple agents can write to the same worklog, but they share one active
session/turn pointer. That is enough for leader-level tracking, but not enough
for clean parallel lane attribution.

Attach:

- `--lane <name>` global or command-level flag
- optional `--parent-turn <id>` for worker lanes
- lane-aware entries, blockers, decisions, evidence, attention items, and share output
- pane/report labels that show lane without making the UI noisy

Acceptance criteria:

- Two concurrent lanes can write entries without overwriting one global active
  turn pointer.
- `view report`, `attention`, `handoff`, and `share` preserve lane labels where
  they change interpretation.
- A parent turn can summarize worker lane results without becoming a recursive
  tracing graph.
- Existing single-lane workflows keep the same output unless lane metadata is
  present.

Do not attach:

- a full distributed tracing graph
- recursive worker orchestration
- per-tool-call visualization

### 2. Generic Hook/Event Ingest

Decision: NEXT.

Reason: manual logging depends on agent discipline. Hooks are the right
integration point, but raw trace capture would dilute the product.

Attach:

- `ntts-flightlog ingest --source codex|claude|gemini|generic --event <name>`
- JSON stdin support
- `agent_events` table
- event dedupe
- redaction
- promotion rules:
  - test pass -> evidence candidate
  - test fail -> blocker candidate
  - permission denied -> blocker candidate
  - compaction/stop -> handoff reminder

Prerequisite gate:

- `docs/storage-redaction-policy.md` or equivalent must define stored fields,
  dropped fields, secret patterns, payload retention, and operator-visible audit
  behavior before `ingest` ships.

Acceptance criteria:

- Unredacted hook payloads are never stored by default.
- Duplicate hook events do not create duplicate blockers/evidence candidates.
- Promotion rules create reviewable candidates, not silent final decisions.
- JSON ingest failure reports the rejected field/path without leaking the whole
  payload.

Do not attach:

- always-visible raw event stream
- unredacted payload storage
- cloud collector

### 3. Hook Starter Kits

Decision: NEXT.

Reason: users need automatic capture without editing configs by hand, but the
tool should not mutate global agent config unexpectedly.

Attach:

- `ntts-flightlog hooks print --agent codex|claude|gemini`
- `ntts-flightlog hooks doctor`
- docs for opt-in install

Acceptance criteria:

- `hooks print` writes copyable config only; it does not mutate global agent
  config by default.
- `hooks doctor` checks whether the printed hook can reach the local binary and
  worklog path.
- Starter kits document exactly which events are captured and which fields are
  redacted or dropped.

Do not attach yet:

- automatic config mutation by default
- hidden global startup behavior

### 4. Hook/Event Ingest

Decision: ATTACHED AS FIRST SLICE.

Reason: hook events are useful only if they stay local, bounded, and redacted.
The first implementation stores compact audit records and promotes obvious
test/evidence or blocker candidates without saving raw payloads.

Attached:

- `ntts-flightlog ingest --source codex|claude|gemini|generic --event <name>`
- JSON stdin
- `agent_events` audit table
- redaction before persistence
- dedupe by `dedupe_key`
- test-pass evidence candidate promotion
- test-fail and permission-denied blocker candidate promotion

Still not attached:

- automatic hook installation
- raw payload retention
- all-tool-call replay
- silent final decisions from hook events

### 5. Evidence Automation

Decision: NEXT.

Reason: Phase E strict readiness currently depends on manual artifacts. If the
product claims evidence-bound behavior, the project should make missing evidence
visible with one command.

Attach:

- `ntts-flightlog evidence-check`
- `ntts-flightlog evidence-report --persona self-retro|agent-operator|team-share`
- wrappers around readiness/lint scripts
- next required action per persona/metric

Acceptance criteria:

- `evidence-check` exits non-zero for missing strict GA evidence without
  modifying evidence files.
- `evidence-report` cites concrete local artifacts and marks placeholders as
  placeholders.
- The commands reuse Phase E readiness rules instead of creating a parallel
  definition of readiness.
- The report's next action is specific enough for an agent to execute in one
  follow-up turn.

Do not attach:

- fake generated evidence
- relaxing the GA gate to make readiness pass

## What We Should Keep Out For Now

```text
reject for now

  cloud dashboard
  PR automation
  raw transcript archive
  cost dashboard
  prompt management
  broad personal memory
  all-tool-call replay
```

Reasons:

- They compete with larger existing categories.
- They weaken the terminal-side wedge.
- They increase data sensitivity.
- They turn the product from a control surface into infrastructure.

## Roadmap Summary

```text
done:
  handoff -> attention -> share -> storage/redaction policy -> first lane tracking slice

next:
  ingest -> hooks -> evidence automation

later:
  richer agent comparison
  lightweight team evidence bundles
  optional integrations built from share/export artifacts

not now:
  cloud dashboard
  PR runner
  raw tracing platform
```

The practical next implementation sequence is:

1. Generic hook/event ingest
2. Hook starter kits
3. Evidence automation
4. Agent comparison after lane data is trustworthy

## Investor-Safe Positioning

```text
AI coding agents create more work-in-progress than humans can comfortably track.

ntts-flightlog is the local-first control surface that makes long-running agent
work legible: turns, decisions, evidence, blockers, attention, handoff, and
shareable status.

It does not try to be the agent. It makes agent work governable.
```

## Stop Condition For Product Scope

Add a feature only when it improves at least one of these:

- live operator orientation
- context survival
- evidence quality
- blocker visibility
- team handoff/share
- local-first trust

If it mainly improves raw trace completeness, cloud analytics, PR automation, or
general project management, keep it out until the terminal sidecar is excellent.
