-- 0002_turn_anchors.sql: Turn Intent Anchor (TIA) columns on the turns table.
-- Phase A.5 — prevents agent context drift by persisting turn intent across
-- context loss events.
--
-- All new columns are nullable (or DEFAULT 0 for drift_alerts) so existing
-- rows from A1-A5 remain valid; A5's 7 round-trip predicates are unaffected.

ALTER TABLE turns ADD COLUMN intent             TEXT;
ALTER TABLE turns ADD COLUMN constraints_json   TEXT;   -- JSON array of constraint strings
ALTER TABLE turns ADD COLUMN done_when          TEXT;
ALTER TABLE turns ADD COLUMN drift_alerts       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE turns ADD COLUMN anchor_last_shown_at TEXT;  -- ISO 8601; NULL until first refresh
