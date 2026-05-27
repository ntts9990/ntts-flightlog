# E0 3-Agent Explicit Attachment Rehearsal

Generated: 2026-05-27T08:01:19Z

- repository: /Users/sungyub/Documents/Projects/ntts-flightlog
- binary: local build from `./cmd/flightlog`
- worklog scope: temporary local worklog; artifact below preserves command evidence

## Result

| Agent | auto | turn-start | entry | evidence | turn-end | handoff | stop |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `codex` | pass | pass | pass | pass | pass | pass | pass |
| `claude` | pass | pass | pass | pass | pass | pass | pass |
| `gemini` | pass | pass | pass | pass | pass | pass | pass |

## Agent Stats

```text
NTTS Flightlog agent-stats [window=all] [agent=전체]
auto_detect_correct_rate: 0.0% (0/3 sessions)
auto_detect_unknown_rate: 100.0% (3/3 sessions)
auto_detect_mismatch_rate: 0.0% (0/3 sessions)
override_rate: 100.0% (3/3 sessions)

claude     sessions=1 turns=1 complete=1 completion=100.0% blockers=0 blocker_freq=0.000
codex      sessions=1 turns=1 complete=1 completion=100.0% blockers=0 blocker_freq=0.000
gemini     sessions=1 turns=1 complete=1 completion=100.0% blockers=0 blocker_freq=0.000
```

## Command Evidence

### codex

```text
$ ntts-flightlog --agent codex auto "Phase E explicit codex rehearsal"
작업 기록 파일: /var/folders/3x/z31z27yn167_hb7bjcc8wqh40000gn/T/flightlog-agent-rehearsal.TZYEcA/worklog/main.md
현재 shell에서는 tmux side pane 렌더링을 사용할 수 없습니다.
$ ntts-flightlog --agent codex turn-start "codex attachment rehearsal" --intent ... --constraints ... --done-when ...
⚓ 의도: explicit --agent attribution rehearsal for codex
📐 제약: no external evidence fabrication | local temp worklog only
✅ 완료조건: auto turn-start entry evidence turn-end handoff all succeed
$ ntts-flightlog --agent codex entry "codex entry smoke"
$ ntts-flightlog --agent codex evidence "codex evidence smoke"
$ ntts-flightlog --agent codex turn-end "codex rehearsal complete"
$ ntts-flightlog --agent codex handoff --format md
# NTTS Flightlog Handoff

생성: 2026-05-27T08:01:19Z
작업로그: /var/folders/3x/z31z27yn167_hb7bjcc8wqh40000gn/T/flightlog-agent-rehearsal.TZYEcA/worklog
상태: 대기
모드: solo
초점: 마지막 턴 1 완료: 0s.
다음: 다음 작업 턴을 기다립니다.
세션 경과: 0s

현재 턴
- turn-1: codex attachment rehearsal
- 상태/경과: complete / 0s
- 의도: explicit --agent attribution rehearsal for codex
- 제약: no external evidence fabrication, local temp worklog only
- 완료조건: auto turn-start entry evidence turn-end handoff all succeed
- 마지막 결과: codex rehearsal complete

열린 블로커
- 없음

근거 없는 결정
- 없음

최근 근거
- turn-1 b41f8866d…: codex evidence smoke

추천 다음 행동
- 완료조건을 기준으로 검증한 뒤 turn-end에 결과를 남기세요.
$ ntts-flightlog --agent codex stop
작업 기록 pane 중지됨.
```

### claude

```text
$ ntts-flightlog --agent claude auto "Phase E explicit claude rehearsal"
작업 기록 파일: /var/folders/3x/z31z27yn167_hb7bjcc8wqh40000gn/T/flightlog-agent-rehearsal.TZYEcA/worklog/main.md
현재 shell에서는 tmux side pane 렌더링을 사용할 수 없습니다.
$ ntts-flightlog --agent claude turn-start "claude attachment rehearsal" --intent ... --constraints ... --done-when ...
⚓ 의도: explicit --agent attribution rehearsal for claude
📐 제약: no external evidence fabrication | local temp worklog only
✅ 완료조건: auto turn-start entry evidence turn-end handoff all succeed
$ ntts-flightlog --agent claude entry "claude entry smoke"
$ ntts-flightlog --agent claude evidence "claude evidence smoke"
$ ntts-flightlog --agent claude turn-end "claude rehearsal complete"
$ ntts-flightlog --agent claude handoff --format md
# NTTS Flightlog Handoff

생성: 2026-05-27T08:01:19Z
작업로그: /var/folders/3x/z31z27yn167_hb7bjcc8wqh40000gn/T/flightlog-agent-rehearsal.TZYEcA/worklog
상태: 대기
모드: solo
초점: 마지막 턴 2 완료: 0s.
다음: 다음 작업 턴을 기다립니다.
세션 경과: 0s

현재 턴
- turn-2: claude attachment rehearsal
- 상태/경과: complete / 0s
- 의도: explicit --agent attribution rehearsal for claude
- 제약: no external evidence fabrication, local temp worklog only
- 완료조건: auto turn-start entry evidence turn-end handoff all succeed
- 마지막 결과: claude rehearsal complete

열린 블로커
- 없음

근거 없는 결정
- 없음

최근 근거
- turn-2 f88d99e87…: claude evidence smoke

추천 다음 행동
- 완료조건을 기준으로 검증한 뒤 turn-end에 결과를 남기세요.
$ ntts-flightlog --agent claude stop
작업 기록 pane 중지됨.
```

### gemini

```text
$ ntts-flightlog --agent gemini auto "Phase E explicit gemini rehearsal"
작업 기록 파일: /var/folders/3x/z31z27yn167_hb7bjcc8wqh40000gn/T/flightlog-agent-rehearsal.TZYEcA/worklog/main.md
현재 shell에서는 tmux side pane 렌더링을 사용할 수 없습니다.
$ ntts-flightlog --agent gemini turn-start "gemini attachment rehearsal" --intent ... --constraints ... --done-when ...
⚓ 의도: explicit --agent attribution rehearsal for gemini
📐 제약: no external evidence fabrication | local temp worklog only
✅ 완료조건: auto turn-start entry evidence turn-end handoff all succeed
$ ntts-flightlog --agent gemini entry "gemini entry smoke"
$ ntts-flightlog --agent gemini evidence "gemini evidence smoke"
$ ntts-flightlog --agent gemini turn-end "gemini rehearsal complete"
$ ntts-flightlog --agent gemini handoff --format md
# NTTS Flightlog Handoff

생성: 2026-05-27T08:01:19Z
작업로그: /var/folders/3x/z31z27yn167_hb7bjcc8wqh40000gn/T/flightlog-agent-rehearsal.TZYEcA/worklog
상태: 대기
모드: solo
초점: 마지막 턴 3 완료: 0s.
다음: 다음 작업 턴을 기다립니다.
세션 경과: 0s

현재 턴
- turn-3: gemini attachment rehearsal
- 상태/경과: complete / 0s
- 의도: explicit --agent attribution rehearsal for gemini
- 제약: no external evidence fabrication, local temp worklog only
- 완료조건: auto turn-start entry evidence turn-end handoff all succeed
- 마지막 결과: gemini rehearsal complete

열린 블로커
- 없음

근거 없는 결정
- 없음

최근 근거
- turn-3 68dd0b522…: gemini evidence smoke

추천 다음 행동
- 완료조건을 기준으로 검증한 뒤 turn-end에 결과를 남기세요.
$ ntts-flightlog --agent gemini stop
작업 기록 pane 중지됨.
```

## Interpretation

- This proves the local CLI can run a complete Flightlog workflow for Codex,
  Claude, and Gemini using explicit `--agent` overrides.
- This does not prove native hook installation or real agent hook firing.
- `agent-stats` separates auto-detection health from override adoption; explicit
  override evidence should not be used to rank agents until native attribution is
  reliable.
