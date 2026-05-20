-- 0005_live_blocker_accumulation.sql: make open blocker accumulation live.

DROP VIEW IF EXISTS metric_blocker_accumulation;

CREATE VIEW metric_blocker_accumulation AS
    SELECT
        id         AS blocker_id,
        opened_at,
        closed_at,
        CASE
            WHEN closed_at IS NULL THEN
                MAX(0, unixepoch('now') - unixepoch(opened_at))
            ELSE accumulated_seconds
        END        AS accumulated_seconds
    FROM blockers;
