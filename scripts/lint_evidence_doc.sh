#!/usr/bin/env bash
# Lint Phase E GA acceptance evidence without judging the prose.
set -euo pipefail

DOC="${1:-docs/v2-ga-acceptance-evidence.md}"
[[ -f "$DOC" ]] || { echo "missing evidence doc: $DOC" >&2; exit 1; }

sections=(
  "Self-Retro"
  "Agent-Operator"
  "Team-Share"
)
metrics=(
  "turn_duration|turn 소요시간|turn duration|turn elapsed"
  "blocker_accumulation|blocker 누적|blocker accumulation|차단 시간"
  "agent_completion|agent 완료율|agent completion|완료율"
  "agent_blocker_freq|agent blocker 빈도|blocker 빈도"
  "evidence_bound_decisions|evidence-bound|evidence가 붙은"
)

extract_section() {
  local name="$1" in_sec=0
  while IFS= read -r line; do
    if [[ "$line" =~ ^##[[:space:]]+.*$name ]]; then
      in_sec=1
      continue
    fi
    if [[ $in_sec -eq 1 && "$line" =~ ^##[[:space:]]+ ]]; then
      break
    fi
    [[ $in_sec -eq 1 ]] && printf '%s\n' "$line"
  done < "$DOC"
}

pass=1
for section in "${sections[@]}"; do
  text="$(extract_section "$section")"
  if [[ -z "$text" ]]; then
    echo "FAIL $section: section missing"
    pass=0
    continue
  fi
  count=0
  for pat in "${metrics[@]}"; do
    if printf '%s\n' "$text" | LC_ALL=en_US.UTF-8 grep -Eiq "$pat"; then
      count=$((count + 1))
    fi
  done
  if [[ $count -lt 4 ]]; then
    echo "FAIL $section: $count/5 metrics cited, want >=4"
    pass=0
  else
    echo "PASS $section: $count/5 metrics cited"
  fi
done

grep -Eiq "adversarial review" "$DOC" || { echo "FAIL: adversarial review reference missing"; pass=0; }
grep -Eiq "external .*ack|acknowledg" "$DOC" || { echo "FAIL: external ack reference missing"; pass=0; }

[[ $pass -eq 1 ]]
