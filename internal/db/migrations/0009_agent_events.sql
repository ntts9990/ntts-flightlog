-- 0009_agent_events.sql: Redacted hook/event ingest audit table.

CREATE TABLE IF NOT EXISTS agent_events (
    id                  TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    session_id          TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    turn_id             TEXT REFERENCES turns(id) ON DELETE SET NULL,
    source              TEXT NOT NULL,
    event_name          TEXT NOT NULL,
    event_time          TEXT NOT NULL,
    summary             TEXT NOT NULL,
    severity            TEXT,
    dedupe_key          TEXT UNIQUE,
    promotion_status    TEXT NOT NULL DEFAULT 'none' CHECK (promotion_status IN ('none','candidate','promoted','rejected','duplicate')),
    promoted_entry_id   TEXT REFERENCES entries(id) ON DELETE SET NULL,
    redaction_version   TEXT NOT NULL,
    dropped_field_count INTEGER NOT NULL DEFAULT 0,
    rejected_reason     TEXT,
    command_summary     TEXT,
    exit_code           INTEGER,
    duration_ms         INTEGER,
    lane                TEXT
);

CREATE INDEX IF NOT EXISTS idx_agent_events_session ON agent_events(session_id, event_time);
CREATE INDEX IF NOT EXISTS idx_agent_events_turn ON agent_events(turn_id);
CREATE INDEX IF NOT EXISTS idx_agent_events_source ON agent_events(source, event_name);
