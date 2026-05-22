# NTTS Flightlog

NTTS Flightlog is a local-first terminal companion for long-running AI coding sessions. It records turns, decisions, evidence, blockers, and agent context into SQLite, mirrors a readable `main.md`, and can keep the current worklog visible in a tmux side pane.

The project is agent-agnostic: it works with **Claude Code**, **Codex**, **Gemini CLI**, or any shell workflow. Agent-specific skill packages are included only to make startup natural inside each agent.

Flightlog keeps the important state visible:

- current mode: `solo`, `ralph`, `team`, `plan`, `review`, `autopilot`, or `other`
- current focus, next step, and turn intent anchors
- turn start/end with elapsed time
- decisions, evidence, blockers, and milestones
- five local metrics for retrospective review
- Korean-first pane output
- pinned tmux menu header with no scrollback accumulation

Runtime dependencies for the CLI are bundled in the Go binary. The live side pane requires `tmux`; install scripts use standard shell tools. Flightlog does not send telemetry or require cloud services.

## Preview

```text
[1]평면 [2]턴별 [3]결정 [4]블로커 [5]리포트 [6]시각화
[r]새로고침 [q]종료
----------------------------------------
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

✓ 2026-05-19T04:50:22Z  [evidence]
  pytest 통과
73개 테스트 통과.
```

## Install

One install script handles every supported agent. By default it auto-detects existing agent directories (`~/.codex`, `~/.claude`, `~/.gemini`) and installs the skill package into each. It also installs the CLI to `~/.local/bin` when a GitHub release is available.

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

## Usage

Run commands from the repository root you want to log.

```bash
ntts-flightlog auto
ntts-flightlog mode ralph "Ralph 단일-owner 검증 루프"
ntts-flightlog turn-start "배포 실패 원인 조사" \
  --intent "재현 가능한 원인 찾기" \
  --constraints "프로덕션 변경 금지,로그 기반 판단" \
  --done-when "원인과 검증 명령이 기록됨"
ntts-flightlog decision "API만 재배포" "worker 변경 없음"
ntts-flightlog evidence "pytest 통과" "73개 테스트 통과"
ntts-flightlog turn-end "배포 smoke 검증 완료"
```

## Commands

```text
ntts-flightlog auto [title]
ntts-flightlog --lane worker-a turn-start <title> [--parent-turn id]
ntts-flightlog start [title]
ntts-flightlog stop
ntts-flightlog status <label> [focus] [next]
ntts-flightlog mode <solo|ralph|team|plan|review|autopilot|other> [detail]
ntts-flightlog turn-start <title> [--intent text] [--constraints a,b] [--done-when text]
ntts-flightlog turn-end [summary]
ntts-flightlog entry <title> [detail]
ntts-flightlog decision <title> [detail]
ntts-flightlog evidence <title> [detail] [--link decision-id-or-title]
ntts-flightlog evidence-check [--strict] [--format text|json]
ntts-flightlog evidence-report --persona self-retro|agent-operator|team-share [--format text|json]
ntts-flightlog blocker <title> [detail]
ntts-flightlog blocker-resolve <id-or-title> [resolution]
ntts-flightlog ingest --source <agent> --event <name> < event.json
ntts-flightlog hooks print --agent codex|claude|gemini
ntts-flightlog hooks doctor [--agent codex|claude|gemini] [--format text|json]
ntts-flightlog handoff [--format text|md|json]
ntts-flightlog attention [--format text|json] [--window day|week|all] [--agent name]
ntts-flightlog share [--format md|json] [--window day|week|all]
ntts-flightlog report [--format text|json] [--window day|week|all] [--agent name]
ntts-flightlog agent-stats [--format text|json] [--window day|week|all] [--agent name]
ntts-flightlog doctor
ntts-flightlog refresh-anchor [turn_id]
ntts-flightlog drift-check [turn_id]
ntts-flightlog path                                  # absolute path of main worklog file
ntts-flightlog turn-path [N]                         # absolute path of turn N (default: latest)
ntts-flightlog view <flat|turns|decisions|blockers|report|visual|tui>
ntts-flightlog migrate
ntts-flightlog self-upgrade
```

Use `--lane <name>` on write commands when multiple subagents or team workers
are logging in parallel. Each lane keeps its own active turn pointer so
`worker-a` can end its current turn without closing `worker-b`'s turn.

`ingest` reads one hook/event JSON object from stdin, redacts it before storage,
stores only bounded audit fields in `agent_events`, deduplicates by
`dedupe_key`, and promotes test-pass/test-fail events into reviewable
evidence/blocker candidates.

`hooks print` emits copyable hook starter commands only; it never mutates global
agent config. `hooks doctor` checks that the local binary, worklog directory,
and redacted ingest path are reachable.

`evidence-check` is the read-only Phase E readiness gate. Advisory mode reports
missing artifacts and placeholders; `--strict` returns non-zero while GA evidence
is incomplete. `evidence-report` prints the concrete persona-specific gap and
next action.

## Metrics

`ntts-flightlog report` computes five local metrics:

1. turn duration
2. blocker accumulation
3. agent completion rate
4. agent blocker frequency
5. evidence-bound decision ratio

Use `ntts-flightlog agent-stats` to inspect Phase E agent attribution health, including `auto_detect_correct_rate`, `auto_detect_unknown_rate`, `auto_detect_mismatch_rate`, and `override_rate`.

Use `ntts-flightlog attention` to turn metric and state signals into operator
actions: stale blockers, decisions without evidence, active turns without
evidence, drift alerts, long turns without outcomes, and agent attribution
warnings. Add `--format json` for a stable machine-readable schema.

## Views and clickable turns

Each `turn-start` creates `.ntts-flightlog/turns/turn-{N}.md` and mirrors every subsequent `entry`/`decision`/`evidence`/`blocker`/`turn-end` into it. That gives every main task its own scannable history alongside the flat main log.

Inside the tmux pane, the top bar is a 2-line menu — press `1`–`6` to switch views, `r` to reload, `q` to quit:

```text
[1]평면 [2]턴별 [3]결정 [4]블로커 [5]리포트 [6]시각화
[r]새로고침 [q]종료
```

- `1` flat — chronological live log.
- `2` turns — compact turn index with status, elapsed time, signal counts, and the latest result.
- `3` decisions — ADR-lite decision log with turn context and linked/same-turn evidence counts.
- `4` blockers — open-risk board with open blockers first and resolved blockers below.
- `5` report — operational summary of work volume, attention items, turn progress, decision evidence coverage, and blocker state.
- `6` visual — compact ASCII progress bars for turn completion, decision evidence, blocker resolution, entry mix, and lane distribution.

`evidence --link` accepts either a decision entry ID or a unique decision title
fragment. Ambiguous matches fail instead of guessing. `blocker-resolve` accepts a
blocker ID, entry ID, exact title, or unique title fragment and records the
resolution note for the blockers view.

Use `ntts-flightlog handoff` before switching agents or restarting a session. It
prints a compact handoff packet with current status, active turn anchors, open
blockers, decisions that still need evidence, latest evidence, and a recommended
next action.

Use `ntts-flightlog share --window week --format md` when the status needs to
leave the pane for a PR, issue, email, or Phase E team-share artifact. It
includes completed turns, active blockers, decisions/evidence, metric
highlights, and requested review/help.

For parallel work, add `--lane worker-a` to `turn-start`, `entry`, `decision`,
`evidence`, `blocker`, `turn-end`, `handoff`, `refresh-anchor`, and
`drift-check`. Lane labels appear in JSON handoff/share output and the report
view's lane summary when lane metadata exists.

Turn-start titles in the pane are OSC 8 hyperlinks. cmd/ctrl-click in iTerm2, WezTerm, Kitty, Ghostty, or the VS Code integrated terminal opens the corresponding `turn-{N}.md` in your default editor. Outside the pane, use `ntts-flightlog turn-path N` to get the path directly.

## Environment

```text
WORKLOG_DIR       default: .ntts-flightlog
WORKLOG_FILE      default: ${WORKLOG_DIR}/main.md
REFRESH_SECONDS   default: 2
PANE_PERCENT      default: 34
```

## Distribution Layout

```
skill/ntts-flightlog/     # Agent skill package (shared by Codex, Claude Code, Gemini)
  SKILL.md                # Agent-facing description
  references/             # Design notes
  agents/openai.yaml      # Codex-specific UI metadata (ignored by other agents)
cmd/flightlog/            # Go CLI entrypoint
internal/                 # CLI, DB, TUI, metrics, migration, and agent detection packages
e2e/                      # End-to-end tests
testdata/                 # Fixtures and golden files
scripts/install.sh        # Multi-agent installer and release downloader
scripts/install-from-github.sh  # curl|bash bootstrap
```

## Development

```bash
go test ./...
go test ./... -race -count=1
go test ./e2e -tags=e2e -count=1
go test ./e2e -tags='e2e tmux' -run TestTmux -count=1
scripts/build-local.sh   # writes dist/local-current/ntts-flightlog
```

Use `dist/local-current/ntts-flightlog` for local smoke tests. Release and CI
artifacts may still use `dist/flightlog`; the explicit local folder prevents
stale release-style binaries from being confused with the current dev build.

For local 3-agent sanity checks:

```bash
scripts/sanity_3_agents_tmux.sh
```

The check writes `docs/e0-3-agent-tmux-sanity.md` and verifies `claude`, `codex`, and `gemini` inside tmux panes when those CLIs are installed.

For Phase E evidence readiness:

```bash
scripts/phase_e_readiness.sh          # advisory while evidence is being collected
scripts/phase_e_readiness.sh --strict # GA-blocking readiness check
ntts-flightlog evidence-check --strict
ntts-flightlog evidence-report --persona team-share
```

Use `docs/phase-e-ga-readiness-roadmap.md` for the remaining GA-blocking
workstreams and stop conditions. Use `docs/metric-interpretation-guide.md` when
writing or reviewing metric-based evidence.

## Privacy and Scope

All state is written under `.ntts-flightlog/` in the repository where you run the CLI. Metrics stay local in SQLite. Generated worklogs, runtime state, build artifacts, and coverage files are intentionally ignored by git.

## License

MIT
