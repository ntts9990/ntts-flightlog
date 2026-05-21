# Storage And Redaction Policy

Status date: 2026-05-21

This policy gates automated hook/event ingest in `ntts-flightlog`.

The product boundary remains:

```text
local-first terminal sidecar
for human control during long-running AI coding sessions
```

This policy exists to keep that boundary intact when agent hooks start sending
structured events.

## Policy Summary

`ntts-flightlog` may store compact operational facts that help the human stay in
control. It must not become a raw transcript archive, all-tool-call replay
system, secret sink, or cloud telemetry collector.

Default posture:

- storage is repo-local under `.ntts-flightlog/`
- network access is not required
- unredacted hook payloads are never stored by default
- raw event streams are hidden from pane views by default
- promoted events create reviewable candidates, not silent final decisions

## Current Storage Boundary

Current stable state is written by explicit CLI commands:

- sessions
- turns
- entries
- decisions
- evidence
- blockers
- decision/evidence links
- decision status
- agent attribution
- turn anchors and drift alerts
- generated markdown mirrors

Generated runtime path:

```text
.ntts-flightlog/
```

This path may contain sensitive project context and must remain gitignored.

## Hook/Event Storage Boundary

`ntts-flightlog ingest` writes an `agent_events` table with only redacted,
bounded fields.

Allowed fields:

| Field | Required | Notes |
| --- | --- | --- |
| `id` | yes | local event ID |
| `session_id` | yes | current flightlog session |
| `turn_id` | no | active turn if available |
| `source` | yes | `codex`, `claude`, `gemini`, or `generic` |
| `event_name` | yes | compact lifecycle/event name |
| `event_time` | yes | ISO timestamp or ingest time |
| `summary` | yes | redacted one-line operational summary |
| `severity` | no | `info`, `warning`, `error` |
| `dedupe_key` | no | stable hash/key for idempotency |
| `promotion_status` | yes | `none`, `candidate`, `promoted`, `rejected` |
| `promoted_entry_id` | no | linked entry/blocker/evidence if promoted |
| `redaction_version` | yes | policy version used |
| `dropped_field_count` | yes | audit count, not raw payload |
| `rejected_reason` | no | compact rejection reason |

Conditionally allowed fields:

| Field | Default | Rule |
| --- | --- | --- |
| `payload_json` | dropped | May be stored only after redaction and only when operator opts in. |
| `command` | summarized | Store command name/category, not full command, unless safe and explicitly useful. |
| `file_paths` | redacted | Store repo-relative paths only; drop home paths and temporary secret paths. |
| `exit_code` | stored | Safe operational field. |
| `duration_ms` | stored | Safe operational field. |

Disallowed fields by default:

- API keys, tokens, cookies, SSH keys, OAuth codes, session IDs, private keys
- `.env` values or full environment dumps
- raw prompt/completion bodies
- raw private email/document/customer data
- full command output when it may include secrets or proprietary source text
- full file contents
- raw browser DOM snapshots
- OS-wide application context unrelated to the current repository
- exact home-directory paths unless converted to repo-relative or basename-only

## Redaction Rules

Redaction must run before persistence, rendering, promotion, and error output.

Minimum secret patterns:

- `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GITHUB_TOKEN`, `GH_TOKEN`
- `*_API_KEY`, `*_TOKEN`, `*_SECRET`, `*_PASSWORD`
- bearer tokens: `Bearer <value>`
- private key blocks: `-----BEGIN ... PRIVATE KEY-----`
- GitHub personal access tokens, fine-grained tokens, and OAuth tokens
- Slack, Discord, Langfuse, Phoenix, MLflow, database, and cloud provider tokens
- HTTP basic auth URLs
- `.env` assignment lines

Replacement format:

```text
[REDACTED:<kind>]
```

Examples:

- `OPENAI_API_KEY=sk-...` -> `OPENAI_API_KEY=[REDACTED:secret]`
- `Authorization: Bearer abc` -> `Authorization: Bearer [REDACTED:token]`
- `/Users/alice/project/file.go` -> `<home>/project/file.go` or repo-relative path

Redaction should prefer dropping a field over storing an uncertain value.

## Retention Policy

Default retention:

- structured operational state is retained until the operator removes
  `.ntts-flightlog/`
- raw hook payloads are not retained
- redacted optional payloads, if ever enabled, must have an explicit retention
  window and operator-visible status

Minimum future controls before optional payload retention:

- `ntts-flightlog ingest --store-payload=redacted` opt-in
- `ntts-flightlog privacy status`
- `ntts-flightlog privacy purge-events`
- documentation that distinguishes generated runtime state from curated fixtures

No future implementation may silently introduce raw payload retention.

## Promotion Rules

Hook events may become user-visible state only through reviewable promotion.

Allowed automatic candidates:

| Event | Candidate |
| --- | --- |
| test command passed | evidence candidate |
| test command failed | blocker candidate |
| permission denied | blocker candidate |
| compaction/session stop | handoff reminder |
| long idle or abandoned turn signal | attention candidate |

Promotion constraints:

- candidates must be labeled as candidates until accepted
- promotion must use redacted summaries only
- duplicate events must not create duplicate blockers/evidence
- final decisions must not be created silently from hook events
- command output must be summarized, not copied verbatim

## Error And Audit Behavior

Ingest failures must not leak full payloads.

Allowed error details:

- rejected field path
- reason code
- event source
- event name
- redaction version

Disallowed error details:

- full rejected payload
- raw secret value
- raw command output
- full environment dump

Operator-visible audit should show:

- number of fields stored
- number of fields dropped
- whether optional redacted payload storage is enabled
- redaction version
- promotion status

## Hook Starter Kit Gate

No hook starter kit may be shipped until the implementation can prove:

- unredacted payloads are never stored by default
- redaction runs before storage and render
- duplicate events are deduped
- rejected JSON reports field/path without leaking the payload
- generated hook snippets are opt-in and printed by default
- no global agent config is mutated without explicit user action

## Adapter Guidance

AI Tool Suite and other adapters should prefer stable JSON command output:

- `ntts-flightlog attention --format json`
- `ntts-flightlog share --format json`
- `ntts-flightlog handoff --format json`
- `ntts-flightlog report --format json`

Adapters should not scrape:

- raw `.ntts-flightlog/main.md`
- pane-rendered text
- raw future `agent_events.payload_json`
- generated runtime files without operator opt-in

## Verification Checklist

Before implementing `ingest`, tests must cover:

- known secret redaction patterns
- nested JSON redaction
- field dropping
- home-path or absolute-path normalization
- duplicate event dedupe
- candidate promotion without silent final decision creation
- rejected JSON error shape
- no raw payload persistence by default

Suggested verification command:

```bash
go test ./...
go test ./... -race -count=1
git diff --check
```
