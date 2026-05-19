# NTTS Flightlog

NTTS Flightlog is a small Codex/terminal companion that keeps a concise live task log in a tmux side pane.

It is built for long-running AI coding sessions where chat scrollback gets noisy. Flightlog keeps the important state visible:

- current mode: `solo`, `ralph`, `team`, `plan`, `review`, `autopilot`, or `other`
- current focus and next step
- turn start/end with elapsed time
- decisions, evidence, blockers, and milestones
- Korean-first pane output
- low-flicker redraw using `fswatch` when available

## Preview

```text
작업 기록 PANE  /path/to/repo/.omx/worklog/main.md
--------------------------------------------------------------------------------
# Codex 작업 기록

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

### Codex skill install

```bash
git clone https://github.com/ntts9990/ntts-flightlog.git
mkdir -p ~/.codex/skills
cp -R ntts-flightlog/skill/ntts-flightlog ~/.codex/skills/
```

Restart Codex so the skill list is reloaded. Then call:

```text
$ntts-flightlog 시작해줘
```

### CLI install

```bash
git clone https://github.com/ntts9990/ntts-flightlog.git
mkdir -p ~/.local/bin
cp ntts-flightlog/bin/ntts-flightlog ~/.local/bin/
chmod +x ~/.local/bin/ntts-flightlog
```

Optional, recommended on macOS:

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

If you installed only the Codex skill, use:

```bash
~/.codex/skills/ntts-flightlog/scripts/flightlog.sh auto
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
ntts-flightlog path
```

## Environment

```text
WORKLOG_DIR       default: .omx/worklog
WORKLOG_FILE      default: .omx/worklog/main.md
REFRESH_SECONDS   default: 2
PANE_PERCENT      default: 34
WORKLOG_RENDERER  default: color; set to glow to use glow when installed
```

## Distribution Notes

This repo intentionally ships as:

- a Codex skill under `skill/ntts-flightlog`
- a standalone CLI script under `bin/ntts-flightlog`

That keeps installation simple for Codex, Claude Code, or any terminal workflow.

## License

MIT
