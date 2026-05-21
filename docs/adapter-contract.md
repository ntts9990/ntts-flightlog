# AI Tool Suite Adapter Contract

Status date: 2026-05-21

This document defines the stable surfaces AI Tool Suite can use to understand
`ntts-flightlog` without copying internal code or scraping raw transcripts.

## Integration Boundary

`ntts-flightlog` is a local-first terminal sidecar for long-running AI coding
sessions. AI Tool Suite should treat it as an agent-operation evidence source:
it can provide status, attention, handoff, and share artifacts about work that
happened locally.

It is not an evaluation judge, hosted telemetry service, PR automation agent, or
raw tool-call archive.

## Stable Commands

Run commands from the repository whose work is being logged.

```bash
ntts-flightlog attention --format json --window week
ntts-flightlog share --format json --window week
ntts-flightlog handoff --format json
ntts-flightlog report --format json --window week
ntts-flightlog agent-stats --format json --window week
ntts-flightlog ingest --source codex --event test.finished < event.json
```

All commands are closed-network by default. They read local `.ntts-flightlog/`
state and write to stdout. They require no API keys or hosted services.

## Stable Artifacts

| Artifact | Command Or Path | Schema Version | Stability | Use |
| --- | --- | --- | --- | --- |
| Attention JSON | `ntts-flightlog attention --format json` | `attention.v1` | Stable | Operator risk/action evidence |
| Share JSON | `ntts-flightlog share --format json` | `share.v1` | Stable | Portable session/team status |
| Handoff JSON | `ntts-flightlog handoff --format json` | `handoff.v1` | Stable | Continuation context |
| Report JSON | `ntts-flightlog report --format json` | `report.v1` | Stable | Operational metrics summary |
| Ingest JSON response | `ntts-flightlog ingest --source <agent> --event <name>` | `ingest.v0` | Beta | Redacted hook/event audit intake |
| Golden attention fixture | `testdata/golden/attention_schema.json` | `attention.v1` | Stable | Adapter fixture and schema example |
| Runtime state | `.ntts-flightlog/` | `runtime-state.v1` | Generated | SQLite, markdown, turn files, pane metadata |

## Example: Attention JSON

Representative fields:

```json
{
  "generated_at": "2026-05-21T04:30:00Z",
  "window": "week",
  "agent": "",
  "summary": {
    "total": 2,
    "high": 1,
    "medium": 1,
    "low": 0
  },
  "items": [
    {
      "severity": "high",
      "source_type": "blocker",
      "source_id": "blk_123",
      "title": "Open blocker is stale",
      "reason": "Blocker has been open longer than the configured attention window.",
      "recommended_action": "Resolve it, update it, or move it into the next handoff."
    }
  ]
}
```

Fields safe to rely on:

- `generated_at`
- `window`
- `agent`
- `summary.total`
- `summary.high`
- `summary.medium`
- `summary.low`
- `items[].severity`
- `items[].source_type`
- `items[].source_id`
- `items[].title`
- `items[].reason`
- `items[].recommended_action`

## Example: Share JSON

Representative fields:

```json
{
  "generated_at": "2026-05-21T04:30:00Z",
  "window": "week",
  "summary": {
    "sessions": 1,
    "turns": 4,
    "completed_turns": 3,
    "active_turns": 1,
    "entries": 18,
    "decisions": 2,
    "evidence": 5,
    "active_blockers": 1
  },
  "completed_turns": [],
  "active_blockers": [],
  "decisions": [],
  "metric_highlights": [],
  "requested_review": []
}
```

Fields safe to rely on:

- `summary`
- `completed_turns[].id`
- `completed_turns[].sequence`
- `completed_turns[].title`
- `completed_turns[].agent`
- `completed_turns[].started_at`
- `completed_turns[].ended_at`
- `completed_turns[].elapsed`
- `completed_turns[].outcome`
- `active_blockers[]`
- `decisions[]`
- `metric_highlights[]`
- `requested_review[]`

## Beta: Ingest JSON Response

`ingest` reads JSON from stdin and returns a bounded response. It never stores
the raw payload by default.

Fields safe to rely on during beta:

- `ok`
- `event_id`
- `duplicate`
- `promotion_status`
- `promoted_entry_id`
- `redaction_version`
- `dropped_field_count`

## Experimental Fields

Do not treat these as adapter-stable yet:

- additional hook/event fields beyond the `ingest.v0` response
- lane/team metadata beyond current `lane.v1` fields
- agent comparison/scorecard fields before lane and ingest coverage are proven
- pane-rendered Korean text layout
- raw `.ntts-flightlog/main.md` prose layout

## Required Secrets

None.

`ntts-flightlog` must remain useful without OpenAI, Anthropic, GitHub, cloud
storage, or hosted observability credentials.

## Network Posture

Default posture: `offline`.

The CLI reads and writes local files only. The tmux side pane is a local terminal
renderer. Release install scripts may use the network to download a binary, but
runtime logging, reports, attention, handoff, and share artifacts do not require
network access.

## Generated Output Paths

Generated runtime paths:

```text
.ntts-flightlog/
dist/
cli_coverage.out
```

`.ntts-flightlog/` contains repo-local SQLite and markdown worklog state. It may
contain sensitive project context and should stay out of git.

Curated fixtures:

```text
testdata/golden/
e2e/
internal/**/*_test.go
```

## Structured Error Behavior

Current CLI errors are command-line errors returned on stderr/stdout through
Cobra. JSON commands should be invoked in test-covered happy paths for adapter
ingestion. AI Tool Suite should treat non-zero exit status as a failed artifact
generation and should not try to parse partial output as trusted evidence.

Future hook/event ingest must report rejected fields or paths without storing or
printing full unredacted payloads.

## Rollback Path

If an adapter breaks:

1. Stop ingesting `ntts-flightlog` runtime state directly.
2. Fall back to `ntts-flightlog share --format json --window week`.
3. If share JSON is affected, fall back to `ntts-flightlog handoff --format json`.
4. If machine-readable output is affected, treat the project as profile-only
   until the golden tests are updated and pass.

## Regenerate Representative Artifacts

Build the local binary:

```bash
go build -o dist/flightlog ./cmd/flightlog
```

Generate representative artifacts from the current repo-local runtime state:

```bash
dist/flightlog attention --format json --window week
dist/flightlog share --format json --window week
dist/flightlog handoff --format json
dist/flightlog report --format json --window week
```

Verify contracts:

```bash
go test ./...
go test ./e2e -tags=e2e -count=1
go test ./... -race -count=1
git diff --check
```
