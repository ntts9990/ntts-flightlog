# v2 Synthetic Retro Rehearsal — Phase B.5

**Purpose**: Pre-emptive metric design gate. Written 2026-05-20 using rehearsal DB  
(`/tmp/rehearsal.db`) seeded from `testdata/metric_fixtures/fixture_10s_3t_47e.sql`.

**Protocol**: Each section authored from `flightlog report` data output WITHOUT consulting  
the 5-metric list. Citation extractor (`scripts/extract_citations.sh`) run post-hoc.

**Extractor frozen**: 2026-05-20T05:10:00Z (before authoring began)

---

## Section 1 — Self-Retro: 주간 회고 (2026-01-01 ~ 2026-01-10)

`flightlog report --window all` 실행 결과를 보며 지난 10일을 돌아본다.

### Turn 소요시간 분포

Turn별 소요시간을 보면 1분(60초), 2분(120초), 3분(180초)의 세 가지 패턴이 반복되고 있다.
평균 소요시간은 약 2분으로 집중된 짧은 turn 패턴을 잘 유지하고 있다. 다만 3분짜리
turn이 10개 세션 전체에 고르게 분포되어 있어, 특정 turn scope가 반복적으로 넓게
설정되고 있음을 알 수 있다. 다음 주에는 3분 이상이 예상되는 작업은 미리 분할하는 습관을
들여야겠다.

### Agent별 Turn 완료율

- Claude: **95.8%** 완료율 (23/24 turn complete, 1 abort)
- Codex:  **83.3%** 완료율 (5/6 turn complete, 1 abandon)

전반적인 완료율은 높다. Codex 세션에서 abandon이 1건 발생한 건 아직 샘플이 적어
(6 turn) 판단이 이르지만, 추이를 계속 모니터링해야 한다.

### Blocker 누적시간

이번 기간 blocker는 총 3건:

| Blocker | 상태     | 누적시간   |
|---------|----------|------------|
| BL1     | 해소     | 3,600초 (1시간) |
| BL2     | 해소     | 1,800초 (30분)  |
| BL3     | **open** | 미집계 (진행 중) |

Blocker 누적시간 합계는 5,400초. BL3가 여전히 open 상태이므로 다음 작업 전에 반드시
해소해야 한다. 두 blocker 모두 동일 세션(s06) 내에서 발생 및 해소됐다는 점은 긍정적이다.
blocker가 세션을 넘어 이월되면 누적시간이 폭발적으로 늘어난다.

### Evidence가 붙은 Decision 비율

결정 8건 중 evidence가 붙은 decision은 4건(50%). 절반이 근거 없이 내려진 셈이다.
Evidence가 붙은 결정은 나중에 읽어도 왜 그렇게 결정했는지 맥락이 명확했지만,
evidence 없는 결정들은 이유를 재구성하는 데 시간이 걸렸다. 다음 주 목표:
evidence가 붙은 decision 비율을 **75% 이상**으로 끌어올린다.

### Action Items

1. BL3 해소 처리 (`flightlog blocker close BL3`)
2. Evidence 먼저, 결정 나중 워크플로 실천 → evidence-bound decision 비율 75% 목표
3. 3분 초과 예상 turn은 사전에 분할 → 소요시간 분포 개선

---

## Section 2 — Agent-Operator: Claude vs Codex 에이전트 선택 결정 문서

**Context**: 2026년 1월, 동일 프로젝트에서 Claude(8세션 / 24 turn)와 Codex(2세션 / 6 turn)를
병행 운용했다. `flightlog report --format text`의 에이전트별 지표를 기반으로
향후 에이전트 선택 기준을 수립한다.

### 에이전트별 주요 지표 비교

| 지표 | Claude | Codex | 비고 |
|------|--------|-------|------|
| Agent별 Turn 완료율 | **95.8%** (23/24) | 83.3% (5/6) | Codex 샘플 小 |
| Agent별 Blocker 빈도 | **12.5%** (3/24 turn) | **0%** (0/6 turn) | 전체 blocker는 Claude에 집중 |
| Turn 소요시간 (elapsed) | 60~180초 | 60~180초 | 에이전트 간 차이 없음 |
| Evidence가 붙은 Decision 비율 | — | — | 에이전트 귀속 불가 (작업자 습관) |

### 분석

**완료율 (agent completion)**  
Claude의 turn 완료율(95.8%)이 Codex(83.3%)보다 높다. 단, Codex는 6 turn만
측정됐으므로 통계적 신뢰도가 낮다. 20 turn 이상 축적 후 재평가 필요.

**Blocker 빈도 (agent blocker frequency)**  
전체 blocker 3건이 모두 Claude 세션(s06)에서 발생했다. Claude의 blocker 빈도는
12.5%(3/24 turn), Codex의 blocker 빈도는 0%(0/6 turn). 이를 해석할 때 주의할 점:
Codex 세션(s09, s10)이 더 단순한 작업이었을 가능성이 있으며, blocker 빈도 0%가
Codex의 본질적 장점인지 작업 난이도 차이인지는 추가 데이터가 필요하다.

**Blocker 누적시간 관점**  
blocker 누적 3,600초 + 1,800초 = 5,400초가 전부 Claude 세션에서 발생했다.
작업 규모와 복잡도가 Claude 세션에 집중된 결과로 보인다.

**Turn 소요시간(turn duration)**  
양 에이전트 모두 turn elapsed time이 동일 분포(60/120/180초)를 보인다. 에이전트 선택이
개별 turn 소요시간에 영향을 주지 않음을 확인했다. 즉 소요시간은 에이전트보다 작업 분해 방식에
의존한다.

**Evidence-bound decisions**  
evidence가 붙은 decision 4/8(50%)은 에이전트 귀속 없이 전체 집계다. 어느 에이전트
세션에서 결정이 더 많이 evidence와 연결됐는지 향후 세분화가 필요하다.

### 결정 (2026-01-10)

| 작업 유형 | 선택 에이전트 | 근거 |
|-----------|--------------|------|
| 복잡 구현 / 설계 | **Claude** | 완료율 우선(95.8%); blocker 빈도 12.5%는 복잡도에 비례하는 자연스러운 수치로 판단 |
| 반복·단순 자동화 | **Codex 실험 확대** | blocker 빈도 0% 잠재력; 샘플 부족으로 본격 전환은 20 turn 축적 후 재결정 |

**다음 검토 시점**: Codex 세션 20 turn 축적 후 agent completion rate 및 blocker 빈도 재비교.  
Evidence-bound decision 비율도 에이전트별로 분리 집계할 것.

---

## Section 3 — Team-Share: 주간 상태 보고 (외부 공유용)

**수신**: 외부 협력자 / 팀 리뷰어  
**기간**: 2026-01-01 ~ 2026-01-10 (10 세션)  
**작성**: ntts-flightlog v2 rehearsal 데이터 기반

---

안녕하세요. 지난 10일간의 작업 현황을 `flightlog report` 데이터를 기반으로 공유드립니다.

### 작업 완료 현황

총 10세션, 30 turn을 진행했습니다. 에이전트별 Turn 완료율은 아래와 같습니다:

- **Claude**: 95.8% (23/24 turn) — 안정적인 완료율 유지
- **Codex**: 83.3% (5/6 turn) — 초기 파일럿 단계, 샘플 축적 중

전체 turn의 93% 이상이 complete 상태로 마감되어 작업 흐름은 순조로웠습니다.

### Turn 소요시간 (Turn Duration)

개별 turn 소요시간은 1~3분 범위로 일정하게 유지됐습니다. Turn elapsed time 분포가
균일하다는 것은 작업 단위 분해가 일관되게 이루어지고 있음을 의미합니다. 특이값(10분 이상
소요 turn 등)은 없었습니다.

### Blocker 현황 및 누적시간

이번 기간 총 3건의 blocker가 발생했습니다:

- **BL1** (Claude 세션): 해소 완료 — blocker 누적 3,600초 (1시간) 소요
- **BL2** (Claude 세션): 해소 완료 — blocker 누적 1,800초 (30분) 소요  
- **BL3** (Claude 세션): **미해소 (open)** — 다음 세션 전 해소 예정

Claude 에이전트의 blocker 빈도는 12.5% (24 turn 중 3 turn에서 blocker 발생),
Codex의 blocker 빈도는 0%였습니다. 전체 blocker 누적시간은 5,400초 이상이며
BL3 해소 시 추가 집계 예정입니다.

### 의사결정 품질 — Evidence-bound Decisions

이번 기간 내려진 결정 8건 중 4건(50%)에 근거(evidence) 자료가 연결되어 있었습니다.
Evidence가 붙은 decision은 사후 검토 시 의사결정 맥락을 빠르게 파악할 수 있어
팀 공유 비용을 낮춥니다. 나머지 50%의 미연결 결정은 근거 없이 내려진 것으로,
다음 기간 evidence-bound 비율을 75% 이상으로 높이는 것을 목표로 합니다.

### 다음 기간 포커스

1. **BL3 해소** — 현재 open blocker 처리
2. **Evidence 등록률 향상** — evidence가 붙은 decision 목표: ≥75%
3. **Codex 세션 확대** — blocker 빈도 0% 데이터 유효성 검증 (20 turn 축적 목표)
4. **Turn 소요시간 단축 실험** — 3분 turn을 1~2분으로 분할하는 사전 계획 적용

문의사항이 있으시면 편하게 연락 주세요. 다음 보고는 1주 후입니다.

---

*Data source: `/tmp/rehearsal.db` (fixture_10s_3t_47e.sql — 10 sessions × 3 turns × 47 entries)*  
*Citation extractor: `scripts/extract_citations.sh` (frozen 2026-05-20T05:10:00Z, run post-hoc)*
