#!/usr/bin/env bash
# scripts/extract_citations.sh — B.5.2 Pre-registered citation extractor.
#
# FROZEN: Regex patterns registered BEFORE v2-retro-rehearsal.md was authored.
# Registration timestamp: 2026-05-20T05:10:00Z
# Do NOT modify METRIC_PATTERNS after first commit — post-hoc regex tuning
# defeats the spontaneous-citation gate (B.5 guardrail).
#
# Usage:  bash scripts/extract_citations.sh [path/to/v2-retro-rehearsal.md]
# Output: per-section citation count + matched metric list, then PASS/FAIL gate.
# Exit 0 = PASS (all 3 sections ≥3 distinct metrics)
# Exit 1 = FAIL (any section <3 metrics → metric redesign required before Phase C)

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOC="${1:-${PROJECT_ROOT}/.omc/specs/v2-retro-rehearsal.md}"

if [[ ! -f "$DOC" ]]; then
  echo "ERROR: rehearsal doc not found: $DOC" >&2
  echo "       Expected: .omc/specs/v2-retro-rehearsal.md" >&2
  exit 1
fi

# ── FROZEN METRIC PATTERNS (B.5.2, registered 2026-05-20T05:10:00Z) ─────────
# Patterns: Korean + English aliases per metric, ERE syntax for grep -E.
# Source of patterns: B.5 task spec (keyword regex section).
METRIC_NAMES=(
  "turn_duration"
  "blocker_accumulation"
  "agent_completion"
  "agent_blocker_freq"
  "evidence_bound_decisions"
)
METRIC_PATTERNS=(
  "turn 소요시간|turn duration|turn elapsed|소요"
  "blocker 누적|blocker accumulation|차단 시간"
  "agent.{0,5}완료율|agent completion|완료율"
  "agent.{0,5}blocker.{0,5}빈도|blocker 빈도"
  "evidence.{0,5}decision|evidence-bound|evidence가 붙은"
)
# ── END FROZEN PATTERNS ───────────────────────────────────────────────────────

GATE_THRESHOLD=3  # rehearsal threshold (Phase E real gate uses ≥4)

# Extract text content for Section N.
# Reads from "## Section N" header up to the next "## " header or EOF.
get_section() {
  local n=$1
  local in_sec=0
  while IFS= read -r line; do
    # Detect start of our section (## Section N followed by space or end of line)
    if echo "$line" | LC_ALL=en_US.UTF-8 grep -qE "^## Section ${n}([^0-9]|$)"; then
      in_sec=1
      continue
    fi
    # Detect start of next top-level section → stop
    if [[ $in_sec -eq 1 ]] && echo "$line" | LC_ALL=en_US.UTF-8 grep -qE "^## "; then
      break
    fi
    if [[ $in_sec -eq 1 ]]; then
      printf '%s\n' "$line"
    fi
  done < "$DOC"
}

# Check if a string matches a pattern (case-insensitive, ERE, UTF-8)
matches() {
  local text="$1"
  local pat="$2"
  echo "$text" | LC_ALL=en_US.UTF-8 grep -qEi "$pat"
}

echo "═══════════════════════════════════════════════════════════════"
echo "  B.5 Citation Extractor — v2-retro-rehearsal.md"
echo "  Gate: ≥${GATE_THRESHOLD} distinct metrics per section (rehearsal threshold)"
echo "  Doc:  $DOC"
echo "═══════════════════════════════════════════════════════════════"

GATE_PASS=1
declare -a SECTION_RESULTS=()

for sec_n in 1 2 3; do
  sec_text="$(get_section "$sec_n")"

  if [[ -z "$sec_text" ]]; then
    echo ""
    echo "── Section ${sec_n} ──────────────────────────────────────────────────"
    echo "  ERROR: section not found in doc — check ## Section ${sec_n} heading"
    GATE_PASS=0
    SECTION_RESULTS+=("0")
    continue
  fi

  echo ""
  echo "── Section ${sec_n} ──────────────────────────────────────────────────"

  cited_metrics=()
  for i in "${!METRIC_NAMES[@]}"; do
    name="${METRIC_NAMES[$i]}"
    pat="${METRIC_PATTERNS[$i]}"
    if matches "$sec_text" "$pat"; then
      cited_metrics+=("$name")
      echo "  ✓  $name"
    else
      echo "  ✗  $name"
    fi
  done

  count="${#cited_metrics[@]}"
  SECTION_RESULTS+=("$count")
  echo "  ──"
  echo "  Cited: ${count}/5 distinct metrics"

  if [[ $count -lt $GATE_THRESHOLD ]]; then
    echo "  ⚠  FAIL: <${GATE_THRESHOLD} metrics → metric redesign required before Phase C"
    GATE_PASS=0
  else
    echo "  ✓  PASS (${count} ≥ ${GATE_THRESHOLD})"
  fi
done

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Section results: S1=${SECTION_RESULTS[0]:-?}/5  S2=${SECTION_RESULTS[1]:-?}/5  S3=${SECTION_RESULTS[2]:-?}/5"
echo "  Threshold: ≥${GATE_THRESHOLD} per section"

if [[ $GATE_PASS -eq 1 ]]; then
  echo "  GATE RESULT: ✓ PASS — all 3 sections ≥${GATE_THRESHOLD} distinct metrics"
  echo "               Phase E real gate uses ≥4; consider metric clarity if <4 in any section."
  echo "═══════════════════════════════════════════════════════════════"
  exit 0
else
  echo "  GATE RESULT: ✗ FAIL — redesign required (see above)"
  echo "               SendMessage to worker-metrics with redesign request before Phase C."
  echo "═══════════════════════════════════════════════════════════════"
  exit 1
fi
