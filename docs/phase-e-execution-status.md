# Phase E 1-5 Execution Status

Status date: 2026-05-27

This status records the `$ralph` attempt to run the remaining Phase E sequence
in order. It separates completed local evidence from work that requires time or
a real external participant.

## 1. Native Hook Install/Execute Evidence

Status: partially complete.

Completed:

- Claude Code, Codex, and Gemini CLIs all pass `--version` inside tmux panes.
- `ntts-flightlog hooks print --agent codex|claude|gemini` prints
  non-mutating starter commands.
- Starter-shaped ingest smoke events for `codex`, `claude`, and `gemini` were
  accepted by `ntts-flightlog ingest` with redaction version
  `storage-redaction-2026-05-21`.
- `scripts/phase_e_agent_rehearsal.sh` now runs a temporary local worklog
  rehearsal for `codex`, `claude`, and `gemini` using explicit `--agent`
  overrides. Each agent completed `auto`, `turn-start`, `entry`, `evidence`,
  `turn-end`, `handoff`, and `stop`.
- The generated artifact is
  `docs/e0-3-agent-attachment-rehearsal.md`. It records
  `override_rate: 100.0%` and `auto_detect_unknown_rate: 100.0%`, so it proves
  explicit attribution works but does not prove native hook auto-detection.

Still required:

- Install or wire the starter command into each agent's real native hook runner.
- Capture dated evidence that each native hook fired from an actual agent
  session, not only from a manual ingest smoke.
- Reduce or explicitly account for unknown attribution in persistent
  `agent-stats` before using agent metrics for comparison.

## 2. Team-Share External Recipient Acknowledgement

Status: ready to send; blocked on real external participant response.

Completed:

- `.omc/specs/v2-team-share-report.md` now contains a concrete 2026-05-22
  weekly report draft with values for all five metrics.
- `docs/phase-e-team-share-outbound-packet.md` provides the ready-to-send
  external review request and acknowledgement capture instructions.

Still required:

- Choose a real recipient.
- Send the report through a dated channel.
- Record the recipient acknowledgement in the Team-Share artifact and
  `docs/v2-ga-acceptance-evidence.md`.

## 3. Self-Retro Journal Evidence

Status: started, not GA-complete.

Completed:

- `.omc/specs/alpha-dogfood-log.md` now contains a real 2026-05-22 Day 1 entry
  citing all five metrics and one behavior-change tag.

Still required:

- Continue real usage for four weeks.
- Maintain at least three daily entries per week.
- Preserve at least one real `[CHANGED-BY-METRIC: metric_id]` behavior change.

## 4. Adversarial Review

Status: completed for the current evidence state; verdict is REQUEST CHANGES.

Completed:

- `.omc/specs/v2-adversarial-review.md` now records a 2026-05-22 review.
- The review explicitly blocks GA on missing longitudinal Self-Retro evidence
  and missing Team-Share external acknowledgement.

Still required:

- Re-run adversarial review after Self-Retro and Team-Share are no longer
  placeholders.

## 5. Strict Readiness

Status: intentionally not passing.

Completed:

- Advisory readiness passes.
- Strict readiness was run with the updated checker and correctly fails on:
  - `placeholders`: 0 placeholders remain.
  - `alpha_dated_entries`: 1 dated entry, below the 12-entry minimum.
  - `external_ack`: no real external acknowledgement.
- `evidence-report --persona team-share` now reports all five metrics present
  and asks for the dated external acknowledgement as the next concrete gap.
- Self-Retro week-four evidence is explicitly deferred until real usage exists.

Still required:

- Complete the real external/time-bound evidence above.
- Re-run `ntts-flightlog evidence-check --strict`.

## 6. Semantic Evidence Readiness

Status: local checker hardening complete; GA evidence still incomplete.

Completed:

- `evidence-check` now distinguishes token-level metric mentions from concrete
  gate-counting evidence.
- `evidence-report --format json` preserves `present` as token/alias found and
  adds `status` plus `counts_toward_gate` for semantic readiness.
- Current Self-Retro deferred metric lines report as non-concrete instead of
  satisfying the persona gate.
- Token lint remains separate: `scripts/lint_evidence_doc.sh` checks that metric
  names are mentioned, while `ntts-flightlog evidence-check` checks whether those
  mentions are concrete enough for readiness.

Still required:

- Continue the real four-week Self-Retro journal.
- Add a dated real Team-Share external acknowledgement.
- Re-run adversarial review after those artifacts are concrete.
