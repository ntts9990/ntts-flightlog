-- 0006_decision_status.sql: ADR-lite lifecycle state for decision entries.

CREATE TABLE IF NOT EXISTS decision_status (
    decision_entry_id        TEXT PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
    status                   TEXT NOT NULL DEFAULT 'accepted'
                             CHECK (status IN ('accepted','superseded','rejected')),
    superseded_by_entry_id   TEXT REFERENCES entries(id) ON DELETE SET NULL,
    superseded_at            TEXT,
    rationale                TEXT
);

CREATE INDEX IF NOT EXISTS idx_decision_status_status
    ON decision_status(status);
