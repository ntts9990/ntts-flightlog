-- 0003_metric_views.sql: 5 analytic metric SQL views for ntts-flightlog v2 Phase B3.
-- Applied by the migration runner in db.go (lexicographic order after 0002).
-- All views use IF NOT EXISTS so repeated migration is idempotent.

-- metric_turn_duration: per-turn elapsed time with agent attribution.
-- Korean metric: Turn 소요시간 분포 (metric 1).
CREATE VIEW IF NOT EXISTS metric_turn_duration AS
    SELECT
        id         AS turn_id,
        agent_id,
        elapsed_ms
    FROM turns;

-- metric_blocker_accumulation: blocker lifecycle from opened to resolved/open.
-- Korean metric: Blocker 누적시간 (metric 2).
CREATE VIEW IF NOT EXISTS metric_blocker_accumulation AS
    SELECT
        id                   AS blocker_id,
        opened_at,
        closed_at,           -- NULL while blocker is still open
        accumulated_seconds
    FROM blockers;

-- metric_agent_completion: fraction of turns with status='complete' per agent.
-- Denominator includes all statuses (active, complete, abort, abandon).
-- Agents with NULL agent_id are excluded.
-- Korean metric: Agent별 Turn 완료율 (metric 3).
CREATE VIEW IF NOT EXISTS metric_agent_completion AS
    SELECT
        agent_id,
        CAST(SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) AS REAL)
            / COUNT(*)  AS completion_rate,
        SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) AS complete_count,
        COUNT(*)                                               AS total_count
    FROM turns
    WHERE agent_id IS NOT NULL
    GROUP BY agent_id;

-- metric_agent_blocker_freq: average blockers per distinct turn per agent.
-- LEFT JOIN ensures agents with zero blockers still appear with freq=0.
-- Agents with NULL agent_id are excluded.
-- Korean metric: Agent별 Blocker 빈도 (metric 4).
CREATE VIEW IF NOT EXISTS metric_agent_blocker_freq AS
    SELECT
        t.agent_id,
        CAST(COUNT(b.id) AS REAL) / COUNT(DISTINCT t.id) AS blocker_freq,
        COUNT(b.id)                                        AS blocker_count,
        COUNT(DISTINCT t.id)                               AS turn_count
    FROM turns t
    LEFT JOIN blockers b ON b.turn_id = t.id
    WHERE t.agent_id IS NOT NULL
    GROUP BY t.agent_id;

-- metric_evidence_bound_decisions: fraction of decisions that have linked evidence.
-- Returns a single aggregate row. Ratio is 0.0 when there are no decisions.
-- Korean metric: Evidence가 붙은 Decision 비율 (metric 5).
CREATE VIEW IF NOT EXISTS metric_evidence_bound_decisions AS
    SELECT
        CASE WHEN COUNT(DISTINCT e.id) = 0 THEN 0.0
             ELSE CAST(COUNT(DISTINCT del.decision_entry_id) AS REAL)
                  / COUNT(DISTINCT e.id)
        END                                     AS ratio,
        COUNT(DISTINCT del.decision_entry_id)   AS linked_count,
        COUNT(DISTINCT e.id)                    AS total_count
    FROM entries e
    LEFT JOIN decision_evidence_links del ON del.decision_entry_id = e.id
    WHERE e.kind = 'decision';
