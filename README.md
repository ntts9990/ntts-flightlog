# NTTS Flightlog

NTTS Flightlog is a small terminal companion that keeps a concise live task log in a tmux side pane while any CLI coding agent works. It is agent-agnostic and ships with skill packages for **Claude Code**, **Codex**, and **Gemini CLI**, plus a standalone `ntts-flightlog` CLI you can drive from any shell.

It is built for long-running AI coding sessions where chat scrollback gets noisy. Flightlog keeps the important state visible:

- current mode: `solo`, `ralph`, `team`, `plan`, `review`, `autopilot`, or `other`
- current focus and next step
- turn start/end with elapsed time
- decisions, evidence, blockers, and milestones
- Korean-first pane output
- low-flicker redraw using `fswatch` when available

Runtime dependencies: `bash`, `tmux`, `awk` (always available on macOS/Linux). Optional: `fswatch` (file-change redraw), `glow` (alternative renderer). No dependency on Codex CLI, Claude Code, Gemini CLI, oh-my-codex, or oh-my-claudecode — the skills are loaded by their respective agents, but the script itself is just bash.

## Preview

```text
작업 기록 PANE  /path/to/repo/.ntts-flightlog/main.md
--------------------------------------------------------------------------------
# 작업 기록

## 현재 상태

- 상태: 대기
- 모드: ralph
- 초점: 마지막 턴 3 완료: 4m 12s.
- 다음: 다음 작업 턴을 기다립니다.
- 경과: 18m 02s

■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
▶ 2026-05-19T04:49:10Z  [turn-3-start]
  followups ranking 개선
────────────────────────────────
시작 시각: 2026-05-19T04:49:10Z.

■ 2026-05-19T04:50:22Z  [evidence]
  pytest 통과
73개 테스트 통과.
```

## Install

One install script handles every supported agent. By default it auto-detects which agent directories already exist (`~/.codex`, `~/.claude`, `~/.gemini`) and installs the skill into each, plus the CLI to `~/.local/bin`.

```bash
git clone https://github.com/ntts9990/ntts-flightlog.git
cd ntts-flightlog
./scripts/install.sh
```

Restrict to a single agent if you prefer:

```bash
./scripts/install.sh --claude     # Claude Code only
./scripts/install.sh --codex      # Codex only
./scripts/install.sh --gemini     # Gemini CLI only
./scripts/install.sh --all        # force-install to all three
./scripts/install.sh --no-cli     # skip ~/.local/bin/ntts-flightlog
```

One-liner (clones to a temp dir, runs `install.sh`, cleans up):

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/ntts9990/ntts-flightlog/main/scripts/install-from-github.sh)
```

Then restart your agent so it picks up the new skill. Typical trigger phrases:

```text
ntts-flightlog 시작해줘
flightlog 켜줘
```

Optional, recommended on macOS for flicker-free redraw:

```bash
brew install fswatch
```

## Usage

Run commands from the repository root you want to log.

```bash
ntts-flightlog auto
ntts-flightlog mode ralph "Ralph 단일-owner 검증 루프"
ntts-flightlog turn-start "배포 실패 원인 조사"
ntts-flightlog decision "API만 재배포" "worker 변경 없음"
ntts-flightlog evidence "pytest 통과" "73개 테스트 통과"
ntts-flightlog turn-end "배포 smoke 검증 완료"
```

If the CLI is not on PATH, call the skill script directly. Path depends on which agent installed it:

```bash
~/.claude/skills/ntts-flightlog/scripts/flightlog.sh auto   # Claude Code
~/.codex/skills/ntts-flightlog/scripts/flightlog.sh auto    # Codex
~/.gemini/skills/ntts-flightlog/scripts/flightlog.sh auto   # Gemini CLI
```

## Commands

```text
ntts-flightlog auto [title]
ntts-flightlog start [title]
ntts-flightlog stop
ntts-flightlog status <label> [focus] [next]
ntts-flightlog mode <solo|ralph|team|plan|review|autopilot|other> [detail]
ntts-flightlog turn-start <title>
ntts-flightlog turn-end [summary]
ntts-flightlog entry <title> [detail]
ntts-flightlog decision <title> [detail]
ntts-flightlog evidence <title> [detail]
ntts-flightlog blocker <title> [detail]
ntts-flightlog path                                  # absolute path of main worklog file
ntts-flightlog turn-path [N]                         # absolute path of turn N (default: latest)
ntts-flightlog view <flat|turns|decisions|blockers>  # one-shot ANSI render
```

## Views and clickable turns

Each `turn-start` creates `.ntts-flightlog/turns/turn-{N}.md` and mirrors every subsequent `entry`/`decision`/`evidence`/`blocker`/`turn-end` into it. That gives every main task its own scannable history alongside the flat main log.

Inside the tmux pane, the top bar is a menu — press `1`–`4` to switch views, `r` to reload, `q` to quit:

```text
[1] 평면   [2] 턴별   [3] 결정   [4] 블로커     [r] 새로고침  [q] 종료
```

- `1` flat — every entry in chronological order (default).
- `2` turns — grouped by `turn-{N}.md`, each turn rendered as its own block.
- `3` decisions — only `[decision]` entries.
- `4` blockers — only `[blocker]` entries.

Turn-start titles in the pane are OSC 8 hyperlinks. cmd/ctrl-click in iTerm2, WezTerm, Kitty, Ghostty, or the VS Code integrated terminal opens the corresponding `turn-{N}.md` in your default editor. Outside the pane, use `ntts-flightlog turn-path N` to get the path directly.

## Environment

```text
WORKLOG_DIR       default: .ntts-flightlog (or .omx/worklog if it already exists, for BC)
WORKLOG_FILE      default: ${WORKLOG_DIR}/main.md
REFRESH_SECONDS   default: 2
PANE_PERCENT      default: 34
WORKLOG_RENDERER  default: color; set to glow to use glow when installed
```

## Distribution Layout

```
skill/ntts-flightlog/     # Agent skill package (shared by Codex, Claude Code, Gemini)
  SKILL.md                # Agent-facing description
  scripts/flightlog.sh    # Main script (self-contained)
  references/             # Design notes
  agents/openai.yaml      # Codex-specific UI metadata (ignored by other agents)
bin/ntts-flightlog        # Standalone CLI (identical to scripts/flightlog.sh)
scripts/install.sh        # Multi-agent installer
scripts/install-from-github.sh  # curl|bash bootstrap
```

## License

MIT
