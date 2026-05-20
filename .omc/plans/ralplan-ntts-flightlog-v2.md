# Consensus Plan: NTTS Flightlog v2 (6-month rewrite)

**Status**: PENDING APPROVAL (iteration 2 — post-Architect + Critic consensus)
**Mode**: short RALPLAN-DR
**Spec source**: `.omc/specs/deep-interview-v2-roadmap.md` (ambiguity 19.3%, 9 rounds)
**Generated**: 2026-05-20
**Output policy**: Plan-only. No execution auto-transition. Iter 2 incorporates Architect's 3 Synthesis Proposals + Critic's P0/P1 items.

---

## Requirements Summary

NTTS Flightlog v2 reimplements the v1 bash CLI as a single static Go binary that retains live tmux pane worklog capability and adds offline analytics across **5 core metrics** (turn duration, blocker accumulation, agent completion rate, agent blocker frequency, evidence-bound decision ratio). Three personas (self-retro / agent-operator / team-share) must each cite ≥4 of these 5 metrics in retrospective use before v2.0 GA ships. Six-month single milestone, single developer, six sequential phases (A Foundation → B Renderer+Metrics → **B.5 Synthetic Retro Rehearsal** → C Distribution+CI → D Test Hardening → E Retrospective Gate → F GA).

**Locked from spec (not re-litigated)**: Go language, Bubble Tea + modernc.org/sqlite + GoReleaser stack, agent ID hybrid auto-detect + `--agent` override, MCP/real-time deferred, fullstack 6-month scope (no MVP cut), retrospective gate X≥4.

---

## RALPLAN-DR Summary

### Principles (5, tightened in iter 2)

1. **Brownfield continuity beats greenfield purity** — v1 users `flightlog migrate` lossless to v2, parallel-run friendly, no forced cutover. "Lossless" = 7 enumerated equality predicates (see A5).
2. **CGo-free or bust** — no native C deps in any build target so the 5-OS CI matrix stays a single-runner build (modernc.org/sqlite enforces this). **If cold-start benchmarks fail the budget on any target, the plan halts for re-scoping** — fallback to `mattn/go-sqlite3` (CGo) is **NOT** an in-phase pivot. (Iter 2: hardened from Architect violation finding.)
3. **Acceptance is qualitative-first, but qualitative-checked-externally** — the retrospective gate X≥4 (3 personas) is the GA gate. *Critic finding*: self-grading collapses the gate. Iter 2 adds: pre-registered citation extractor + ≥1 truly external persona-occupant + adversarial review of evidence doc.
4. **Phase-gated cumulative risk with explicit slip surfacing** — each Phase A-F has explicit exit criteria. Any slip past phase due-date must be logged to `.omc/specs/v2-slip-log.md` within 24h with date + deferred items + remaining work; alpha journal + retro gate items NEVER slip. (Iter 2: hardened from Architect soft-violation finding.)
5. **Deferral is a feature** — MCP server, real-time intervention, web dashboard, external sync are explicitly v2.1+/v3+. Within F2 (docs), only migration guide is slip-able; **metric-interpretation guide is GA-blocking** because the retrospective gate depends on users understanding what each metric means.

### Decision Drivers (top 3)

1. **Single-developer 6-month feasibility** — every architectural choice picked must shorten iteration loop (compile time, deploy automation, learning curve).
2. **Retrospective gate must be reachable + falsifiable** — alpha dogfooding starts Phase D (Month 4) AND synthetic rehearsal at Phase B.5 (Month 2) so metric-redesign risk surfaces 2 months earlier than v1 plan. Adversarial review prevents author self-attestation theater.
3. **5 metrics must trace to SQL on real data** — every metric's SQL view must be executable on migrated v1 data; if v1 → v2 round-trip loses data needed for any metric, the metric or the schema must change before Phase B exits.

### Viable Options (≥2, substantively explored in iter 2)

#### Option A: Sequential 6-phase + B.5 rehearsal (CHOSEN)

- **Approach**: Phase A (Month 1, Foundation+migrate) → B (Month 2, Renderer+Metrics) → **B.5 (Month 2 last week, Synthetic Retro Rehearsal)** → C (Month 3, Distribution+CI) → D (Month 4, Test Hardening + alpha) → E (Month 5, Retro gate + polish) → F (Month 6, GA).
- **Pros**: Matches spec verbatim, lowest deliberation cost, alpha starts at the right moment (after Distribution so installer is testable), B.5 surfaces metric-design flaws 2 months before Phase E.
- **Cons**: Single milestone risk — if Phase D alpha reveals fundamental metric design flaw despite B.5 rehearsal, only Month 5 buffer. Partial fallback: F2a (migration guide) can slip to v2.0.1, never F2b (metric interpretation).

#### Option B: Phase B/C swap (Distribution earlier) — REJECTED on substantive grounds

- **Approach**: A → C → B → D → E → F. Ship installer + CI matrix in Month 2 so multi-OS binary issues surface earlier (modernc.org/sqlite cold-start on Windows, cross-compile toolchain surprises).
- **Pros (stronger reading)**: Earlier multi-environment shakedown catches OS-specific Go compile + SQLite-driver issues 1 month earlier; alpha testers could install nightly builds from Month 3.
- **Substantive rejection**: The largest single failure mode for v2 is *metric design wrongness*, not *cross-platform build issues*. Architect Synthesis Proposal 3 (cross-platform cold-start bench in A2) addresses the multi-OS de-risk at Month 1 *without* swapping phases. Earlier installer also provides little alpha value because v2's analytics is the *reason to install* — installer without metrics is low-information. Option A + A2 bench captures Option B's de-risk benefit without its serialization cost. **Rejected.**

#### Option C: Vertical-slice 6-week sprints — REJECTED on substantive grounds (not procedural)

- **Approach**: Sprint 1 (sessions+turns end-to-end) → Sprint 2 (entries+blockers+decisions+evidence) → Sprint 3 (5 metrics + report view) → Sprint 4 (distribution + CI + alpha + GA).
- **Pros**: End-to-end working software every 6 weeks, earlier integration feedback.
- **Substantive rejection**: Single dev's cognitive bandwidth for full-stack touching every sprint creates serialization on the critical path. Phase B (Bubble Tea TUI) is the steepest learning curve and is unsuited to interleaving with backend + distribution. Vertical slicing also fights the deep-interview spec's natural componentization (5 components, phase-organized). Context-switch cost per sprint exceeds the integration-feedback benefit for a *single* dev. **Rejected on dev-economics grounds, independent of R6 spec lock.** (R6 lock is a corroborating but not the primary reason.)

→ **Option A proceeds, augmented with Phase B.5.** Options B and C documented with substantive invalidation (no strawmen).

---

## Implementation Steps (Phase A–F, traced to spec + Critic operationalizations)

### Phase A — Foundation (Month 1)

Trace: spec § "Phase A: Foundation"; covers Constraints (Functional § Runtime, DB, Backward compat) + Goal § 5 metrics data model.

**A1. Go module bootstrap**
- Files: `go.mod`, `go.sum`, `cmd/flightlog/main.go`, `internal/cli/root.go`
- `go mod init github.com/ntts9990/ntts-flightlog`
- Cobra CLI skeleton: `flightlog --help` lists 14 subcommands (12 v1 + `report` + `migrate`)
- Version embedded via `-ldflags "-X main.version=..."`

**A2. SQLite 7-table schema + migration runner + concurrency + cross-platform bench**
- Files: `internal/db/schema.go`, `internal/db/migrations/0001_init.sql`, `internal/db/db.go`, `internal/db/concurrent_test.go`, `internal/db/bench_test.go`
- Tables (per spec § Ontology):
  - `sessions(id, started_at, ended_at, mode, agent_id, agent_detected, agent_override, title, focus, next_step)` — **`agent_detected` and `agent_override` are separate columns**, both nullable; aggregation prefers override when present; mismatch is logged.
  - `turns(id, session_id, sequence_no, title, started_at, ended_at, status, elapsed_ms, agent_id)`
  - `entries(id, session_id, turn_id NULL, kind, title, detail, created_at, agent_id)`
  - `blockers(id, turn_id, entry_id, title, opened_at, closed_at, status, accumulated_seconds)`
  - `decisions(entry_id PK)` (view over entries WHERE kind='decision')
  - `evidence(entry_id PK)` (view over entries WHERE kind='evidence')
  - `decision_evidence_links(decision_entry_id, evidence_entry_id, created_at, note)`
- Driver: `modernc.org/sqlite` (CGo-free, validates Principle 2)
- **Connection settings (locked at iter 2)**: `PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA synchronous=NORMAL` set on every connection open in `internal/db/db.go`. Single `database/sql` connection per process; if profiling reveals contention, escalate to small pool (max 3) — *but not arbitrarily*.
- Migration runner: minimal home-grown (`flightlog migrate up/down`) — avoids `golang-migrate` heavy dep; single binary KPI.
- DB path: `.ntts-flightlog/flightlog.db` (per spec)
- **Concurrent-writer test** (`concurrent_test.go`): spawn 2 goroutines — one running 100 `INSERT INTO entries`, the other running 100 `SELECT` from report views. Assert zero `SQLITE_BUSY` errors and total time < 1s. **Must pass on all 5 OS×arch CI targets.**
- **Cold-start benchmark** (`bench_test.go`): `BenchmarkColdOpen` opens DB + reads version row, runs in CI matrix on all 5 OS×arch targets. **Fails CI if median > 60ms (40ms headroom under the 100ms total budget).**

**A3. Agent auto-detection module**
- Files: `internal/agent/detect.go`, `internal/agent/detect_test.go`
- Env heuristic (per spec § Constraints):
  - `CLAUDE_DESKTOP_VERSION` set → `claude`
  - `CODEX_HOME` set → `codex`
  - `GEMINI_API_KEY` set → `gemini`
  - Parent process tree match: `claude-desktop`, `codex`, `gemini` → corresponding agent
- Returns `(detected_agent string, signals []string)` so audit log keeps evidence
- Unit tests: fixture-based for Claude/Codex/Gemini env-var sets + unknown fallback. Explicit fixture file `testdata/agent_fixtures/{claude,codex,gemini,unknown}.json` enumerates each scenario.

**A4. CLI subcommands (12 v1 + `migrate` + `report`)**
- Files: `internal/cli/{start,stop,auto,status,mode,turn_start,turn_end,entry,decision,evidence,blocker,path,turn_path,view,migrate,report}.go`
- Each subcommand writes to SQLite AND mirrors to `main.md` (read-only export view for v1 compat)
- `--agent <name>` global flag overrides auto-detection; both `agent_detected` and `agent_override` recorded in sessions/turns/entries
- Korean default labels preserved (per v1 SKILL.md)

**A5. `flightlog migrate` v1 → v2 with enumerated lossless predicates**
- Files: `internal/migrate/v1.go`, `internal/migrate/v1_test.go`
- Parse `.ntts-flightlog/main.md`, `turns/turn-*.md`, `mode`, `turn-counter`, `session-start-epoch`, `turn-start-epoch` (ignore `pane-id`)
- Insert into SQLite preserving timestamps, agent_id (best-effort detection at migration time)
- **"Lossless" operationalized as 7 round-trip equality predicates** (iter 2, addresses Critic P0.4):
  1. Entry count equality (`SELECT COUNT(*) FROM entries` post-migrate = source entry count)
  2. Timestamp byte-equality (ISO 8601 string preserved verbatim)
  3. Kind preserved (entry/decision/evidence/blocker/mode)
  4. **Title UTF-8 NFC byte-equality** (explicit NFC normalization before comparison; macOS HFS+ NFD risk addressed)
  5. **Detail multi-line body byte-equality** (newlines, indentation, embedded backticks preserved)
  6. **OSC 8 URL payload byte-equality** (turn-title hyperlinks survive round-trip)
  7. Ordering preserved (entries in source order; `sequence_no` monotonic per session)
- **A5 round-trip test fixture must include**: ≥1 entry combining Korean text + emoji + OSC 8 hyperlink + multi-paragraph detail. Author's own `.ntts-flightlog/main.md` (137 lines, 41 `###` headings) is the primary fixture; synthetic edge-case fixture supplements.

**A-Exit criteria (iter 2)**:
- `flightlog migrate` runs lossless (7 predicates above PASS) on author's own `.ntts-flightlog/main.md`
- All v1 subcommands produce identical Korean-headed output on enumerated golden test fixtures (`testdata/golden/v1_subcommand_*.txt`, terminal width pinned to 100 cols)
- `flightlog --version` cold start < 100ms on M1 + warm fs
- **Concurrent-writer test PASS on all 5 OS×arch CI targets** (iter 2)
- **Cold-start bench median ≤ 60ms on all 5 OS×arch CI targets** (iter 2)
- **`agent_detected` and `agent_override` columns enumerated in schema** + sample query that disambiguates them documented in `internal/db/README.md`

### Phase A.5 — Turn Intent Anchor (Month 1 last 2 days) [NEW — post-execution discovery]

Trace: spec Addendum (2026-05-20); user real-world pain — agent context drift caused 1-hour work loss in a different session. v2 thesis expanded from "metric analytics" to also include "intent persistence across agent context loss."

**A.5.1. 0002 migration: turns table anchor columns**
- File: `internal/db/migrations/0002_turn_anchors.sql`
- `ALTER TABLE turns ADD COLUMN intent TEXT;`
- `ALTER TABLE turns ADD COLUMN constraints_json TEXT;`
- `ALTER TABLE turns ADD COLUMN done_when TEXT;`
- `ALTER TABLE turns ADD COLUMN drift_alerts INTEGER NOT NULL DEFAULT 0;`
- `ALTER TABLE turns ADD COLUMN anchor_last_shown_at DATETIME;`
- Existing rows: all new columns NULL/0 (backward compat preserved for A5 migrate)

**A.5.2. turn-start CLI flags**
- File: `internal/cli/turn_start.go` (extend)
- Flags: `--intent <text>`, `--constraints <comma-sep>` (parsed to JSON array), `--done-when <text>`
- All optional; if none provided, anchor block omitted from main.md
- main.md mirror: append Korean anchor block after `### TS [turn-N-start] title`:
  ```
  ⚓ 의도: {intent}
  📐 제약: {constraints[0]} | {constraints[1]} | ...
  ✅ 완료조건: {done_when}
  ```

**A.5.3. `flightlog refresh-anchor [TURN_ID]` command**
- File: `internal/cli/refresh_anchor.go`
- Reads turns row for current (or specified) turn → prints anchor block to stdout in Korean
- Updates `turns.anchor_last_shown_at`
- Use case: agent in mid-task can run this command to re-prime context after compression

**A.5.4. `flightlog drift-check [TURN_ID]` command**
- File: `internal/cli/drift_check.go`
- For current/specified turn:
  - Fetch entries WHERE turn_id = X AND created_at > turn.started_at
  - For each entry, simple keyword-overlap check against constraints (e.g., if constraint="git history만" and entry detail contains "DB 직접 쿼리", flag drift)
  - On drift detected: `INSERT INTO blockers(turn_id, title="auto-drift: <reason>", ...)` + `UPDATE turns SET drift_alerts = drift_alerts + 1`
- Returns drift count + list of mismatches via stdout (Korean)
- v2.0 = keyword matching only. NLP semantic check deferred to v2.1+.

**A.5.5. Renderer anchor display**
- File: `internal/worklog/view.go` (extend)
- `RenderFlat` and `RenderTurns`: for each turn with non-NULL intent, render anchor block (Lipgloss style — distinct color, e.g., 117 cyan)
- After each `entry/decision/evidence/blocker` CLI write, if current turn has anchor + (entry count since last anchor_last_shown_at) ≥ 5: print `⚓ ANCHOR REMINDER: <intent>` to stdout

**A.5-Exit criteria**:
- `flightlog migrate` from v1 main.md still works (A5 round-trip predicates still PASS — NULL anchor columns are OK)
- `flightlog turn-start --intent X --constraints Y --done-when Z` populates DB + main.md anchor block
- `flightlog refresh-anchor` outputs anchor + updates `anchor_last_shown_at`
- `flightlog drift-check` detects a planted keyword drift (positive test) + does NOT false-positive on aligned entry (negative test)
- `view flat` shows anchor block per turn
- `go test ./...` 100% PASS, race clean, vet clean

### Phase B — Renderer + Metrics (Month 2)

Trace: spec § "Phase B: Renderer + Metrics"; covers core-cli Goal/Constraints, 5 metrics SQL.

**B1. Bubble Tea TUI skeleton**
- Files: `internal/tui/app.go`, `internal/tui/views/{flat,turns,decisions,blockers,report}.go`
- Bubble Tea Model–Update–View: pane content is Lipgloss-styled markdown
- Menu header (1/2/3/4/5/r/q keys) — 5th view = `report`
- **DB change notification**: prefer SQLite `update_hook` via cgo-free polling on `sqlite_sequence` table (2s tick) over file mtime; if WAL `journal_mode` makes mtime unreliable, this is required. (Iter 2: addresses Architect Tradeoff Tension #3.)

**B2. Color system port from v1**
- File: `internal/tui/styles.go`
- Lipgloss styles mapped 1:1 from v1 awk: category-fixed (entry=109, decision=215, evidence=114, blocker=203, mode=220) + 8-turn cycle (207/39/213/99/198/165/75/141)
- OSC 8 hyperlink wrapping for turn titles → `file://` per-turn export

**B3. 5 metric SQL views + Go query functions**
- Files: `internal/metrics/metrics.go`, `internal/db/migrations/0002_metric_views.sql`
- 5 SQL views (per spec):
  - `metric_turn_duration` — `SELECT turn_id, agent_id, elapsed_ms FROM turns`
  - `metric_blocker_accumulation` — `SELECT blocker_id, opened_at, closed_at, accumulated_seconds FROM blockers`
  - `metric_agent_completion` — `SELECT agent_id, COUNT(*) FILTER (WHERE status='complete') * 1.0 / COUNT(*) AS rate FROM turns GROUP BY agent_id`
  - `metric_agent_blocker_freq` — `SELECT agent_id, COUNT(blockers.*) * 1.0 / COUNT(DISTINCT turns.id) AS freq FROM turns LEFT JOIN blockers ... GROUP BY agent_id`
  - `metric_evidence_bound_decisions` — ratio of decisions with linked evidence over total decisions
- Each metric has Go func + unit test on fixture data

**B4. `flightlog report` command**
- Files: `internal/cli/report.go`, `internal/cli/report_format.go`
- Flags: `--format text|json`, `--window day|week|all`, `--agent <name>`
- Text output: 5 sections, ANSI-colored summary table per metric
- JSON output: schema-stable, frozen JSON schema at `testdata/golden/report_schema.json` + validation in `internal/metrics/schema_test.go`

**B-Exit criteria (iter 2, contradiction resolved)**:
- v2 renders all v1 views (flat/turns/decisions/blockers) **byte-identical to v1 ANSI output** on enumerated fixed-seed inputs. Terminal width pinned to 100 cols, terminal type `xterm-256color`. (Iter 2: "visual diff acceptable" removed; ANSI byte-equality is the gate. Visual differences in user's own terminal are informational only, not gating.)
- All 5 metrics produce correct numbers on **enumerated fixture: exactly 10 sessions × 3 turns × exactly 47 entries**, distributed as: 30 entries (`kind=entry`), 8 decisions, 6 evidence (4 linked to decisions, 2 unlinked), 3 blockers (2 resolved, 1 open). Expected metric values pre-computed and frozen in `testdata/fixture_expected_metrics.json`.
- `flightlog report --format json` output validates against frozen JSON schema.

### Phase B.5 — Synthetic Retro Rehearsal (Month 2, last 2 days) [NEW in iter 2]

Trace: Architect Synthesis Proposal 1; closes the load-bearing risk of Month-5 metric-redesign with no buffer.

**B5.1. Seed dogfood DB with B-Exit fixture**
- Copy `testdata/fixture_expected_metrics.json` fixture into a runtime DB at `.ntts-flightlog/rehearsal.db`
- Run `flightlog report` against rehearsal.db for each persona window

**B5.2. Author writes 1-page mock retrospective per persona**
- File: `.omc/specs/v2-retro-rehearsal.md`
- Section 1 (self-retro): author writes a mock "weekly retrospective" using ONLY the metrics surfaced by `flightlog report` from rehearsal.db; no other data sources
- Section 2 (agent-operator): mock "Codex vs Claude decision" document citing metrics
- Section 3 (team-share): mock weekly status report

**B5.3. Citation extraction pre-registered**
- File: `scripts/extract_citations.sh` (or `cmd/citation-extractor/main.go`)
- Regex/keyword matcher for each metric's canonical name: e.g., `(?i)(turn duration|turn 소요시간|turn elapsed)` → `metric_turn_duration` cited
- **Pre-registered before B5.2** (write extractor before writing rehearsal docs to prevent post-hoc tuning)

**B5.4. Rehearsal gate**
- Run extractor over `v2-retro-rehearsal.md` sections
- **If any persona section <3 distinct metrics cited, redesign that metric SQL or rename for clarity BEFORE entering Phase C**
- This is the **only allowed exit-gate redesign** path; later redesigns escalate to slip log per Principle 4

**B5-Exit criteria**:
- Citation extractor frozen and committed
- Rehearsal docs written
- All 3 persona sections cite ≥3 metrics (rehearsal threshold; real gate at Phase E uses ≥4)
- Any metric SQL changes documented in `0002_metric_views.sql` revision history before C1 begins

### Phase C — Distribution + CI (Month 3)

Trace: spec § "Phase C: Distribution + CI" + § Distribution Constraints.

**C1. GoReleaser configuration**
- File: `.goreleaser.yml`
- Builds (5 explicit targets, iter 2 enumerated): `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`, `windows/amd64`. (No `windows/arm64` — see R4.)
- Archives: `tar.gz` (unix) + `zip` (windows), include `LICENSE`, `README.md`, `CHANGELOG.md`
- `-ldflags "-s -w -X main.version={{.Version}}"` for binary size ≤ 25MB
- Brews tap: `ntts9990/homebrew-tap` (separate repo, `Formula/flightlog.rb`)
- Scoop bucket: `ntts9990/scoop-bucket` (separate repo)
- AUR (optional, lower priority): publish bin package

**C2. GitHub Actions CI matrix**
- File: `.github/workflows/ci.yml`
- Explicit OS×arch combos (iter 2 enumerated):
  - `macos-14` (arm64) → builds `darwin/arm64`
  - `macos-13` (amd64) → builds `darwin/amd64`
  - `ubuntu-22.04` (amd64) → builds `linux/amd64`
  - `ubuntu-22.04-arm` (arm64) or qemu-arm runner → builds `linux/arm64`
  - `windows-2022` (amd64) → builds `windows/amd64`
- Jobs per PR: `lint` (golangci-lint), `test` (go test ./... -race), `build` (cross-compile target), `e2e` (smoke), `golden` (snapshot diff), **`concurrent` (A2 concurrent-writer test)**, **`bench` (A2 cold-start bench)**
- Release workflow: tag `v*` triggers GoReleaser
- **Evidence-doc CI lint** (iter 2, addresses Critic P1.12): `scripts/lint_evidence_doc.sh` checks `docs/v2-ga-acceptance-evidence.md` for 3 section headings + ≥4 metric tokens per section + ratio computation. Runs as required CI check before release tag.

**C3. install.sh simplification**
- File: `scripts/install.sh` (rewrite from v1's 100+ lines to ~40 lines)
- Detect OS/arch → download GitHub Release asset → verify checksum → install to `$LOCAL_BIN_DIR` (default `~/.local/bin`)
- `--codex/--claude/--gemini/--all/--no-cli` flags scoped to *skill SKILL.md installation only* (not CLI binary which is now download-based)

**C4. `flightlog self-upgrade` command**
- File: `internal/cli/self_upgrade.go`
- Poll GitHub releases API for latest tag, compare against embedded version
- If newer: download asset → verify checksum → atomic replace (`os.Rename` after tmp download)
- Refuse to upgrade if running via Homebrew (let user `brew upgrade` instead)
- **Designated slip target** (per Principle 5 + R5): if C4 slips past Month 3 EOD, defer to v2.0.1 — log entry in `.omc/specs/v2-slip-log.md`.

**C-Exit criteria**:
- `brew install ntts9990/tap/flightlog` works on M1 macOS and Intel macOS
- `scoop install flightlog` works on Windows 11
- CI matrix 5 jobs green on `main` for 3 consecutive PRs
- All 5 release artifacts ≤ 25MB after `-ldflags "-s -w"`

### Phase D — Test Hardening + Alpha (Month 4)

Trace: spec § "Phase D: Test Hardening + Acceptance" + § Acceptance Criteria.

**D1. Test suite expansion**
- Files: `internal/**/*_test.go`, `e2e/*.go`, `testdata/golden/*.json`
- Unit: all public functions in `internal/agent`, `internal/db`, `internal/metrics`, `internal/migrate` ≥80% line coverage
- Integration: SQLite migration round-trip (7 predicates), agent detection per fixture environment, metric SQL on seeded fixture
- Sub-command E2E: `e2e/cli_test.go` runs binary as subprocess, asserts stdout/exit code
- Golden snapshot: `report --format json` output frozen via `goldie`

**D2. P0 24-scenario automation**
- File: `testdata/p0_scenarios.yaml` (24 scenarios from v1 manual checklist) + `e2e/p0_test.go`
- YAML format: `name, setup_steps[], invoke[], expect_stdout_match, expect_state_match`
- S8 (stop/restart 후 main.md 이어붙임) and S9 (mode switch within turn) explicitly added — were untested in v1

**D3. Property/contract tests**
- File: `internal/{agent,db,metrics}/*_property_test.go`
- agent_id format: regex `^[a-z][a-z0-9_-]{1,31}$` invariant
- SQLite schema migration: forward then backward = identity
- 5-metric invariants: e.g., `0 ≤ evidence_bound_decisions ≤ 1`, `agent_completion_rate ≤ 1`

**D4. Alpha self-deployment (Month 4 week 2) [iter 2: bar raised]**
- Author dogfoods v2 binary on own work
- Creates `.omc/specs/alpha-dogfood-log.md` — daily journal across 4 weeks
- **Each weekday entry MUST contain ≥1 metric citation + ≥3 entries/week with metric-triggered behavior change** (iter 2: raised from "1+ daily entry written")
- Citation extractor (from B5.3) runs over journal; weekly summary committed

**D-Exit criteria**:
- CI green on `main` for D1-D3 work (including concurrent + bench)
- Alpha journal active with ≥3 daily entries/week + ≥1 citation per entry (per iter 2 raised bar)
- P0 24 + S8 + S9 all PASS automated (no manual checklist remains)

### Phase E — Retrospective Gate (Month 5)

Trace: spec § "Phase E: Retrospective Gate" + § Acceptance Retrospective Gate; persona independence machinery added in iter 2.

**E1. Self-retro persona validation**
- Continuation of D4 alpha journal — 4 full weeks
- **Spontaneous citation protocol** (iter 2, addresses Critic P0.1):
  - Author writes daily journal WITHOUT looking at the 5-metric list during writing
  - Citation extractor (B5.3) runs *post-hoc* over journal text → counts distinct metric token matches
  - Behavior change tracked via journal entries tagged `[CHANGED-BY-METRIC: <metric_id>]` (operationalized, per Critic P0)
- Pass condition: ≥4 of 5 distinct metrics cited in journal + ≥1 `[CHANGED-BY-METRIC]` tag

**E2. Agent-operator persona validation**
- **Persona-occupant**: author may occupy (spec permits, L120 "본인 또는 외부 1명"), BUT iter 2 adds:
  - Author writes ≥1 "agent-comparison decision document" in `.omc/specs/v2-agent-operator-decisions.md`
  - Document cites ≥4 metrics with specific numbers from dogfood DB
  - **Adversarial review**: separate session of Claude/Codex sub-agent given raw documents + 5-metric list challenges each claimed citation ("Did the author cite this because metric was useful, or because gate requires it?"). Challenge results logged.

**E3. Team-share persona validation [iter 2: external requirement]**
- **Requirement**: At least one weekly report must have a **real external recipient** (named in advance, e.g., a colleague, a fellow OSS contributor, an Anthropic team member). "Hypothetical audience" (per spec L122) is allowed for the *second* report, NOT both. (Iter 2: stricter than spec's permissive reading to address Critic P0.1.)
- Each report ≥4 of 5 metrics + plain-language interpretation
- External recipient acknowledges receipt (email/Slack message dated); ack saved to evidence doc

**E4. Auto-detection neutrality check [iter 2: detection-rate vs override-rate distinction]**
- Compute on dogfood DB at Month 5 end:
  - `auto_detect_correct_rate` = sessions where `agent_detected = agent_override OR agent_override IS NULL`, agent_detected NOT NULL
  - `auto_detect_unknown_rate` = sessions where `agent_detected IS NULL OR agent_detected = 'unknown'`
  - `override_rate` = sessions where `agent_override IS NOT NULL`
- **Gate**: `auto_detect_unknown_rate < 10%` (sessions where detection failed entirely)
- If higher: harden detection heuristics in Month 5 buffer
- (Iter 2: distinguishes failed detection from user override per Critic P0.7)

**E5. Polish**
- Cold-start budget: profile `flightlog entry` and trim < 100ms p99 on M1
- Binary size: trim deps if any target >25MB
- Bug fix sweep from alpha journal

**E-Exit criteria (iter 2)**:
- 3 personas × ≥4 metrics cited = gate PASS, validated by citation extractor (NOT author self-grading)
- ≥1 persona has real external occupant or recipient (E3 satisfies this)
- Adversarial review challenges responded to in evidence doc
- `auto_detect_unknown_rate <10%`
- All P0 + property tests green
- Cold start <100ms, binary ≤25MB per target

### Phase F — GA (Month 6)

Trace: spec § "Phase F: GA Release + v2.1 Planning"; F2 split in iter 2.

**F1. Release v2.0.0**
- Tag `v2.0.0` triggers GoReleaser
- GitHub Releases artifacts × 5
- Homebrew tap formula bumped
- Scoop bucket manifest bumped

**F2a. Migration guide [SLIP-ABLE]**
- File: `docs/migration-v1-to-v2.md` — full migrate command guide + caveats
- Can slip to v2.0.1 if Phase D/E slipped, per slip-surfacing protocol (Principle 4)

**F2b. Metric interpretation guide [GA-BLOCKING — iter 2 separation]**
- File: `docs/metrics-interpretation.md` — 5 metric definitions, example SQL, decision patterns, plain-language meaning
- **CANNOT slip**. Retrospective gate model assumes users understand each metric; without this doc, the gate doesn't work for any user beyond the author.
- (Iter 2: addresses Critic P0.2 doc-slip contradiction)

**F3. v2.1 backlog grooming**
- File: `.omc/plans/v2.1-backlog.md`
- Sections: MCP server design sketch, external sync (Linear/Notion) integration, real-time intervention research

**F-Exit criteria**:
- v2.0.0 release artifacts live on GitHub + Homebrew + scoop
- **F2b (metric interpretation guide) shipped at GA** (iter 2 hard gate)
- F2a (migration guide) shipped at GA OR slip log entry committed
- Retrospective gate evidence saved to `docs/v2-ga-acceptance-evidence.md`
- Evidence-doc CI lint (C2) PASS on the published evidence doc

---

## Risks and Mitigations (iter 2 revised)

| # | Risk | Probability | Impact | Mitigation (iter 2) |
|---|---|---|---|---|
| R1 | Retrospective gate X≥4 misses | M | High | **Phase B.5 Synthetic Retro Rehearsal at Month 2 end** catches metric-design flaws 2 months before real alpha (per Architect SP1). Real alpha at Month 4 week 2 → Month 5 has 3+ weeks redesign buffer. If both B.5 and alpha-month-1 reveal issues, escalate to slip log. |
| R2 | Agent auto-detection <90% reliable | M | M | Hybrid `--agent` override always available; `agent_detected` + `agent_override` columns disambiguate. Gate is `auto_detect_unknown_rate < 10%` measured on sessions w/o override (iter 2 disambiguation). |
| R3 | Bubble Tea learning curve eats Phase B | L | M | Phase B starts with 1-week spike (B1 only). If spike fails, fallback to Lipgloss-only ANSI port (no full Bubble Tea), still valid Go architecture. |
| R4 | GoReleaser misconfig on Windows arm64 | L | L | `windows/arm64` excluded from matrix; v2.1+ adds via `winget`. |
| R5 | Single-dev 6-month slip | M | High | **Explicit slip-surfacing protocol** (iter 2, Principle 4): Phase D EOD week 3 status check; any deferral committed to `.omc/specs/v2-slip-log.md` within 24h with date + items + remaining work. **Designated slip targets**: F2a (migration guide) → v2.0.1; C4 (self-upgrade) → v2.0.1. **NEVER slip**: alpha journal, retro gate, F2b (metric interpretation), E4 detection ratio. |
| R6 | modernc.org/sqlite cold-start >100ms or contention | L→M | M | **Cross-platform bench in A2** measures on all 5 OS×arch (iter 2, Architect SP3). **No CGo fallback** — if bench fails, plan halts for re-scoping (iter 2, Principle 2 tightened). WAL + busy_timeout=5000 mitigates contention; concurrent test in A2 validates. |
| R7 | v1 main.md parse fidelity on edge entries | M | M | **A5 round-trip test enforces 7 enumerated predicates** including UTF-8 NFC, multi-paragraph bodies, OSC 8 payloads, ordering (iter 2 covers R7's enumerated failure modes). Hard gate at A-Exit. |
| R8 | Homebrew tap rejected by formula linter | L | L | Tap is author-owned (`ntts9990/homebrew-tap`); `brew audit` runs in CI. |
| R9 | Color rendering diff v1 vs v2 | M | L | **ANSI byte-equality** is the gate (iter 2, contradiction resolved). Visual differences in user's own terminal are informational only. |
| R10 | Spec ambiguity leak into late-phase implementation | M | L | **Reversibility classified** (iter 2): irreversible after A2 ships → schema (7 tables), connection settings (WAL+busy_timeout), agent_detected/override columns. Reversible until Phase B end → log format, report text-format styling. Document each in code comments where chosen. |
| R11 [NEW] | Persona independence collapse | M | High | **Pre-registered citation extractor (B5.3) + ≥1 external recipient (E3) + adversarial review (E2)** prevent author self-attestation theater (iter 2 addresses Critic P0.1). |
| R12 [NEW] | Metric-interpretation doc slips, breaking external-user retro gate | L | High | **F2b is GA-blocking** (iter 2, Principle 5 + Critic P0.2). Cannot slip; F2a can. |

---

## Verification Steps (iter 2 expanded)

### Automated (CI matrix every PR)
- `go vet ./... && golangci-lint run` → 0 warnings
- `go test ./... -race -count=1` → 100% pass
- `go test ./e2e/... -tags=e2e` → 100% pass
- `go test ./internal/db -run TestConcurrent` → 100% pass on all 5 targets (iter 2)
- `go test ./internal/db -bench=BenchmarkColdOpen -benchtime=5x` → median ≤ 60ms (iter 2)
- `make build-all` (5 target cross-compile) → 5 binaries, each ≤25MB
- Golden snapshot diff: `report --format json` against `testdata/golden/report.json`
- P0 + S8 + S9: `go test ./e2e -run TestP0Scenarios` → 26/26 PASS
- **`scripts/lint_evidence_doc.sh docs/v2-ga-acceptance-evidence.md`** → required check before release tag (iter 2)

### Manual (per phase exit)
- Phase A exit: author runs `flightlog migrate` on own `.ntts-flightlog/` and visually inspects DB content via `sqlite3 .ntts-flightlog/flightlog.db ".dump"`; 7-predicate round-trip output recorded
- Phase B exit: side-by-side v1 vs v2 pane on **enumerated fixed-seed input at 100-col terminal**, ANSI byte-equality verified (iter 2: no visual-diff escape clause)
- Phase B.5 exit: rehearsal docs written, extractor frozen, ≥3 metric citations per persona section (iter 2 NEW)
- Phase C exit: `brew install` on personal MacBook + `scoop install` on a Windows VM
- Phase D exit: alpha journal ≥3 daily entries/week × 4 weeks, citation extractor weekly summary committed (iter 2 raised bar)
- Phase E exit: retrospective gate evidence document with metric-citation counts ≥4 per persona, **adversarial review log present, external recipient ack present**
- Phase F exit: release artifacts visible on GitHub Releases + Homebrew + scoop; **F2b shipped at GA**

### Acceptance gate (Phase E required, Phase F unlocks)
- `docs/v2-ga-acceptance-evidence.md` contains 3 sections, each with:
  - 4+ direct quotes from alpha journal / agent-operator decision doc / team-share report
  - Each quote cites a named metric, verified by citation extractor (iter 2)
- Adversarial review log linked
- E3 external recipient ack linked (iter 2)
- `auto_detect_unknown_rate < 10%` measured per E4 disambiguation (iter 2)

---

## Plan Health Indicators (iter 2)

- **File/line citations**: 50+ specific file paths (>80% of claims)
- **Testable criteria**: 22 of 26 with concrete numbers/booleans; remaining 4 are qualitative gates (E1-E3 persona evidence + adversarial review) each operationalized with citation extractor + external ack (per Critic P0.1)
- **Vague terms**: 0 instances of "fast", "robust", "good UX" — all metrics quantified
- **All risks have mitigations**: 12/12 ✓ (10 original + 2 new in iter 2)
- **Spec traceability**: Every Phase A-F task cross-references `.omc/specs/deep-interview-v2-roadmap.md` section
- **Principle violations**: 0 (iter 2 fixed both, per Architect)
- **Self-contradictions**: 0 (iter 2 fixed B-Exit byte-vs-visual, R6 fallback)

---

## ADR (iter 2 final)

### Decision
NTTS Flightlog v2 = 6-month single-milestone Go rewrite of v1 bash CLI, with offline 5-metric analytics, single static binary distributed via Homebrew/scoop/AUR/GitHub Releases, gated by qualitative retrospective acceptance (≥4 of 5 metrics cited per persona across 3 personas with pre-registered citation extraction + external persona-occupant + adversarial review).

### Drivers
1. Single-dev 6-month feasibility (architectural choices favor short iteration loop)
2. Retrospective gate reachable + falsifiable (B.5 rehearsal + adversarial review)
3. 5 metrics SQL-executable on migrated v1 data (lossless round-trip enforced)

### Alternatives Considered
- **Option B (Phase B/C swap)**: rejected on substantive grounds — A2 cross-platform bench captures multi-OS de-risk benefit without serialization cost.
- **Option C (vertical-slice sprints)**: rejected on substantive grounds — single-dev context-switch cost exceeds integration-feedback benefit; Bubble Tea learning curve unsuited to interleaving.
- **Rust / Zig** (deep-interview R9): rejected pre-plan; Go wins on 9/10 KPI axes.
- **MVP cut to 3-month v2.0** (deep-interview R6): rejected by user; fullstack 6-month preferred.
- **MCP-server first-class agent ID** (deep-interview R7): deferred to v2.1+.
- **mattn/go-sqlite3 CGo fallback** (iter 1 R6): dropped in iter 2 per Architect SP3 + Principle 2 hardening; no in-phase pivot.

### Why Chosen
Option A + B.5 rehearsal is the lowest-deliberation-cost path that (a) honors all spec locks, (b) provides 2-month early warning on metric design via B.5, (c) keeps multi-OS de-risk at Month 1 via A2 bench, (d) operationalizes the retrospective gate so it's not theater.

### Consequences
- 84 dev-day capacity utilization (~92% of 91-day budget) — tight but feasible.
- Designated slip targets: F2a (migration guide), C4 (self-upgrade) → v2.0.1 if needed.
- F2b (metric interpretation guide) is GA-blocking — adds 4-day Phase F docs commitment that cannot defer.
- WAL + busy_timeout commits to a specific concurrency model; SQLite write contention assumptions are testable from A2 onward.
- Principle 2 (CGo-free or bust) is strict — failed cross-platform bench = plan halt, not in-phase pivot.

### Follow-ups
- v2.1+ backlog (per spec § F3): MCP server, external sync, real-time intervention
- Post-GA: monitor `auto_detect_unknown_rate`; if it climbs >10% across broader user base, harden detection heuristics in v2.0.x patch
- Post-GA: evaluate B.5 rehearsal effectiveness — did it actually catch the issues that the real Phase E would have caught?

---

## Capacity Budget (iter 2)

- 130 calendar working days × 70% effective = **91 effective dev-days**
- Iter 2 estimated effort: **~84 dev-days** (iter 1's 79 + 4.5 from Synthesis Proposals + ~0.5 net from Critic operationalizations after dedup)
- Headroom: ~7 dev-days (~8%)
- Designated slip targets (F2a, C4) provide additional 7 dev-days of buffer if Phase D slips past Month 4 week 3

### Known underestimates (Architect flagged)
- A4 (14 subcommands w/ SQLite + main.md mirror) at 8 days may actually be 12-15 days
- D1 (≥80% line coverage) at 8 days may actually be 12 days if tests are retrofit
- Phase E elapsed floor: 20 working days (calendar-bound, uncompressible)

### Slip-trigger protocol
- **Phase D EOD week 3**: hard status check. If incomplete: log slip + activate F2a/C4 deferral path.
- **Phase E EOD week 2**: rehearse Phase E adversarial review on partial evidence. If review reveals systemic citation gaps, escalate to spec-level decision (potentially: defer GA to v2.0 RC).
- **F2b status at Phase F start**: must be in draft. If not drafted by Month 6 week 1, this is the critical-path signal that GA itself is at risk.

---

## Changelog

- **2026-05-20 (iter 1)**: Initial draft from Planner pass.
- **2026-05-20 (iter 2, this version)**: Architect approve-with-changes (3 Synthesis Proposals) + Critic approve-with-changes (P0 1-7, P1 8-13, P2 14-16) applied. Key changes:
  - Phase B.5 Synthetic Retro Rehearsal inserted (Month 2 last 2 days)
  - A2 adds WAL + busy_timeout + concurrent test + cross-platform cold-start bench
  - A5 "lossless" operationalized as 7 enumerated equality predicates including UTF-8 NFC
  - B-Exit ANSI byte-equality only (visual-diff escape clause removed)
  - E1-E3 persona independence machinery: pre-registered citation extractor (B5.3) + adversarial review (E2) + ≥1 real external recipient (E3)
  - E4 detection-rate vs override-rate disambiguation
  - F2 split: F2a migration guide (slip-able) + F2b metric interpretation guide (GA-blocking)
  - Principle 2 hardened (drop CGo fallback), Principle 4 hardened (explicit slip surfacing)
  - R5 explicit slip protocol, R6 CGo fallback removed, R7 aligned with A5, R10 reversible/irreversible classified, R11/R12 added
  - Option B/C invalidations replaced with substantive critique (not procedural)
  - ADR section filled
  - Capacity budget: 84 dev-days estimated, 92% utilization, designated slip targets enumerated
  - Self-contradictions audited: 0 remaining

---

## Final Status

**PENDING APPROVAL** — This plan is the output of the deep-interview → omc-plan consensus chain. Per user's stated intent ("계획 문서만, 실행 X"), no auto-handoff to autopilot/team/ralph. To proceed with execution, the user must explicitly approve a separate execution path (`/team`, `/ralph`, or `/autopilot`) referencing this plan path. Until then, the plan stands as the deliverable.
