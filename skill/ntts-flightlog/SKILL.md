---
name: ntts-flightlog
description: Keep a concise live flightlog for the current Codex task and show it in a tmux side pane while work continues. Use when the user wants main-task progress documented clearly, visible in terminal, or tracked alongside parallel/team-style work without necessarily using team mode.
---

# NTTS Flightlog

Use this skill to maintain a short, structured flightlog while Codex works. It writes a repo-local markdown file and opens a tmux side pane that live-renders the file.

## Design Principles

- Put the answer first: the top block always shows state, focus, next step, and elapsed time.
- Reduce cognitive load: keep each entry short and avoid raw command dumps.
- Use signaling: strong turn blocks mark main tasks; colors distinguish decisions, evidence, and blockers.
- Break each entry into metadata, task title, then detail so narrow panes do not force hard-to-read wrapped lines.
- Avoid flicker: redraw in place, and use file-change redraw when `fswatch` is available.
- Support closure: every `turn-start` should end with `turn-end` so incomplete work is visible.
- Preserve scanability: use stable headings and short lines rather than prose-heavy logs.

For future revisions, see `references/design-rationale.md`.

## What It Provides

- Repo-local flightlog: `.omx/worklog/main.md`
- Live tmux side pane viewer
- Simple append/update commands for milestones, decisions, blockers, verification, and next steps
- Turn tracking with start/end timestamps and elapsed time
- Color-cycled terminal rendering with strong turn blocks so main tasks are easy to distinguish from bullets
- Korean is the default language for pane-visible content.

## Core Workflow

1. Start the pane before multi-step work:

   ```bash
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh auto
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh mode ralph "Ralph 단일-owner 검증 루프로 진행."
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh turn-start "배포 실패 원인 조사"
   ```

2. Update the worklog as the task changes:

   ```bash
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh entry "followups scoring 구현" "EvidenceRef 연결은 유지했고 테스트 대기 중."
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh status "검증 중" "pytest/ruff/mypy 실행 중." "검증 통과 후 커밋."
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh evidence "pytest 통과" "73개 테스트 통과."
   ```

3. End each work turn:

   ```bash
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh turn-end "배포 smoke 검증 완료"
   ```

4. Keep entries concise:
   - target result
   - current status
   - important decisions
   - validation evidence
   - blockers or next step

5. Stop the viewer when no longer needed:

   ```bash
   ~/.codex/skills/ntts-flightlog/scripts/flightlog.sh stop
   ```

## Entry Types

- `status <label> [focus] [next]`: replace the current status block.
- `mode <solo|ralph|team|plan|review|autopilot|other> [detail]`: record the current execution mode.
- `turn-start <title>`: append a colored turn start and reset turn timer.
- `turn-end [summary]`: append a colored turn end with elapsed time.
- `entry <title> [detail]`: append a timestamped milestone.
- `decision <title> [detail]`: append a decision record.
- `evidence <title> [detail]`: append verification evidence.
- `blocker <title> [detail]`: append a blocker.

## Good Use

Use this for long-running or multi-branch tasks where the user may not want to read the chat scrollback. Keep the worklog factual and compact. Do not paste raw private email body, secrets, tokens, or verbose command output.

## Notes

- The script must be run from the target repository root.
- Use `auto` by default. It detects whether Codex is running inside tmux and starts the side pane only when tmux is active.
- If tmux is unavailable, the script still creates and updates the markdown file.
- The live pane defaults to ANSI color rendering. Set `WORKLOG_RENDERER=glow` to use `glow` when installed.
- The viewer uses `fswatch` when installed; otherwise it redraws in place every `REFRESH_SECONDS`.
- Fully automatic startup across all Codex windows requires a global hook; treat that as opt-in because it affects every session.
- Keep pane-visible entries in Korean unless the user asks for another language.
