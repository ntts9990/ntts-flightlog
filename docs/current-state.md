# NTTS Flightlog Current State

Date: 2026-05-21

Purpose: give an external developer enough context to understand what this
repository is trying to become, what is implemented now, what is still pending,
and what should happen next without reading every planning artifact first.

This document is the canonical project status and handoff entry point. Detailed
product decisions stay in the linked source documents.

## 1. Executive Summary

NTTS Flightlog is a local-first terminal sidecar for long-running AI coding
sessions. It records turns, decisions, evidence, blockers, agent attribution,
and operational metrics into repo-local SQLite while mirroring a readable
markdown worklog. Its live tmux pane keeps the human operator oriented while an
AI coding agent works.

The product is not an agent runner. It does not open pull requests, execute
coding tasks, or try to become a cloud observability system. Its job is to make
agent work governable: what is happening, why decisions were made, what evidence
supports them, what is blocked, and what context must survive compaction or
handoff.

Current implementation status:

- The Go CLI and tmux sidecar are substantially implemented.
- Core state objects exist: sessions, turns, entries, decisions, evidence,
  blockers, metrics, agent attribution, turn anchors, lane ownership, and
  redacted agent event audit records.
- Latest product work adds bounded hook/event ingest with redaction,
  deduplication, and reviewable evidence/blocker candidate promotion.
- Product direction has been clarified around the local-first terminal sidecar
  wedge.
- The next implementation sequence is bounded hook/event ingest, hook starter
  kits, evidence automation, and then richer agent comparison. Storage/redaction
  policy, the first lane/team tracking slice, and bounded ingest are now
  implemented.

## 2. Developer Goal

The developer goal is to make NTTS Flightlog the local control surface for
humans running long AI coding sessions across tools such as Codex, Claude Code,
Gemini CLI, Aider, OpenHands, or Copilot-style agents.

The product should help an operator answer:

- What is the current work unit?
- What changed recently?
- Which decisions were made, and what evidence supports them?
- Which blockers or stale risks need attention?
- What should be handed to the next agent or reviewer?
- Can this status be shared without exposing a raw transcript or cloud account?

Important constraints:

- Local-first: state stays under the repository's `.ntts-flightlog/` directory.
- Agent-agnostic: the CLI works from any shell workflow; agent skills only make
  startup and usage natural.
- Korean-first pane output: the side pane should remain compact and scannable
  for the current operator.
- SQLite plus markdown: SQLite is the source of structured state; markdown keeps
  the worklog readable and v1-compatible.
- Turn as primitive: a turn is the bounded work unit with intent, constraints,
  done-when, entries, outcome, evidence, blockers, and elapsed time.
- No cloud dashboard, no raw transcript archive, no PR automation, no broad
  personal memory, and no all-tool-call replay in the current product boundary.
- No runtime LLM calls currently exist in the product. If future product code
  adds LLM behavior, `docs/llm-prompting-policy.md` is the gating policy for
  prompt contracts, structured outputs, evals, redaction, and prompt-injection
  handling.

## 3. Product Direction

Primary product direction source:

- `docs/product-direction.md`
- `docs/llm-prompting-policy.md`

Product decision record:

- `docs/user-investor-question-decisions.md`

Strategic planning source:

- `.omx/plans/ntts-flightlog-next-product-plan-2026-05-21.md`

The current product thesis:

```text
NTTS Flightlog is the local-first control surface for long-running AI coding
sessions. It turns any coding agent's messy work into a live, turn-bounded
operating picture: what is happening, why choices were made, what evidence
supports them, what is blocked, and what context must survive compaction.
```

The core sidecar surfaces are:

- live worklog view for recency
- turn index for bounded work units
- decision log for rationale
- blocker view for open risks
- report view for metrics and attention
- handoff packet for session recovery
- share export for teammates, reviewers, issues, or email

The product intentionally differs from adjacent categories:

- Langfuse/LangSmith-style observability traces LLM apps; Flightlog orients a
  human operating coding agents locally.
- AgentLogs-style transcript products preserve history; Flightlog distills
  decisions, blockers, evidence, attention, and next action.
- Copilot/OpenHands/Aider-style tools perform coding work; Flightlog monitors
  and disciplines the work.
- Pieces-style memory spans the desktop; Flightlog keeps project-local
  operational memory.

## 4. Current Repository State

Repository:

```text
/Users/sungyub/Documents/Projects/ntts-flightlog
```

Latest committed baseline at time of this status document:

```text
d191874 Make handoffs a first-class continuation artifact
cbcb14e Use Korean labels in the report decision summary
825edbb Show blocker age in the risk view
76db8e3 Preserve summary view headers in the pane
d9922aa Lock doctor preflight behavior with tests
```

Current worktree contains uncommitted changes. The changed tracked files are:

```text
README.md
docs/phase-e-evidence-workflow.md
docs/product-direction.md
internal/cli/cli_commands_test.go
internal/cli/root.go
internal/tui/views/data.go
internal/tui/views/report.go
internal/tui/views/views_extra_test.go
skill/ntts-flightlog/SKILL.md
testdata/golden/cli_help.txt
```

Current untracked files are:

```text
docs/current-state.md
docs/user-investor-question-decisions.md
internal/cli/attention.go
internal/cli/share.go
internal/metrics/attention.go
internal/metrics/attention_test.go
internal/metrics/share.go
internal/metrics/share_test.go
testdata/golden/attention_schema.json
```

The uncommitted work implements and documents:

- `ntts-flightlog attention --format text|json`
- `ntts-flightlog share --window day|week|all --format md|json`
- report view `주의 필요` attention section
- attention-backed requested review/help in share output
- README, skill, and CLI help updates for the new commands
- Phase E workflow guidance for generating team-share evidence from the CLI
- product direction and user/investor decision documentation

## 5. Implemented Surfaces

### CLI And State

Entry point:

- `cmd/flightlog/main.go`

Command registration:

- `internal/cli/root.go`

Current user-facing commands include:

- `auto`, `start`, `stop`, `status`, `mode`
- `turn-start`, `turn-end`, `entry`, `decision`, `evidence`, `blocker`,
  `blocker-resolve`
- `handoff`
- `attention`
- `share`
- `ingest`
- `report`
- `agent-stats`
- `doctor`
- `refresh-anchor`
- `drift-check`
- `path`, `turn-path`, `view`, `migrate`, `self-upgrade`

### Views

The live pane and one-shot renderer expose:

- `flat`: chronological worklog
- `turns`: compact turn index
- `decisions`: ADR-lite decision log
- `blockers`: open-risk board with resolution state
- `report`: operational summary, metrics, and attention items
- `tui`: interactive terminal UI path

Relevant files:

- `internal/tui/views/data.go`
- `internal/tui/views/report.go`
- `internal/tui/views/views_extra_test.go`
- `internal/cli/view.go`
- `internal/cli/start.go`

### Handoff

`handoff` is a first-class continuation artifact for compaction, restart,
agent switching, and reviewer context.

Relevant files:

- `internal/cli/handoff.go`
- `internal/cli/cli_commands_test.go`

### Attention

`attention` turns metric and state signals into operator actions. It covers
stale blockers, decisions without evidence, active turns without evidence,
drift alerts, long turns without outcomes, and agent attribution warnings.

Relevant files:

- `internal/cli/attention.go`
- `internal/metrics/attention.go`
- `internal/metrics/attention_test.go`
- `testdata/golden/attention_schema.json`

### Share

`share` emits portable status for PRs, issues, email, or Phase E team-share
evidence. It includes completed turns, active blockers, decisions/evidence,
metric highlights, and requested review/help.

Relevant files:

- `internal/cli/share.go`
- `internal/metrics/share.go`
- `internal/metrics/share_test.go`
- `docs/phase-e-evidence-workflow.md`

### Doctor And Distribution

`doctor` verifies local installation and worklog health. The repository also
contains install scripts, GoReleaser config, skill package files, and local
build guidance.

Relevant files:

- `internal/cli/doctor.go`
- `scripts/install.sh`
- `scripts/install-from-github.sh`
- `.goreleaser.yml`
- `skill/ntts-flightlog/SKILL.md`

## 6. Documentation And Planning Assets

Use this section as the documentation map.

Product and direction:

- `README.md`: install, usage, command surface, development commands.
- `docs/product-direction.md`: product positioning, view contract, and design
  guardrails.
- `docs/user-investor-question-decisions.md`: attach/next/later/reject
  decisions from user, manager, investor, and buyer questions.

Phase E and GA evidence:

- `docs/phase-e-evidence-workflow.md`: how to produce real Phase E evidence.
- `docs/v2-ga-acceptance-evidence.md`: draft evidence scaffold and GA gate
  summary.
- `docs/adversarial-review-framework.md`: separate review procedure for
  challenging Phase E evidence.
- `docs/phase-e-persona-recruitment.md`: persona recruitment guidance.
- `docs/e0-3-agent-tmux-sanity.md`: local 3-agent tmux sanity evidence.

OMX plans and execution context:

- `.omx/plans/ralplan-ntts-flightlog-v2.md`: original v2 consensus plan.
- `.omx/plans/ntts-flightlog-next-product-plan-2026-05-21.md`: research-backed
  next product plan.
- `.omx/plans/prd-overnight-sidecar-ralph.md`: prior Ralph PRD.
- `.omx/plans/test-spec-overnight-sidecar-ralph.md`: prior Ralph test spec.
- `.omx/context/next-session-flightlog-product-scope-20260521T011941Z.md`:
  recent next-session handoff context for the current uncommitted work.

## 7. Event And Ownership Contract Status

The product now implements the first bounded generic hook/event ingest slice.
It is an intentionally narrow audit surface, not a raw tool-call archive.

Current event-like state is stored through explicit CLI commands plus redacted
ingest:

- entries
- decisions
- evidence
- blockers
- turn start/end
- mode/status
- agent attribution
- anchors and drift alerts
- `agent_events` audit records

Implemented ingest contract:

- `ntts-flightlog ingest --source codex|claude|gemini|generic --event <name>`
- JSON stdin support
- `agent_events` table
- event dedupe
- redaction before storage
- promotion rules that create reviewable evidence/blocker candidates for test
  pass/fail and permission-denied signals

Important ownership rule:

- Storage and redaction policy is documented in
  `docs/storage-redaction-policy.md` and must gate future hook/event ingest.
- Hook ingest must not store unredacted raw payloads by default.
- Raw event streams must stay hidden by default unless promoted into useful
  operator state.
- Product code should not add hook starter kits or automatic config mutation
  before those starter kits can target the redacted ingest surface.

## 8. Implementation Roadmap

Current roadmap from `docs/user-investor-question-decisions.md`:

```text
done:
  handoff -> attention -> share -> storage/redaction policy -> first lane tracking slice -> bounded ingest

next:
  hook starter kits -> evidence automation -> richer agent comparison

later:
  richer agent comparison
  lightweight team evidence bundles
  optional integrations built from share/export artifacts

not now:
  cloud dashboard
  PR runner
  raw tracing platform
```

Immediate product sequence:

1. Commit and publish the bounded ingest slice.
2. Use `docs/storage-redaction-policy.md` as the safety boundary for future
   hook starter kits and any expansion of ingest payload fields.
3. Extend lane/team tracking only where richer parallel ownership is needed.
4. Add hook starter kits as opt-in printed snippets and diagnostics.
5. Add evidence automation wrappers.
6. Extend agent comparison after lane and ingest data are trustworthy.

## 9. Ultragoal And OMX State

Ultragoal source:

- `.omx/ultragoal/brief.md`
- `.omx/ultragoal/goals.json`
- `.omx/ultragoal/ledger.jsonl`

Current `.omx/ultragoal/goals.json` contains one durable goal:

- `G001-ntts-flightlog-phase-e-continuation`
- status: `complete`
- evidence: committed and pushed `4b8747c Make Phase E evidence readiness
  explicit`

That completed goal added Phase E evidence workflow support, advisory readiness
checks, README instructions, and alpha journal strict-counting fixes.

Current active OMX workflow modes at the time this document was created:

```text
none
```

## 10. What Is Done

Done or attached in the product:

- Go CLI entrypoint and command registration.
- Repo-local SQLite storage and markdown mirroring.
- Live tmux side pane with Korean-first menu labels.
- Flat, turns, decisions, blockers, and report views.
- Turn intent anchors and drift-check support.
- Decision/evidence linking and decision lifecycle state.
- Blocker age and blocker resolution state.
- Five local metrics for retrospective review.
- Agent attribution and `agent-stats`.
- `doctor` local preflight.
- `handoff` continuation packet.
- `attention` action queue.
- `share` portable team/reviewer status export.
- Storage and redaction policy for future automated hook payloads.
- First lane/team tracking slice with `--lane`, lane-specific active turns,
  `parent_turn_id`, and lane labels in structured outputs.
- Phase E evidence workflow documentation.
- Product boundary documentation.

## 11. What Is Not Done Yet

Not done or not yet attached:

- Generic hook/event ingest.
- Hook starter kits.
- `evidence-check` and `evidence-report` commands.
- Richer agent comparison based on lane-aware data.
- Real four-week Phase E evidence.
- External recipient acknowledgement for team-share evidence.
- Final adversarial review artifact for GA evidence.
- Cloud dashboard, PR runner, raw trace platform, broad memory, and all-tool-call
  replay are explicitly out of scope for now.

## 12. Immediate Next Step

The next developer should not start hook ingest first.

Recommended order:

1. Review the uncommitted attention/share/product-scope diff.
2. Commit it with the repository Lore Commit Protocol if the diff is accepted.
3. Use `docs/storage-redaction-policy.md` as the prerequisite for future
   `ingest` and hook starter kit implementation.
4. Then start lane/team tracking, because agent comparison and team value depend
   on trustworthy parallel ownership data.

Useful commands:

```bash
git status --short
git diff --check
go test ./...
go test ./e2e -tags=e2e -count=1
go test ./... -race -count=1
go build -o dist/flightlog ./cmd/flightlog
```

## 13. Verification Evidence

Fresh verification reported for the current uncommitted attention/share and
product-scope work:

```text
git diff --check                         PASS
go test ./...                            PASS
go test ./e2e -tags=e2e -count=1         PASS
go build -o dist/flightlog ./cmd/flightlog PASS
go test ./... -race -count=1             PASS
```

Context7 MCP cleanup was also verified:

```text
rg -n "context7|Context7" ~/.codex/config.toml
```

Expected result: no matches.

## 14. Safety And Workflow Rules

Repository rules:

- Do not commit generated runtime state from `.ntts-flightlog/`, `.omx/`,
  `.omc/`, `dist/`, or coverage files.
- Keep diffs small and reviewable.
- Prefer existing package patterns and local helpers.
- Add dependencies only when explicitly justified.
- Preserve Korean output patterns for CLI/pane-visible text.
- Run targeted tests first, then `go test ./...`; use race/e2e/build checks for
  broader changes.

Product safety rules:

- Do not implement hook/event ingest before storage/redaction policy.
- Do not make raw events visible by default.
- Do not mutate global agent configuration by default.
- Do not turn Flightlog into a cloud dashboard, PR automation agent, raw
  transcript archive, or general project manager.
- Product boundary changes should update `docs/product-direction.md` and
  `docs/user-investor-question-decisions.md`.

Commit protocol:

- Commits must follow the Lore Commit Protocol from `AGENTS.md`.
- The intent line should explain why the change was made.
- Include useful trailers such as `Constraint:`, `Rejected:`, `Confidence:`,
  `Scope-risk:`, `Directive:`, `Tested:`, and `Not-tested:` when they add
  decision value.

## 15. Recovery Notes

If a future session loses context, start here:

1. Read this document.
2. Read `docs/product-direction.md`.
3. Read `docs/user-investor-question-decisions.md`.
4. Check `git status --short`.
5. Check `.omx/context/next-session-flightlog-product-scope-20260521T011941Z.md`
   if still present.
6. Run `git diff --check` before editing.

The safe continuation path is to finish the current attention/share commit, then
move to storage/redaction policy before any hook/event ingest work.
