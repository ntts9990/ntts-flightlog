# ntts-flightlog Product Direction

## Positioning

ntts-flightlog is a live sidecar for AI coding sessions. It keeps the human
oriented by turning messy agent activity into turns, decisions, evidence, and
blockers inside the terminal.

The product is not trying to be a cloud dashboard, session replay system, or
general project manager. Its durable difference is proximity: it runs next to
the agent in a tmux pane and shows the smallest useful state that lets the human
retain control while work is happening.

## Differentiators

- Live sidecar, not dashboard: show the current operating picture inside the
  terminal where the coding agent is already running.
- Human-readable first: optimize for short Korean status, compact summaries,
  and scanability rather than raw trace completeness.
- Agent discipline: make the agent externalize intent, decisions, evidence, and
  blockers while work is in progress.
- Repo-local by default: SQLite plus markdown, no account, no cloud dependency,
  and private-by-default state.
- Turn as the primitive: a turn is the bounded work unit with intent,
  constraints, completion criteria, status, and result.

## View Contract

Each pane view must answer a different question.

### 1. Flat: Live Log

Question: what just happened?

Flat remains the raw chronological activity stream. It is allowed to be noisy
because its job is recency and audit trail continuity.

### 2. Turns: Turn Index

Question: what work units exist, where are they, and what came out of them?

Turns must not repeat full log bodies. It should show each turn as a compact
index row with status, elapsed time, entry counts, risk/evidence signals, and
the latest result.

### 3. Decisions: Decision Log

Question: why did we choose this path, and what evidence supports it?

Decisions are ADR-lite records. They should show turn context, reason/detail,
evidence linkage, and status when available. A decision is for choices that are
expensive to reverse or easy for future agents to forget.

### 4. Blockers: Open Risks

Question: what is currently blocking or threatening progress?

Blockers should prioritize open items, show resolved items separately, and make
turn context plus timing visible. A blocker is not just a red log entry; it is a
work-management object.

## Design Guardrails

- Do not add a broad web dashboard before the terminal sidecar is excellent.
- Do not make every view a filtered copy of the same timeline.
- Do not surface raw tool-call volume unless it directly improves orientation.
- Prefer compact status objects over prose-heavy logs in the side pane.
- Keep Korean pane-visible content by default.
- Do not add LLM-backed product behavior without a prompt contract, structured
  output schema, eval fixtures, and redaction/injection handling under
  `docs/llm-prompting-policy.md`.

## Product Decision Records

- `docs/user-investor-question-decisions.md` maps likely user, manager, buyer,
  and investor questions to explicit attach / next / later / reject decisions.
