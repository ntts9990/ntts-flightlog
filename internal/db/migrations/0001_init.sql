-- 0001_init.sql: Initial schema for ntts-flightlog v2
-- 7 tables (4 real + 2 views + 1 link table) per plan A2.
-- _schema_migrations is infrastructure managed by db.go, not listed here.

-- sessions: one row per work session.
-- agent_detected and agent_override are SEPARATE nullable columns (plan Principle 2, A2 spec).
-- agent_id = effective agent (agent_override when set, else agent_detected).
CREATE TABLE IF NOT EXISTS sessions (
    id             TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    started_at     TEXT NOT NULL,  -- ISO 8601, e.g. 2026-05-20T03:24:31Z
    ended_at       TEXT,           -- ISO 8601; NULL while session is active
    mode           TEXT NOT NULL DEFAULT 'solo',
    agent_id       TEXT,           -- effective agent: override ?? detected ?? NULL
    agent_detected TEXT,           -- raw auto-detected agent; nullable
    agent_override TEXT,           -- explicit --agent flag value; nullable
    title          TEXT,
    focus          TEXT,
    next_step      TEXT
);

-- turns: sub-units within a session.
CREATE TABLE IF NOT EXISTS turns (
    id          TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    sequence_no INTEGER NOT NULL,  -- monotonically increasing within session (1-based)
    title       TEXT,
    started_at  TEXT NOT NULL,     -- ISO 8601
    ended_at    TEXT,              -- ISO 8601; NULL while turn is active
    status      TEXT NOT NULL DEFAULT 'active',  -- active|complete|abort|abandon
    elapsed_ms  INTEGER,           -- NULL until turn ends
    agent_id    TEXT
);

-- entries: the primary log record (entry|decision|evidence|blocker|mode).
-- turn_id is nullable: session-level entries (e.g. mode changes) have no turn.
CREATE TABLE IF NOT EXISTS entries (
    id         TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    turn_id    TEXT REFERENCES turns(id) ON DELETE SET NULL,  -- nullable
    kind       TEXT NOT NULL CHECK (kind IN ('entry','decision','evidence','blocker','mode')),
    title      TEXT NOT NULL,
    detail     TEXT,       -- multi-line body; newlines/indentation preserved verbatim
    created_at TEXT NOT NULL,  -- ISO 8601
    agent_id   TEXT
);

-- blockers: tracks a blocking issue tied to a turn/entry.
CREATE TABLE IF NOT EXISTS blockers (
    id                  TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    turn_id             TEXT REFERENCES turns(id) ON DELETE SET NULL,
    entry_id            TEXT REFERENCES entries(id) ON DELETE CASCADE,
    title               TEXT NOT NULL,
    opened_at           TEXT NOT NULL,  -- ISO 8601
    closed_at           TEXT,           -- ISO 8601; NULL while blocker is open
    status              TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','resolved')),
    accumulated_seconds INTEGER NOT NULL DEFAULT 0
);

-- decisions: view over entries WHERE kind='decision'.
-- entry_id PK is inherited from entries.id via the view.
CREATE VIEW IF NOT EXISTS decisions AS
    SELECT * FROM entries WHERE kind = 'decision';

-- evidence: view over entries WHERE kind='evidence'.
CREATE VIEW IF NOT EXISTS evidence AS
    SELECT * FROM entries WHERE kind = 'evidence';

-- decision_evidence_links: many-to-many between decisions and evidence entries.
CREATE TABLE IF NOT EXISTS decision_evidence_links (
    decision_entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    evidence_entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    created_at        TEXT NOT NULL,  -- ISO 8601
    note              TEXT,
    PRIMARY KEY (decision_entry_id, evidence_entry_id)
);

-- Indexes for common query patterns.
CREATE INDEX IF NOT EXISTS idx_turns_session   ON turns(session_id, sequence_no);
CREATE INDEX IF NOT EXISTS idx_entries_session ON entries(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_entries_turn    ON entries(turn_id);
CREATE INDEX IF NOT EXISTS idx_entries_kind    ON entries(kind);
CREATE INDEX IF NOT EXISTS idx_blockers_turn   ON blockers(turn_id);
