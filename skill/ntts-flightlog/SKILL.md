---
name: ntts-flightlog
description: Keep a concise live flightlog for the current task and show it in a tmux side pane while the agent works. Use when the user wants main-task progress documented clearly, visible in terminal, or tracked alongside parallel/team-style work without necessarily using team mode. Works with any CLI coding agent (Claude Code, Codex, Gemini, etc.) through the Go-based ntts-flightlog CLI.
---

# NTTS Flightlog

Use this skill to maintain a short, structured flightlog while the agent works. It writes a repo-local markdown file and opens a tmux side pane that live-renders the file.

## Design Principles

- Put the answer first: the top block always shows state, focus, next step, and elapsed time.
- Reduce cognitive load: keep each entry short and avoid raw command dumps.
- Use signaling: strong turn blocks mark main tasks; colors distinguish decisions, evidence, and blockers.
- Break each entry into metadata, task title, then detail so narrow panes do not force hard-to-read wrapped lines.
- Avoid flicker: redraw in place, and use file-change redraw when `fswatch` is available.
- Support closure: every `turn-start` should end with `turn-end` so incomplete work is visible.
- Preserve scanability: use stable headings and short lines rather than prose-heavy logs.
- Agent-agnostic: the primary runtime is the Go-based `ntts-flightlog` CLI. The installed `scripts/flightlog.sh` path is kept as a compatibility wrapper and delegates to the Go CLI when available. No coupling to Codex, Claude Code, Gemini, oh-my-codex, or oh-my-claudecode.

For future revisions, see `references/design-rationale.md`.

## What It Provides

- Repo-local flightlog: `.ntts-flightlog/main.md` (falls back to `.omx/worklog/main.md` if that directory already exists, for backward compatibility)
- Per-turn worklog files at `.ntts-flightlog/turns/turn-{N}.md` — every entry inside a turn is mirrored here so each main task has its own scannable history
- Live tmux side pane viewer with a top menu bar: `[1] flat  [2] turns  [3] decisions  [4] blockers  [5] report  [r] reload  [q] quit`
- OSC 8 hyperlinks on turn-start titles — cmd/ctrl-click in iTerm2 / WezTerm / Kitty / Ghostty / VS Code terminal opens the per-turn markdown file in the OS default app
- Simple append/update commands for milestones, decisions, blockers, verification, and next steps
- Turn tracking with start/end timestamps and elapsed time
- Color-cycled terminal rendering with strong turn blocks so main tasks are easy to distinguish from bullets
- Korean is the default language for pane-visible content.

## Core Workflow

The skill should call the installed Go CLI (`ntts-flightlog`, when `~/.local/bin` is on PATH). The absolute skill script path remains available for older agent instructions, but it delegates to the Go CLI when installed. Examples below use the CLI for portability.

1. Start the pane before multi-step work:

   ```bash
   ntts-flightlog auto
   ntts-flightlog mode ralph "Ralph 단일-owner 검증 루프로 진행."
   ntts-flightlog turn-start "배포 실패 원인 조사"
   ```

2. Update the worklog as the task changes:

   ```bash
   ntts-flightlog entry "followups scoring 구현" "EvidenceRef 연결은 유지했고 테스트 대기 중."
   ntts-flightlog status "검증 중" "pytest/ruff/mypy 실행 중." "검증 통과 후 커밋."
   ntts-flightlog evidence "pytest 통과" "73개 테스트 통과."
   ```

3. End each work turn:

   ```bash
   ntts-flightlog turn-end "배포 smoke 검증 완료"
   ```

4. Keep entries concise:
   - target result
   - current status
   - important decisions
   - validation evidence
   - blockers or next step

5. Stop the viewer when no longer needed:

   ```bash
   ntts-flightlog stop
   ```

If the CLI is not on PATH, call the skill script directly. It will delegate to `ntts-flightlog`, `flightlog`, or `NTTS_FLIGHTLOG_BIN` when one is available:

```bash
# Claude Code install
~/.claude/skills/ntts-flightlog/scripts/flightlog.sh auto
# Codex install
~/.codex/skills/ntts-flightlog/scripts/flightlog.sh auto
```

## Entry Types

- `status <label> [focus] [next]`: replace the current status block.
- `mode <solo|ralph|team|plan|review|autopilot|other> [detail]`: record the current execution mode.
- `turn-start <title>`: append a colored turn start, reset turn timer, and create `.ntts-flightlog/turns/turn-{N}.md`.
- `turn-end [summary]`: append a colored turn end with elapsed time. The per-turn file remains; the next `turn-start` opens a new file.
- `entry <title> [detail]`: append a timestamped milestone (mirrored to the active turn file).
- `decision <title> [detail]`: append a decision record (mirrored).
- `evidence <title> [detail]`: append verification evidence (mirrored).
- `blocker <title> [detail]`: append a blocker (mirrored).
- `attention [--window day|week|all] [--format text|json]`: print stale blockers, decisions missing evidence, drift, long turns, and agent attribution warnings as recommended actions.
- `handoff [--format text|md|json]`: print a pasteable session handoff with status, active turn anchors, open blockers, decisions missing evidence, latest evidence, and a recommended next action.
- `share [--window day|week|all] [--format md|json]`: print a PR/issue/email-ready team status report with completed turns, blockers, decisions/evidence, metric highlights, and requested help.

## Viewing

- `ntts-flightlog path` — absolute path of the main worklog file.
- `ntts-flightlog turn-path [N]` — absolute path of turn N (or the most recent turn). Useful in scripts: `code "$(ntts-flightlog turn-path 3)"`.
- `ntts-flightlog view <flat|turns|decisions|blockers|report>` — one-shot ANSI render of the current state. The tmux side pane uses the same renderer behind the menu keys.
- `ntts-flightlog attention [--format text|json]` — action queue for what needs operator attention before continuing.
- `ntts-flightlog handoff [--format text|md|json]` — compact summary to paste into a new agent/session before continuing work.
- `ntts-flightlog share --window week --format md` — portable status report for PRs, issues, email, or Phase E team-share evidence.
- `ntts-flightlog doctor` — local preflight for binary path/version, DB migrations, pane liveness, and installed skill wrapper delegation.
- Inside the pane, press `1`–`5` to switch views, `r` to reload, `q` to quit. Click a turn title with cmd/ctrl held to open the per-turn file (terminal must support OSC 8 hyperlinks).

## Good Use

Use this for long-running or multi-branch tasks where the user may not want to read the chat scrollback. Keep the worklog factual and compact. Do not paste raw private email body, secrets, tokens, or verbose command output.

## Notes

- The script must be run from the target repository root.
- Use `auto` by default. It detects whether the agent is running inside tmux and starts the side pane only when tmux is active.
- If tmux is unavailable, the script still creates and updates the markdown file.
- The live pane defaults to ANSI color rendering. Set `WORKLOG_RENDERER=glow` to use `glow` when installed.
- The viewer uses `fswatch` when installed; otherwise it redraws in place every `REFRESH_SECONDS`.
- Fully automatic startup across all agent windows requires a global hook; treat that as opt-in because it affects every session.
- Keep pane-visible entries in Korean unless the user asks for another language.
