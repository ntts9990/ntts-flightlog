-- 0008_lane_tracking.sql: Optional lane/team attribution.

ALTER TABLE turns ADD COLUMN lane TEXT;
ALTER TABLE turns ADD COLUMN parent_turn_id TEXT;
ALTER TABLE entries ADD COLUMN lane TEXT;

CREATE INDEX IF NOT EXISTS idx_turns_lane ON turns(lane, status, started_at);
CREATE INDEX IF NOT EXISTS idx_entries_lane ON entries(lane, created_at);
