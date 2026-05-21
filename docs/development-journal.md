# Development Journal

Status date: 2026-05-21

This journal summarizes what changed recently, what remains unstable, and what
AI Tool Suite or sibling projects should learn from `ntts-flightlog`.

The authoritative machine-readable handoff is
`.ai-tool-suite/project-state.json`.

## Recent Changes

`ntts-flightlog` has moved from a live markdown side pane toward a structured
local control surface for long-running AI coding sessions.

Recent product work added:

- `attention --format text|json`
- report-view `주의 필요` section
- `share --window day|week|all --format md|json`
- metric highlights in share output
- `handoff --format text|md|json` as a first-class continuation artifact
- storage and redaction policy for future hook/event ingest
- product boundary documentation in `docs/current-state.md`,
  `docs/product-direction.md`, and
  `docs/user-investor-question-decisions.md`

These changes reinforce the product boundary:

```text
local-first terminal sidecar
for human control during long-running AI coding sessions
```

They also clarify what the project should not become:

```text
cloud dashboard
raw transcript archive
PR automation agent
general project manager
all-tool-call replay
```

## Why It Changed

The early sidecar view was useful for recency, but users also need durable
answers to higher-level questions:

- What is blocked?
- Which decisions lack evidence?
- What should I review next?
- What can I share with a teammate?
- What context survives compaction or agent switching?

`attention`, `share`, and `handoff` convert local worklog state into
machine-readable artifacts without requiring a hosted service.

## What Remains Unstable

Lane/team attribution is the most important missing product boundary. Multiple
agents can write to one worklog today, but parallel worker lanes still depend on
one global active turn pointer. Until explicit lane tracking exists, agent
comparison should remain conservative.

Hook/event ingest now ships as a bounded first slice. Automated ingest remains
gated by `docs/storage-redaction-policy.md`, which defines:

- stored fields
- dropped fields
- secret patterns
- payload retention
- operator-visible audit behavior

The implemented `ntts-flightlog ingest` command reads one JSON object from
stdin, redacts before persistence, stores only bounded `agent_events` audit
fields, deduplicates by `dedupe_key`, and promotes test pass/fail or
permission-denied events into reviewable evidence/blocker candidates.

Pane-rendered text is optimized for Korean terminal scanability and should not
be treated as a stable machine contract. Adapters should use JSON command output
instead.

## What Other Tools May Learn

Reusable patterns:

- Turn-bounded lifecycle: intent, constraints, done-when, entries, outcome,
  evidence, blockers, elapsed time.
- Attention queue: derived operator actions from local state instead of passive
  metrics only.
- Handoff packet: context survival artifact for compaction, restart, agent
  switching, and reviewer handoff.
- Share packet: portable status for PRs, issues, email, or team review without a
  dashboard.
- Local-first trust boundary: SQLite plus markdown under the repo, no account or
  cloud dependency.

These patterns are likely more reusable than the exact CLI or pane UI.

## What AI Tool Suite Should Not Rely On Yet

Do not rely on:

- raw `.ntts-flightlog/main.md` formatting
- pane layout or Korean visible text as a machine schema
- future hook/event fields
- lane/team metadata before the lane model exists
- per-agent scorecards before lane attribution exists
- generated runtime files without an explicit operator opt-in

AI Tool Suite should prefer:

- `ntts-flightlog attention --format json`
- `ntts-flightlog share --format json`
- `ntts-flightlog handoff --format json`
- `ntts-flightlog report --format json`

## Candidates For Shared Contracts

Potential shared contracts after more use:

- `AttentionItem`: severity, source type, source ID, reason, recommended action.
- `WorkTurn`: title, intent, constraints, done-when, outcome, elapsed time,
  agent/lane attribution.
- `EvidenceBoundDecision`: decision, status, linked evidence count, same-turn
  evidence count.
- `HandoffPacket`: status, active turn, open blockers, decisions needing
  evidence, latest evidence, recommended next action.
- `SharePacket`: completed turns, active blockers, decisions, metric
  highlights, requested review/help.

The next product work should keep these contracts small, local, and
closed-network compatible.

## Verification Used Recently

The current product implementation has previously passed:

```bash
go test ./...
go test ./e2e -tags=e2e -count=1
go test ./... -race -count=1
go build -o dist/flightlog ./cmd/flightlog
git diff --check
```

For documentation-only handoff updates, the minimum verification is:

```bash
rg -n "[T]ODO|[T]BD|REPLACE[_]ME|CHANGE[_]ME|[y]our-command-here" docs .ai-tool-suite README.md
git diff --check
go test ./...
```
