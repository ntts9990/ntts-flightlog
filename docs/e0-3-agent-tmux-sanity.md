# E0 3-Agent tmux Sanity

Generated: 2026-05-20T08:10:48Z

- tmux: tmux 3.6a
- repository: /Users/sungyub/Documents/Projects/ntts-flightlog

| Agent CLI | Status | Path | Evidence |
| --- | --- | --- | --- |
| `claude` | pass | `/opt/homebrew/bin/claude` | `2.1.139 (Claude Code)` |
| `codex` | pass | `/opt/homebrew/bin/codex` | `codex-cli 0.130.0` |
| `gemini` | pass | `/opt/homebrew/bin/gemini` | `0.42.0` |

## Interpretation

- `pass` means the CLI exists and returned successfully from `--version` inside a tmux pane.
- `missing` means install or PATH setup is still incomplete.
- `fail` or `timeout` means the CLI exists but the noninteractive version smoke is not currently usable.
