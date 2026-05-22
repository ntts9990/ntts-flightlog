# E0 3-Agent tmux Sanity

Generated: 2026-05-22T00:25:15Z

- tmux: tmux 3.6a
- repository: /Users/sungyub/Documents/Projects/ntts-flightlog

| Agent CLI | Status | Path | Evidence |
| --- | --- | --- | --- |
| `claude` | pass | `/opt/homebrew/bin/claude` | `2.1.139 (Claude Code)` |
| `codex` | pass | `/opt/homebrew/bin/codex` | `codex-cli 0.132.0` |
| `gemini` | pass | `/opt/homebrew/bin/gemini` | `0.42.0` |

## Interpretation

- `pass` means the CLI exists and returned successfully from `--version` inside a tmux pane.
- `missing` means install or PATH setup is still incomplete.
- `fail` or `timeout` means the CLI exists but the noninteractive version smoke is not currently usable.

## Hook Starter Review

Reviewed: 2026-05-22T00:25:15Z

Commands checked:

- `ntts-flightlog hooks print --agent codex`
- `ntts-flightlog hooks print --agent claude`
- `ntts-flightlog hooks print --agent gemini`

Result:

- `codex`, `claude`, and `gemini` hook starter kits all print copyable commands
  without mutating global config.
- Each starter sends only `source`, `event_name`, `summary`, and `dedupe_key`
  into `ntts-flightlog ingest`.
- Each starter explicitly drops raw stdout/stderr, raw prompts, full
  environment, and secrets from ingest.

Remaining attachment risk:

- This proves CLI availability inside tmux panes and non-mutating hook starter
  output. It does not prove live agent hook installation or full native hook
  payload compatibility.

## Hook Starter Ingest Smoke

Ran: 2026-05-22T00:33:23Z

Commands executed:

- `ntts-flightlog ingest --source codex --event session.hook`
- `ntts-flightlog ingest --source claude --event session.hook`
- `ntts-flightlog ingest --source gemini --event session.hook`

Result:

- Codex event accepted with `promotion_status: none`,
  `redaction_version: storage-redaction-2026-05-21`, and
  `dropped_field_count: 0`.
- Claude event accepted with `promotion_status: none`,
  `redaction_version: storage-redaction-2026-05-21`, and
  `dropped_field_count: 0`.
- Gemini event accepted with `promotion_status: none`,
  `redaction_version: storage-redaction-2026-05-21`, and
  `dropped_field_count: 0`.

Remaining attachment risk:

- This proves the hook starter payload shape is accepted by ingest. It still
  does not prove installation in each agent's native hook runner.
