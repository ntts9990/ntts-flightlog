-- fixture_10s_3t_47e.sql: B-Exit gate fixture for ntts-flightlog v2 Phase B.
--
-- Counts (verified):
--   10 sessions × 3 turns = 30 turns
--   entries table: 30 kind='entry' + 8 kind='decision' + 6 kind='evidence' + 3 kind='blocker' = 47
--   blockers table: 3 rows (2 resolved, 1 open)
--   decision_evidence_links: 4 rows (4 evidence linked, 2 evidence unlinked)
--
-- Agent split:
--   sessions s01–s08 → agent_id='claude'  (24 turns)
--   sessions s09–s10 → agent_id='codex'   (6 turns)
--
-- Turn status:
--   s01–s07: all 3 turns 'complete' (21 turns)
--   s08: T08_1=complete, T08_2=complete, T08_3=abort (→ claude total: 23 complete + 1 abort = 24)
--   s09: all 3 'complete'
--   s10: T10_1=complete, T10_2=complete, T10_3=abandon (→ codex total: 5 complete + 1 abandon = 6)
--
-- Expected metric values (frozen in testdata/fixture_expected_metrics.json):
--   metric_agent_completion:
--     claude → completion_rate = 23/24 ≈ 0.9583333..., complete_count=23, total_count=24
--     codex  → completion_rate =  5/6  ≈ 0.8333333..., complete_count=5,  total_count=6
--   metric_agent_blocker_freq:
--     claude → blocker_freq = 3/24 = 0.125, blocker_count=3, turn_count=24
--     codex  → blocker_freq = 0/6  = 0.000, blocker_count=0, turn_count=6
--   metric_evidence_bound_decisions:
--     ratio = 4/8 = 0.5, linked_count=4, total_count=8

-- ── SESSIONS ────────────────────────────────────────────────────────────────
INSERT INTO sessions (id, started_at, ended_at, mode, agent_id) VALUES
  ('s01', '2026-01-01T00:00:00Z', '2026-01-01T06:00:00Z', 'solo', 'claude'),
  ('s02', '2026-01-02T00:00:00Z', '2026-01-02T06:00:00Z', 'solo', 'claude'),
  ('s03', '2026-01-03T00:00:00Z', '2026-01-03T06:00:00Z', 'solo', 'claude'),
  ('s04', '2026-01-04T00:00:00Z', '2026-01-04T06:00:00Z', 'solo', 'claude'),
  ('s05', '2026-01-05T00:00:00Z', '2026-01-05T06:00:00Z', 'solo', 'claude'),
  ('s06', '2026-01-06T00:00:00Z', '2026-01-06T06:00:00Z', 'solo', 'claude'),
  ('s07', '2026-01-07T00:00:00Z', '2026-01-07T06:00:00Z', 'solo', 'claude'),
  ('s08', '2026-01-08T00:00:00Z', '2026-01-08T06:00:00Z', 'solo', 'claude'),
  ('s09', '2026-01-09T00:00:00Z', '2026-01-09T06:00:00Z', 'solo', 'codex'),
  ('s10', '2026-01-10T00:00:00Z', '2026-01-10T06:00:00Z', 'solo', 'codex');

-- ── TURNS (30 rows) ─────────────────────────────────────────────────────────
-- elapsed_ms: seq=1→60000(1m), seq=2→120000(2m), seq=3→180000(3m)
INSERT INTO turns (id, session_id, sequence_no, title, started_at, ended_at, status, elapsed_ms, agent_id) VALUES
  -- s01 (claude)
  ('T01_1','s01',1,'Turn 1','2026-01-01T01:00:00Z','2026-01-01T01:01:00Z','complete', 60000,'claude'),
  ('T01_2','s01',2,'Turn 2','2026-01-01T02:00:00Z','2026-01-01T02:02:00Z','complete',120000,'claude'),
  ('T01_3','s01',3,'Turn 3','2026-01-01T03:00:00Z','2026-01-01T03:03:00Z','complete',180000,'claude'),
  -- s02 (claude)
  ('T02_1','s02',1,'Turn 1','2026-01-02T01:00:00Z','2026-01-02T01:01:00Z','complete', 60000,'claude'),
  ('T02_2','s02',2,'Turn 2','2026-01-02T02:00:00Z','2026-01-02T02:02:00Z','complete',120000,'claude'),
  ('T02_3','s02',3,'Turn 3','2026-01-02T03:00:00Z','2026-01-02T03:03:00Z','complete',180000,'claude'),
  -- s03 (claude)
  ('T03_1','s03',1,'Turn 1','2026-01-03T01:00:00Z','2026-01-03T01:01:00Z','complete', 60000,'claude'),
  ('T03_2','s03',2,'Turn 2','2026-01-03T02:00:00Z','2026-01-03T02:02:00Z','complete',120000,'claude'),
  ('T03_3','s03',3,'Turn 3','2026-01-03T03:00:00Z','2026-01-03T03:03:00Z','complete',180000,'claude'),
  -- s04 (claude)
  ('T04_1','s04',1,'Turn 1','2026-01-04T01:00:00Z','2026-01-04T01:01:00Z','complete', 60000,'claude'),
  ('T04_2','s04',2,'Turn 2','2026-01-04T02:00:00Z','2026-01-04T02:02:00Z','complete',120000,'claude'),
  ('T04_3','s04',3,'Turn 3','2026-01-04T03:00:00Z','2026-01-04T03:03:00Z','complete',180000,'claude'),
  -- s05 (claude)
  ('T05_1','s05',1,'Turn 1','2026-01-05T01:00:00Z','2026-01-05T01:01:00Z','complete', 60000,'claude'),
  ('T05_2','s05',2,'Turn 2','2026-01-05T02:00:00Z','2026-01-05T02:02:00Z','complete',120000,'claude'),
  ('T05_3','s05',3,'Turn 3','2026-01-05T03:00:00Z','2026-01-05T03:03:00Z','complete',180000,'claude'),
  -- s06 (claude) — turns that host the 3 blockers
  ('T06_1','s06',1,'Turn 1','2026-01-06T01:00:00Z','2026-01-06T01:01:00Z','complete', 60000,'claude'),
  ('T06_2','s06',2,'Turn 2','2026-01-06T02:00:00Z','2026-01-06T02:02:00Z','complete',120000,'claude'),
  ('T06_3','s06',3,'Turn 3','2026-01-06T03:00:00Z','2026-01-06T03:03:00Z','complete',180000,'claude'),
  -- s07 (claude)
  ('T07_1','s07',1,'Turn 1','2026-01-07T01:00:00Z','2026-01-07T01:01:00Z','complete', 60000,'claude'),
  ('T07_2','s07',2,'Turn 2','2026-01-07T02:00:00Z','2026-01-07T02:02:00Z','complete',120000,'claude'),
  ('T07_3','s07',3,'Turn 3','2026-01-07T03:00:00Z','2026-01-07T03:03:00Z','complete',180000,'claude'),
  -- s08 (claude) — T08_3 aborted
  ('T08_1','s08',1,'Turn 1','2026-01-08T01:00:00Z','2026-01-08T01:01:00Z','complete', 60000,'claude'),
  ('T08_2','s08',2,'Turn 2','2026-01-08T02:00:00Z','2026-01-08T02:02:00Z','complete',120000,'claude'),
  ('T08_3','s08',3,'Turn 3','2026-01-08T03:00:00Z','2026-01-08T03:03:00Z','abort',   180000,'claude'),
  -- s09 (codex) — all complete
  ('T09_1','s09',1,'Turn 1','2026-01-09T01:00:00Z','2026-01-09T01:01:00Z','complete', 60000,'codex'),
  ('T09_2','s09',2,'Turn 2','2026-01-09T02:00:00Z','2026-01-09T02:02:00Z','complete',120000,'codex'),
  ('T09_3','s09',3,'Turn 3','2026-01-09T03:00:00Z','2026-01-09T03:03:00Z','complete',180000,'codex'),
  -- s10 (codex) — T10_3 abandoned
  ('T10_1','s10',1,'Turn 1','2026-01-10T01:00:00Z','2026-01-10T01:01:00Z','complete', 60000,'codex'),
  ('T10_2','s10',2,'Turn 2','2026-01-10T02:00:00Z','2026-01-10T02:02:00Z','complete',120000,'codex'),
  ('T10_3','s10',3,'Turn 3','2026-01-10T03:00:00Z','2026-01-10T03:03:00Z','abandon',  180000,'codex');

-- ── ENTRIES (47 rows) ────────────────────────────────────────────────────────
-- kind='entry' (30 rows): one per turn for all turns, plus one extra per some turns
-- kind='decision' (8 rows): T01_1..T03_2 get a decision each
-- kind='evidence' (6 rows): T04_1..T05_3 get an evidence each
-- kind='blocker' (3 rows): T06_1..T06_3 get a blocker entry each

-- Turns T01_1..T03_2: entry + decision (16 rows for 8 turns → 8 entry + 8 decision)
INSERT INTO entries (id, session_id, turn_id, kind, title, created_at) VALUES
  ('E001','s01','T01_1','entry',   '작업 항목 1',  '2026-01-01T01:00:10Z'),
  ('E002','s01','T01_1','decision','결정 사항 1',  '2026-01-01T01:00:20Z'),
  ('E003','s01','T01_2','entry',   '작업 항목 2',  '2026-01-01T02:00:10Z'),
  ('E004','s01','T01_2','decision','결정 사항 2',  '2026-01-01T02:00:20Z'),
  ('E005','s01','T01_3','entry',   '작업 항목 3',  '2026-01-01T03:00:10Z'),
  ('E006','s01','T01_3','decision','결정 사항 3',  '2026-01-01T03:00:20Z'),
  ('E007','s02','T02_1','entry',   '작업 항목 4',  '2026-01-02T01:00:10Z'),
  ('E008','s02','T02_1','decision','결정 사항 4',  '2026-01-02T01:00:20Z'),
  ('E009','s02','T02_2','entry',   '작업 항목 5',  '2026-01-02T02:00:10Z'),
  ('E010','s02','T02_2','decision','결정 사항 5',  '2026-01-02T02:00:20Z'),
  ('E011','s02','T02_3','entry',   '작업 항목 6',  '2026-01-02T03:00:10Z'),
  ('E012','s02','T02_3','decision','결정 사항 6',  '2026-01-02T03:00:20Z'),
  ('E013','s03','T03_1','entry',   '작업 항목 7',  '2026-01-03T01:00:10Z'),
  ('E014','s03','T03_1','decision','결정 사항 7',  '2026-01-03T01:00:20Z'),
  ('E015','s03','T03_2','entry',   '작업 항목 8',  '2026-01-03T02:00:10Z'),
  ('E016','s03','T03_2','decision','결정 사항 8',  '2026-01-03T02:00:20Z');

-- Turn T03_3: single entry (17th entry)
INSERT INTO entries (id, session_id, turn_id, kind, title, created_at) VALUES
  ('E017','s03','T03_3','entry','작업 항목 9','2026-01-03T03:00:10Z');

-- Turns T04_1..T05_3: entry + evidence (12 rows for 6 turns)
INSERT INTO entries (id, session_id, turn_id, kind, title, created_at) VALUES
  ('E018','s04','T04_1','entry',   '작업 항목 10', '2026-01-04T01:00:10Z'),
  ('E019','s04','T04_1','evidence','근거 자료 1',   '2026-01-04T01:00:20Z'),
  ('E020','s04','T04_2','entry',   '작업 항목 11', '2026-01-04T02:00:10Z'),
  ('E021','s04','T04_2','evidence','근거 자료 2',   '2026-01-04T02:00:20Z'),
  ('E022','s04','T04_3','entry',   '작업 항목 12', '2026-01-04T03:00:10Z'),
  ('E023','s04','T04_3','evidence','근거 자료 3',   '2026-01-04T03:00:20Z'),
  ('E024','s05','T05_1','entry',   '작업 항목 13', '2026-01-05T01:00:10Z'),
  ('E025','s05','T05_1','evidence','근거 자료 4',   '2026-01-05T01:00:20Z'),
  ('E026','s05','T05_2','entry',   '작업 항목 14', '2026-01-05T02:00:10Z'),
  ('E027','s05','T05_2','evidence','근거 자료 5',   '2026-01-05T02:00:20Z'),  -- unlinked
  ('E028','s05','T05_3','entry',   '작업 항목 15', '2026-01-05T03:00:10Z'),
  ('E029','s05','T05_3','evidence','근거 자료 6',   '2026-01-05T03:00:20Z');  -- unlinked

-- Turns T06_1..T06_3: entry + blocker (6 rows for 3 turns)
INSERT INTO entries (id, session_id, turn_id, kind, title, created_at) VALUES
  ('E030','s06','T06_1','entry',  '작업 항목 16','2026-01-06T01:00:10Z'),
  ('E031','s06','T06_1','blocker','블로커 1',    '2026-01-06T01:00:20Z'),
  ('E032','s06','T06_2','entry',  '작업 항목 17','2026-01-06T02:00:10Z'),
  ('E033','s06','T06_2','blocker','블로커 2',    '2026-01-06T02:00:20Z'),
  ('E034','s06','T06_3','entry',  '작업 항목 18','2026-01-06T03:00:10Z'),
  ('E035','s06','T06_3','blocker','블로커 3',    '2026-01-06T03:00:20Z');

-- Turns T07_1..T10_3: 1 entry each (12 rows)
INSERT INTO entries (id, session_id, turn_id, kind, title, created_at) VALUES
  ('E036','s07','T07_1','entry','작업 항목 19','2026-01-07T01:00:10Z'),
  ('E037','s07','T07_2','entry','작업 항목 20','2026-01-07T02:00:10Z'),
  ('E038','s07','T07_3','entry','작업 항목 21','2026-01-07T03:00:10Z'),
  ('E039','s08','T08_1','entry','작업 항목 22','2026-01-08T01:00:10Z'),
  ('E040','s08','T08_2','entry','작업 항목 23','2026-01-08T02:00:10Z'),
  ('E041','s08','T08_3','entry','작업 항목 24','2026-01-08T03:00:10Z'),
  ('E042','s09','T09_1','entry','작업 항목 25','2026-01-09T01:00:10Z'),
  ('E043','s09','T09_2','entry','작업 항목 26','2026-01-09T02:00:10Z'),
  ('E044','s09','T09_3','entry','작업 항목 27','2026-01-09T03:00:10Z'),
  ('E045','s10','T10_1','entry','작업 항목 28','2026-01-10T01:00:10Z'),
  ('E046','s10','T10_2','entry','작업 항목 29','2026-01-10T02:00:10Z'),
  ('E047','s10','T10_3','entry','작업 항목 30','2026-01-10T03:00:10Z');

-- ── BLOCKERS (3 rows) ────────────────────────────────────────────────────────
-- BL1, BL2: resolved; BL3: open (closed_at NULL, accumulated_seconds=0)
INSERT INTO blockers (id, turn_id, entry_id, title, opened_at, closed_at, status, accumulated_seconds) VALUES
  ('BL1','T06_1','E031','블로커 1',
   '2026-01-06T01:00:20Z','2026-01-06T02:00:20Z','resolved',3600),
  ('BL2','T06_2','E033','블로커 2',
   '2026-01-06T02:00:20Z','2026-01-06T02:30:20Z','resolved',1800),
  ('BL3','T06_3','E035','블로커 3',
   '2026-01-06T03:00:20Z', NULL,                 'open',       0);

-- ── DECISION-EVIDENCE LINKS (4 rows, 2 evidence unlinked) ───────────────────
-- Linked: E019→E002, E021→E004, E023→E006, E025→E008
-- Unlinked evidence: E027, E029
INSERT INTO decision_evidence_links (decision_entry_id, evidence_entry_id, created_at) VALUES
  ('E002','E019','2026-01-04T01:00:30Z'),
  ('E004','E021','2026-01-04T02:00:30Z'),
  ('E006','E023','2026-01-04T03:00:30Z'),
  ('E008','E025','2026-01-05T01:00:30Z');
