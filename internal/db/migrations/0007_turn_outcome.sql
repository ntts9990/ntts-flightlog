-- 0007_turn_outcome.sql: persist explicit turn outcomes for the turn index.

ALTER TABLE turns ADD COLUMN outcome TEXT;
