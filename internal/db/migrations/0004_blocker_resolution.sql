-- 0004_blocker_resolution.sql: store human-readable blocker resolution notes.

ALTER TABLE blockers ADD COLUMN resolution_note TEXT;
