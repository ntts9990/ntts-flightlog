# Deep Interview Spec: NTTS Flightlog v2 Roadmap (3~6개월)

## Metadata
- Interview ID: di-v2-roadmap-2026-05-20
- Rounds: 9 (Round 0 topology + Round 1~9 ambiguity)
- Final Ambiguity Score: 19.3%
- Type: brownfield
- Generated: 2026-05-20
- Threshold: 20% (resolved)
- Status: PASSED
- Initial Context Summarized: no
- Challenge Modes Used: contrarian (R4), simplifier (R6)
- Deliverable Scope: **planning document only** (no execution; Phase 5 handoff is omc-plan refinement or stop)

## Clarity Breakdown

| Dimension | Score | Weight | Weighted |
|-----------|-------|--------|----------|
| Goal Clarity | 0.85 | 0.35 | 0.298 |
| Constraint Clarity | 0.89 | 0.25 | 0.223 |
| Success Criteria | 0.81 | 0.25 | 0.203 |
| Context Clarity (brownfield) | 0.66 | 0.15 | 0.099 |
| **Total Clarity** | | | **0.823** |
| **Ambiguity** | | | **0.193** |

### Per-component Ambiguity

| Component | Ambiguity | Status |
|---|---|---|
| data-sync | 0.17 | ≤ threshold |
| distribution | 0.20 | ≤ threshold |
| test-quality | 0.20 | ≤ threshold |
| core-cli | 0.20 | ≤ threshold |
| multi-agent | 0.21 | just above (1pp) |

## Topology

| Component | Status | Description | Coverage / Deferral Note |
|---|---|---|---|
| **Core CLI & Renderer** | active | Go 기반 단일 정적 바이너리. Bubble Tea TUI, tmux pane 통합, 5 view 렌더러, 카테고리 색상·OSC 8 hyperlink 유지 | R1 Goal · R3 Constraints · R9 Constraints |
| **Multi-Agent Integration** | active | agent ID(Claude/Codex/Gemini/…) 자동감지(env heuristic + ps tree) + `--agent` override, agent별 메트릭 2축(완료율·blocker 빈도) | R5 Goal · R7 Constraints |
| **Distribution & Upgrade** | active | GoReleaser → GitHub Releases + Homebrew tap + scoop bucket + AUR + Docker. install.sh는 Go 바이너리 다운로드 wrapper로 단순화. v1 → v2 migration command 포함 | R6 Goal |
| **Data & Sync** | active | main.md + SQLite hybrid. SQLite 스키마: `sessions`, `turns`, `entries`, `decisions↔evidence` 관계, `agent_id` 일급 컬럼. JSON/markdown export. v1 main.md → v2 SQLite migrator | R2 Goal · R5 implied |
| **Test & Quality** | active | Go test (unit + 통합 + sub-command E2E) + golden output snapshot + P0 24건 + S8/S9 자동화 + GitHub Actions CI matrix (macOS arm64/x64 + Linux x64/arm64 + Windows x64). 회고 GA 게이트 X≥4. | R8 Goal+Criteria |

**Deferred**:
- **MCP server mode** (multi-agent component 내) → v2.1+. 사유: hybrid auto-detect + override가 v2.0 충분, MCP는 침투적·구현비 높음 (R7 confirmed_at 2026-05-20)
- **실시간 자가조정/intervention** (cross-cutting) → v3+. 사유: persona 3(실시간) 페르소나는 R2에서 명시적 제외

## Goal

NTTS Flightlog v2는 **단일 정적 Go 바이너리**로서 v1의 라이브 작업 기록 능력을 유지하면서, **3 페르소나(본인 회고 · agent 운영자 · 팀 외부 공유)** 모두가 인용할 수 있는 **5개 핵심 메트릭**을 SQLite에 누적·집계·export 한다. v1으로는 절대 할 수 없던 제1 행동은 **여러 세션·agent를 가로질러 작업 진행 패턴을 사후 분석**하는 것이다.

### 5 Core Metrics (v2 thesis embodied)

1. **turn 소요시간 분포** — turn별 elapsed, 일/주 집계 (data-sync: `turns.elapsed_ms`)
2. **blocker 누적시간** — blocker 발생 → 해소까지 누적 (data-sync: `blockers.opened_at/closed_at`)
3. **agent별 turn 완료율** — `complete` / `(complete + abort + abandon)` per `agent_id`
4. **agent별 blocker 빈도** — `count(blockers)` / `count(turns)` per `agent_id`
5. **evidence가 붙은 decision 비율** — `count(decisions WHERE has linked evidence)` / `count(decisions)` (data-sync: `decisions↔evidence` link table)

## Constraints

### Functional
- **Runtime**: Go 1.22+ (single static binary). bash + awk + tmux-only 원칙 폐기.
- **TUI**: Bubble Tea + Lipgloss + Bubbles. v1 카테고리 색상·8-turn 순환·OSC 8 hyperlink 1:1 이전.
- **DB**: 내장 SQLite via `modernc.org/sqlite` (CGo-free). DB 파일: `.ntts-flightlog/flightlog.db`. main.md는 export view로 유지(read-only mirror).
- **agent ID 캡처**: env heuristic 자동감지(`CLAUDE_DESKTOP_VERSION`, `CODEX_HOME`, `GEMINI_API_KEY`, parent process tree) + `--agent <name>` override flag. 감지 실패 시 `unknown`. **MCP server 모드는 v2.1+로 보류.**
- **Backward compat**: v1 `.ntts-flightlog/main.md`(+`turns/`)을 lossless로 SQLite에 import하는 `flightlog migrate` 명령. v1 디렉터리 보존 (parallel run 가능).
- **tmux 의존**: viewer는 tmux 옵션 (없으면 file-only mode). tmux interop은 `os/exec`로 외부 binary 호출.

### Non-Functional
- **OS 매트릭스**: macOS arm64/x64, Linux x64/arm64, Windows x64 (5 build targets).
- **바이너리 크기**: ≤ 25MB (`-ldflags "-s -w"` 적용 후).
- **Cold start**: < 100ms for `flightlog entry/decision/...` (인터랙티브 사용감 유지).
- **Schema migration**: append-only 마이그레이션, `flightlog migrate up/down` 지원.
- **Telemetry**: 외부 송신 0. 모든 메트릭은 로컬 SQLite에만 저장.

### Distribution
- **GoReleaser**: 단일 `.goreleaser.yml`로 GitHub Releases + Homebrew tap (`ntts9990/homebrew-tap`) + scoop bucket + AUR + Docker image 자동.
- **install.sh**: Go 바이너리 download wrapper로 단순화 (`curl | sh` 한 줄 유지). `--codex/--claude/--gemini/--all/--no-cli` 플래그는 *agent skill 디렉토리 설치만* 담당.
- **Upgrade**: `flightlog self-upgrade` (GitHub releases API 폴링) 또는 Homebrew 표준 `brew upgrade`.

## Non-Goals

- ❌ **MCP server protocol 일급 지원** (v2.1+로 보류)
- ❌ **실시간 자가조정/intervention** (alarm, auto-suspend, auto-summarize) — v3+
- ❌ **외부 서비스 동기화** (Linear, Notion, Slack, Datadog) — v2는 export까지만, integration은 v2.1+
- ❌ **웹 대시보드** — v2는 CLI/TUI only. 브라우저 view는 v3+
- ❌ **bash 호환 유지** — v2는 완전 재작성. v1.x maintenance는 별도 branch
- ❌ **다중 사용자/팀 동시 편집** — single-user local DB. 팀 공유는 export 기반
- ❌ **`fswatch` 통합** — v1에서 이미 mtime polling으로 결론. v2도 동일 패턴 또는 `fsnotify`(Go 표준 패턴) 사용

## Acceptance Criteria

### Functional (Auto-tested)
- [ ] `flightlog start/stop/auto`이 tmux pane 띄우고 종료까지 메뉴 헤더 + 5 view 전환 (1/2/3/4/q 키 + 신규 `5` = report view) 정상 작동
- [ ] `flightlog entry/decision/evidence/blocker <title> [detail]`이 SQLite + main.md mirror에 동시 기록
- [ ] `flightlog turn-start/turn-end`이 turn 카운터 + elapsed 계산 + per-turn markdown 생성 + OSC 8 hyperlink 유지
- [ ] `flightlog mode <mode> [--agent <name>]`이 agent ID + mode를 sessions table에 기록
- [ ] `flightlog report` 명령이 5 core metric을 텍스트/JSON 두 포맷으로 출력
- [ ] `flightlog migrate` 명령이 v1 `.ntts-flightlog/main.md` + `turns/` → SQLite로 lossless 변환 (라운드트립 테스트로 검증)
- [ ] agent detection이 Claude Desktop / Codex / Gemini CLI 환경에서 각각 정확히 식별 (auto-detection unit test)
- [ ] `--agent` override flag가 자동감지값을 덮어쓰며 audit log에 둘 다 보존
- [ ] SQLite 스키마: `sessions`, `turns`, `entries`, `blockers`, `decisions`, `evidence`, `decision_evidence_links` 7개 테이블 + index
- [ ] JSON export: 5 메트릭 + 원본 entries + agent 메타데이터 포함

### Non-Functional (Auto-tested)
- [ ] Cold start (`flightlog entry "..."`): < 100ms on M1 MacBook + warm filesystem cache
- [ ] 바이너리 크기 5 target 모두 ≤ 25MB (`-ldflags "-s -w"` 후)
- [ ] CI matrix: macOS arm64/x64 + Linux x64/arm64 + Windows x64 (5 jobs) 모두 PASS per PR
- [ ] golden output snapshot diff: 5 메트릭 SQL/JSON 출력이 fixture와 byte-equal
- [ ] P0 시나리오 24건 (S1-S24) + S8/S9 자동화 모두 PASS
- [ ] 5 OS×arch 바이너리 download → 실행 → `flightlog --version` smoke 통과

### Retrospective GA Gate (X ≥ 4)
- [ ] **3 페르소나 모두에서 회고가 5 메트릭 중 ≥ 4개를 자발 인용**해야 GA 통과 (옵션 A "엄격" 채택)
  - 본인 회고: 4주간 v2 alpha 사용 후 일지에서 4개 메트릭 자발 인용 + 결정 변경 ≥ 1건
  - agent 운영자 페르소나(본인 또는 외부 1명): agent 비교 결정(예: "X 작업은 Codex가 더 빠르다") 인용 시 메트릭 ≥ 4개 근거
  - 팀 외부 공유: weekly report 작성 시 4개 메트릭 자발 포함
- [ ] 회고 인용도 미달 시 v2.0 GA 차단 → 메트릭/렌더 재설계 후 재평가
- [ ] **agent 메트릭 unknown 비율 < 10%** (auto-detection 신뢰도 확보)

## Assumptions Exposed & Resolved

| Round | Assumption | Challenge | Resolution |
|---|---|---|---|
| R1 | "v2 = v1 + 기능"이 자명 | 어떤 *제1 행동*이 새롭게 가능해지는가? | v2 thesis = **오프라인 후행 분석/메트릭** (실시간 intervention 제외) |
| R2 | analytics는 한 종류의 사용자가 본다 | 페르소나 4종을 제시 — 누가 무엇을 결정하는가? | **3 페르소나**: self-retro + agent-operator + team-share. 실시간 자가조정 제외 |
| R3 | bash 원칙은 v2에도 유지 (default 가정) | bash로 분석엔진이 가능한가? | **bash 폐기, 단일 정적 바이너리** — 모든 후속 결정 cascade |
| R4 (Contrarian) | "출시했으니 GA"가 자명 | 외면당하는 시나리오를 가정하면? | **회고-기반 acceptance** (telemetry 불요, 5 메트릭 인용 카운트) |
| R5 | "메트릭은 직관으로 정의됨" | 5개를 *명시적으로* 정하면? | **Hybrid 5**: turn 시간 + blocker 누적 + agent 완료율 + agent blocker 빈도 + evidence-bound decision 비율 |
| R6 (Simplifier) | 6개월 스코프 = 5 컴포넌트 다 GA | MVP cut으로 3개월 단축 가능한가? | **fullstack 유지** — MVP cut 거부. 6개월 단일 마일스톤 |
| R7 | agent ID는 사용자가 매번 입력 | 자동 vs 수동 vs MCP — 어느 쪽? | **Hybrid 자동감지 + `--agent` override**. MCP는 v2.1+ |
| R8 | "충분한 테스트면 됨" | 자동 회귀 범위 + 회고 임계 *동시 결정* | **엄격 풀자동 + X≥4**. CI matrix 5개, golden snapshot, P0/S8/S9 자동화 |
| R9 | "단일 바이너리" 만으로 충분 | Go vs Rust vs Zig — 어느 쪽? | **Go** (9/10 평가 축 우세, Bubble Tea + modernc.org/sqlite + GoReleaser) |

## Technical Context (Brownfield)

### v1 코드베이스 매핑 (explore agent 결과)

| 영역 | 위치 | 라인 | v2 이전 전략 |
|---|---|---|---|
| CLI 진입점 | `bin/ntts-flightlog` | 686 | Go `cobra` CLI로 재작성. 서브커맨드 12개 유지 + `report` / `migrate` / `self-upgrade` 신규 |
| ANSI 렌더러 | `bin/ntts-flightlog:309-410` (awk) | 100+ | Bubble Tea + Lipgloss model로 재작성. 카테고리 색상 + 8-turn 순환 + OSC 8 함수 단위로 매핑 |
| Pane viewer | `bin/ntts-flightlog:483-570` (heredoc) | 87 | Bubble Tea program (별도 `flightlog view` 서브커맨드 또는 동일 `start`의 자식 process) |
| mtime polling | `get_mtime()` | - | `fsnotify` 또는 동일 mtime polling 유지 (SQLite는 `?cache=shared` + signal로 알림 가능) |
| Skill metadata | `skill/ntts-flightlog/SKILL.md` | 113 | 그대로 유지 (Codex/Claude/Gemini skill 디렉토리에 SKILL.md만 설치) |
| 설치 스크립트 | `scripts/install.sh` + `install-from-github.sh` | 100+ | 단순화 — Go 바이너리 download wrapper로. agent skill 디렉토리 설치는 유지 |
| State files | `.ntts-flightlog/` (main.md, turns/, mode, turn-counter, session-start-epoch, turn-start-epoch, pane-id) | - | `flightlog.db` 신규 + main.md를 export view로 mirror. turns/, mode, turn-counter, *-epoch는 SQLite로 흡수 |

### 핵심 의존성 (Go ecosystem)
- **CLI**: `github.com/spf13/cobra` + `github.com/spf13/viper` (config)
- **TUI**: `github.com/charmbracelet/bubbletea` + `github.com/charmbracelet/lipgloss` + `github.com/charmbracelet/bubbles`
- **SQLite**: `modernc.org/sqlite` (CGo-free)
- **Migration**: `github.com/golang-migrate/migrate/v4` 또는 자체 구현 (간단한 SQL 파일)
- **Logging**: `log/slog` (Go 1.21+ 표준)
- **Test**: `testing` + `testify` + golden snapshot은 `github.com/sebdah/goldie`
- **Release**: GoReleaser
- **CI**: GitHub Actions matrix

### v1 → v2 마이그레이션 데이터 매핑

```
.ntts-flightlog/main.md (Korean headings) ─┐
.ntts-flightlog/turns/turn-*.md            ├─→ flightlog.db
.ntts-flightlog/mode                       │   (sessions, turns, entries, blockers,
.ntts-flightlog/turn-counter               │    decisions, evidence, links)
.ntts-flightlog/session-start-epoch        │
.ntts-flightlog/turn-start-epoch           │
.ntts-flightlog/pane-id                    ─┘ (pane-id는 휘발 정보, 마이그레이션 제외)
```

라운드트립 테스트: SQLite → markdown export → re-import → SQLite 동등성 검증.

## Ontology (Key Entities)

| Entity | Type | Fields | Relationships |
|---|---|---|---|
| `Session` | core | id, started_at, ended_at, mode (solo/ralph/team/plan/review/autopilot/other), agent_id, agent_detected (자동감지값), agent_override (사용자 명시값), title, focus, next_step | has many `Turn`, has many `Entry` |
| `Turn` | core | id, session_id, sequence_no, title, started_at, ended_at, status (complete/abort/abandon), elapsed_ms, agent_id | belongs_to `Session`, has many `Entry`, has many `Blocker` |
| `Entry` | core | id, session_id, turn_id (nullable), kind (entry/decision/evidence/blocker/mode), title, detail, created_at, agent_id | belongs_to `Session`, optionally `Turn` |
| `Decision` | core | (subtype of Entry where kind=decision) | has_many `Evidence` via `decision_evidence_links` |
| `Evidence` | core | (subtype of Entry where kind=evidence) | optionally linked to `Decision`(s) |
| `Blocker` | core | id, turn_id, entry_id (origin), title, opened_at, closed_at, status (open/resolved/abandoned), accumulated_seconds | belongs_to `Turn` |
| `Agent` | supporting | id, name (claude/codex/gemini/...), detection_signals (env vars, parent process), first_seen, last_seen | referenced by `Session`, `Turn`, `Entry` (via agent_id string) |
| `MetricSnapshot` | supporting | id, computed_at, window_start, window_end, scope (session/day/week), payload (5 메트릭 JSON) | optional cache, recompute-on-demand 가능 |
| `decision_evidence_link` | relation | decision_entry_id, evidence_entry_id, created_at, note | joins `Decision` and `Evidence` |

## Ontology Convergence

| Round | Entity Count | New | Changed | Stable | Stability Ratio |
|---|---|---|---|---|---|
| 1 | 3 | 3 (Session, Turn, Entry) | - | - | N/A |
| 2 | 5 | +2 (Persona, Metric) | - | 3 | 60% |
| 3 | 5 | 0 | - | 5 | 100% |
| 4 | 6 | +1 (AcceptanceTest) | - | 5 | 83% |
| 5 | 8 | +3 (Decision, Evidence, Blocker as first-class) | - | 5 | 63% |
| 6 | 8 | 0 | - | 8 | 100% |
| 7 | 9 | +1 (Agent first-class) | - | 8 | 89% |
| 8 | 9 | 0 | - | 9 | 100% |
| 9 | 9 | 0 | - | 9 | 100% |

→ **Round 6 이후 ontology 수렴 안정** (마지막 4 round 평균 stability 97%).

## v2 Roadmap (3~6개월 단계별)

### Phase A: Foundation (Month 1)
1. **Go module 부트스트랩**: `go mod init`, cobra 골격, 7-table SQLite 스키마 + migration runner
2. **SQLite 데이터 계층**: sessions/turns/entries/blockers/decisions/evidence/links + index. modernc.org/sqlite 통합
3. **agent 자동감지 모듈**: env heuristic + parent process tree, unit tests for Claude/Codex/Gemini fixture
4. **CLI 서브커맨드 12개**: start/stop/auto/status/mode/turn-start/turn-end/entry/decision/evidence/blocker/path/turn-path/view (v1 동일 surface 보존)
5. **`flightlog migrate` 명령**: v1 `.ntts-flightlog/` → SQLite lossless. round-trip test.

**Exit criterion (월말)**:
- v1 사용자가 `flightlog migrate` 한 줄로 SQLite로 이전 가능
- 모든 v1 서브커맨드가 SQLite 백엔드 위에서 동일 출력 (golden snapshot test)

### Phase B: Renderer + Metrics (Month 2)
6. **Bubble Tea TUI**: 메뉴 헤더 + 4 view (flat/turns/decisions/blockers) + 신규 5번째 `report` view
7. **카테고리/turn 색상 시스템**: v1 awk 매핑을 Lipgloss styles로 1:1 이식. OSC 8 hyperlink 유지.
8. **5 core metrics SQL**: turn 시간 / blocker 누적 / agent 완료율 / agent blocker 빈도 / evidence-bound decision 비율 — 각각 SQL view + Go query function
9. **`flightlog report` 명령**: 텍스트/JSON 두 포맷, --window day/week/all, --agent filter

**Exit criterion (월말)**:
- v2가 v1의 모든 시각 표현을 동일하게 렌더 (디자인 snapshot 일치)
- 5 메트릭 모두 sample 데이터에 대해 정확한 숫자 산출 (unit test)

### Phase C: Distribution + CI (Month 3)
10. **GoReleaser 통합**: `.goreleaser.yml`, GitHub Releases + Homebrew tap + scoop bucket + AUR 자동
11. **GitHub Actions matrix CI**: macOS arm64/x64 + Linux x64/arm64 + Windows x64, PR마다 5 job
12. **install.sh 단순화**: Go 바이너리 download wrapper. `--codex/--claude/--gemini/--all/--no-cli`은 skill SKILL.md 설치 전용으로 축소
13. **`flightlog self-upgrade`**: GitHub releases API 폴링, 안전한 atomic replace

**Exit criterion (월말)**:
- `brew install ntts9990/tap/flightlog`로 macOS 양 arch 모두 설치 가능
- scoop bucket으로 Windows 설치 가능
- CI 5 job 전부 main에서 green

### Phase D: Test Hardening + Acceptance (Month 4)
14. **Test suite 확장**: unit + 통합 + sub-command E2E + golden output snapshot
15. **P0 24건 자동화**: v1 수동 회귀 시나리오를 Go test로 이식. S8/S9 포함
16. **Property/contract tests**: agent_id format, SQLite migration round-trip, 5 metric SQL invariant
17. **Alpha 배포**: 본인 dogfooding 시작 (4주 일지 작성)

**Exit criterion (월말)**:
- CI 모든 자동 test 항목 통과
- Alpha 본인 dogfooding 첫 주 완료

### Phase E: Retrospective Gate + Polish (Month 5)
18. **Alpha dogfooding 지속**: 4주 일지 + 결정 변경 ≥ 1건 추적
19. **agent-operator 페르소나 검증**: 외부 1명 또는 본인이 agent 비교 결정 1건 인용
20. **Team-share 페르소나 검증**: weekly report 작성, 4 메트릭 자발 포함 확인
21. **Bug fixing + perf tuning**: cold start ≤ 100ms 달성, 바이너리 ≤ 25MB

**Exit criterion (월말)**:
- 3 페르소나 모두에서 X≥4 메트릭 자발 인용 충족
- agent unknown 비율 < 10%

### Phase F: GA Release + v2.1 Planning (Month 6)
22. **GA release**: v2.0.0 태그, GoReleaser release, Homebrew/scoop/AUR 동시 푸시
23. **Migration docs + 예제**: v1 → v2 가이드, 5 메트릭 해석 가이드
24. **v2.1 backlog**: MCP server, 외부 sync, 실시간 intervention 후보 정리

**Exit criterion (월말)**:
- v2.0.0 출시
- 회고 게이트 통과 evidence 문서화

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| 회고 게이트 X≥4 미달 → GA 차단 | M | High | Phase D부터 alpha dogfooding 시작, 임계 미달 시 메트릭 재설계 1개월 buffer |
| agent 자동감지 신뢰도 < 90% | M | M | hybrid override + audit log로 사용자가 후보정 가능, unknown 비율을 GA 게이트 추가 조건 |
| Bubble Tea 학습 곡선이 일정 위협 | L | M | Phase B 첫 주에 prototype spike (1 view), 막힐 경우 v1 ANSI awk 패턴 유지하고 TUI 일부만 도입 |
| GoReleaser 설정 복잡도 | L | M | Phase C에 1주 buffer. cobra/Bubble Tea/GoReleaser 모두 일등급 ecosystem |
| 단독 개발자 6개월 일정 지연 | M | High | Simplifier 거부했지만 Phase E~F 한 페이즈는 deferral 가능 (e.g., self-upgrade를 v2.0.1로) |
| modernc.org/sqlite 성능 (CGo-free) | L | M | benchmark 측정 후 만약 cold-start 100ms 초과하면 CGo `mattn/go-sqlite3`로 fallback (5 target 빌드 복잡도 ↑ 감수) |

## Interview Transcript

<details>
<summary>Full Q&A (9 rounds + Round 0 topology)</summary>

### Round 0 — Topology
**Q**: v2 로드맵을 5개 최상위 컴포넌트로 (Core CLI / Multi-Agent / Distribution / Data&Sync / Test&Quality) 잠가도 되나?
**A**: 5개 그대로 맞음.

### Round 1 — Component: Core CLI / Goal
**Q**: v2의 *제1 행동* (v1으로는 절대 할 수 없는)은?
**A**: 작업 진행 분석/메트릭 (option B).
**Ambiguity**: 69.5%

### Round 2 — Component: Data & Sync / Goal
**Q**: 메트릭의 1차 소비자는? 무엇을 결정하려고?
**A**: 1+2+4 (본인 회고 + agent 운영자 + 팀 외부 공유). 실시간 자가조정 제외.
**Ambiguity**: 63.4%

### Round 3 — Component: Core CLI / Constraints
**Q**: "셸 스크립트만" 원칙을 어디까지 유지?
**A**: 단일 바이너리로 진화 (option B).
**Ambiguity**: 51.4%

### Round 4 — Component: Data & Sync / Criteria | CONTRARIAN MODE
**Q**: v2 출시 후 외면당하는 시나리오를 가정해 acceptance criterion 1개를 역으로 도출?
**A**: 공식 사용자 회고를 쓰면 "5개 메트릭 중 X개 자발 인용" (option D).
**Ambiguity**: 43.2%

### Round 5 — Component: Multi-Agent / Goal
**Q**: 5 핵심 메트릭은 무엇?
**A**: Hybrid — 시간 2 + agent 2 + 증거 1 (option D).
**Ambiguity**: 35.5%

### Round 6 — Component: Distribution / Goal | SIMPLIFIER MODE
**Q**: v2 GA 최소 소집 vs v2.1+ deferral?
**A**: 풀스택 — 6개월 스코프 그대로 (option C).
**Ambiguity**: 33.3%

### Round 7 — Component: Multi-Agent / Constraints
**Q**: agent ID 캡처는?
**A**: Hybrid — 자동감지 + 명시 override (option D). MCP는 v2.1+ 보류.
**Ambiguity**: 29.7%

### Round 8 — Component: Test & Quality / Goal + Criteria
**Q**: 자동 회귀 범위 + 회고 게이트 X 임계?
**A**: 엄격 풀자동 + X≥4 (option A). 추가 신호: 단일 언어 사용 의지.
**Ambiguity**: 22.7%

### Round 9 — Component: Core CLI / Constraints (language)
**Q**: Go vs Rust vs Zig — 어느 하나?
**A**: (비교 분석 요청 후) **Go** 선택.
**Ambiguity**: 19.3% ✅ Threshold met

</details>

## Addendum — Phase A.5 (post-execution discovery, 2026-05-20)

**실제 사용 중 발견된 v2 thesis 보강**: 사용자가 다른 agent 세션에서 context drift로 1시간 작업을 손실. 이는 v1의 "tmux 가시화" 미션의 더 깊은 형태 — **agent가 context를 잃어도 원래 의도가 살아남는 anchor 메커니즘**. deep-interview R4 Contrarian Mode에서 충분히 깊게 다루지 못한 thesis가 실제 사용에서 표면화.

**Phase A.5 Turn Intent Anchor (TIA) 추가** — Phase A 완료 후 Phase B 전:
- 0002 migration: `turns` 테이블에 `intent TEXT`, `constraints_json TEXT`, `done_when TEXT`, `drift_alerts INTEGER DEFAULT 0`, `anchor_last_shown_at DATETIME` 5컬럼 추가
- `turn-start --intent ... --constraints ... --done-when ...` 플래그 확장
- 신규: `refresh-anchor [TURN_ID]`, `drift-check [TURN_ID]` 명령
- renderer anchor 표시 (view flat/turns) + entry 후 ⚓ reminder
- v2.0 범위: 키워드 매칭. NLP semantic drift는 v2.1+

**6번째 메트릭 후보**: drift_alerts/turn — 작업 안전성 지표. 회고 게이트 X≥4가 5/6으로 변경되거나 별도 안전성 게이트로 신설 가능 (Phase A.5 exit 이후 결정).

**용량 영향**: +1-2 dev-days. F2a (migration guide) slip 우선순위 ↑.

## Spec Status

**PENDING APPROVAL** — 이 spec은 deep-interview 단계의 산출물입니다. Phase A.5는 실행 중 발견되어 즉시 통합 결정됨 (사용자 선택). 다음 단계 옵션은 (1) `omc-plan --consensus --direct`로 Planner/Architect/Critic이 plan을 합의 정련, 또는 (2) 추가 정련/실행은 별도 명시 승인. 사용자 의도("계획 문서만, 실행 X")에 따라 autopilot/team/ralph 등 실행 모드로의 자동 전환은 차단됩니다.
