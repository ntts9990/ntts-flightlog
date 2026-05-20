#!/usr/bin/env bash
# Validate the Phase E alpha dogfood journal structure.
set -euo pipefail

DOC="${1:-.omc/specs/alpha-dogfood-log.md}"
STRICT=0
if [[ "${2:-}" == "--strict" ]]; then
  STRICT=1
fi

[[ -f "$DOC" ]] || { echo "missing alpha journal: $DOC" >&2; exit 1; }

required=(
  "Self-retro"
  "CHANGED-BY-METRIC"
  "turn 소요시간|turn duration"
  "blocker 누적시간|blocker accumulation|차단 시간"
  "agent 완료율|agent completion|완료율"
  "agent blocker 빈도|blocker 빈도"
  "evidence-bound decision|evidence가 붙은"
)

pass=1
for pat in "${required[@]}"; do
  if ! LC_ALL=en_US.UTF-8 grep -Eiq "$pat" "$DOC"; then
    echo "FAIL: missing journal guidance token: $pat"
    pass=0
  fi
done

entry_count="$(grep -Ec '^### [0-9]{4}-[0-9]{2}-[0-9]{2}' "$DOC" || true)"
changed_count="$(awk '
  /^### [0-9]{4}-[0-9]{2}-[0-9]{2}/ { in_entries=1 }
  in_entries && /\[CHANGED-BY-METRIC: [a-z_]+\]/ { count++ }
  END { print count + 0 }
' "$DOC")"

if [[ "$entry_count" -eq 0 ]]; then
  echo "PASS: journal scaffold present; no dated entries yet"
  [[ $pass -eq 1 ]]
  exit
fi

echo "INFO: dated entries=$entry_count changed_by_metric=$changed_count"
if [[ "$STRICT" -eq 1 && "$changed_count" -lt 1 ]]; then
  echo "FAIL: strict mode requires at least one [CHANGED-BY-METRIC: metric_id] entry"
  pass=0
fi

[[ $pass -eq 1 ]]
