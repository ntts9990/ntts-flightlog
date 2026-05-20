# internal/db

SQLite data layer for ntts-flightlog v2.

**Driver**: `modernc.org/sqlite` (CGo-free). Per plan Principle 2, `mattn/go-sqlite3` (CGo) is **not** a permitted fallback — if the cold-start benchmark fails any target, the plan halts for re-scoping.

**PRAGMAs** (applied on every connection open in `db.go`):
```sql
PRAGMA journal_mode=WAL;       -- concurrent readers + single writer
PRAGMA busy_timeout=5000;      -- wait up to 5 s before SQLITE_BUSY
PRAGMA synchronous=NORMAL;     -- good durability/perf balance with WAL
```

**Connection model**: single `database/sql` connection per process (`MaxOpenConns=1`). Escalate to a pool of max 3 only if profiling reveals contention.

---

## Schema (7 tables — plan A2)

| Object | Type | Description |
|---|---|---|
| `sessions` | table | One row per work session |
| `turns` | table | Sub-units within a session |
| `entries` | table | Primary log records (entry/decision/evidence/blocker/mode) |
| `blockers` | table | Blocking issues tied to turns/entries |
| `decisions` | **view** | `SELECT * FROM entries WHERE kind='decision'` |
| `evidence` | **view** | `SELECT * FROM entries WHERE kind='evidence'` |
| `decision_evidence_links` | table | Many-to-many decisions ↔ evidence |

---

## agent_detected vs agent_override — disambiguation

`sessions` has **two separate nullable columns** for agent tracking:

| Column | Meaning |
|---|---|
| `agent_detected` | Raw value from env-heuristic / process-tree auto-detection. `NULL` if detection found no signal. |
| `agent_override` | Value supplied via `--agent <name>` CLI flag. `NULL` if flag was not used. |
| `agent_id` | Effective agent: `COALESCE(agent_override, agent_detected)`. Stored denormalized for query convenience. |

### Sample query: sessions where detection and override disagree

```sql
-- Sessions where the user explicitly overrode the auto-detected agent.
-- These are candidates for heuristic improvement.
SELECT
    id,
    started_at,
    agent_detected,
    agent_override,
    agent_id          -- = agent_override (took precedence)
FROM sessions
WHERE agent_override IS NOT NULL
  AND agent_detected IS NOT NULL
  AND agent_override != agent_detected
ORDER BY started_at DESC;
```

### Sample query: detection failure rate (Phase E auto-detect gate)

```sql
-- auto_detect_unknown_rate: sessions where detection failed entirely.
-- Gate: this rate must be < 10% (plan E4).
SELECT
    COUNT(*) FILTER (WHERE agent_detected IS NULL OR agent_detected = 'unknown') AS failed_detection,
    COUNT(*) FILTER (WHERE agent_override IS NULL)                                AS without_override,
    ROUND(
        100.0 *
        COUNT(*) FILTER (WHERE (agent_detected IS NULL OR agent_detected = 'unknown') AND agent_override IS NULL) /
        NULLIF(COUNT(*) FILTER (WHERE agent_override IS NULL), 0),
        1
    ) AS auto_detect_unknown_rate_pct
FROM sessions;
```

### Sample query: agent completion rate (metric 3)

```sql
-- Per-agent turn completion rate (one of the 5 core v2 metrics).
SELECT
    agent_id,
    COUNT(*) FILTER (WHERE status = 'complete') AS complete_turns,
    COUNT(*)                                     AS total_turns,
    ROUND(
        100.0 * COUNT(*) FILTER (WHERE status = 'complete') / COUNT(*), 1
    ) AS completion_rate_pct
FROM turns
WHERE agent_id IS NOT NULL
GROUP BY agent_id
ORDER BY completion_rate_pct DESC;
```

---

## Migration runner

`db.Open()` automatically applies pending migrations from `migrations/*.sql` in lexicographic order. Applied filenames are recorded in `_schema_migrations`. No external `golang-migrate` dependency — single binary KPI maintained.
