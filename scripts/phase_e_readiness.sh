#!/usr/bin/env bash
# Summarize Phase E evidence readiness. Default mode is advisory; --strict is GA-blocking.
set -euo pipefail

STRICT=0
if [[ "${1:-}" == "--strict" ]]; then
  STRICT=1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

failures=0
warn() {
  printf 'WARN: %s\n' "$1"
}
fail() {
  printf 'FAIL: %s\n' "$1"
  failures=$((failures + 1))
}
pass() {
  printf 'PASS: %s\n' "$1"
}

check_file() {
  local path="$1"
  if [[ -f "$path" ]]; then
    pass "$path exists"
  else
    fail "$path missing"
  fi
}

echo "Phase E readiness"
echo "mode: $([[ $STRICT -eq 1 ]] && echo strict || echo advisory)"
echo

check_file ".omc/specs/alpha-dogfood-log.md"
check_file ".omc/specs/v2-agent-operator-decisions.md"
check_file ".omc/specs/v2-team-share-report.md"
check_file ".omc/specs/v2-adversarial-review.md"
check_file "docs/v2-ga-acceptance-evidence.md"
check_file "docs/phase-e-persona-recruitment.md"
check_file "docs/adversarial-review-framework.md"
check_file "docs/e0-3-agent-tmux-sanity.md"

echo
if bash scripts/lint_alpha_journal.sh .omc/specs/alpha-dogfood-log.md; then
  pass "alpha journal scaffold lint"
else
  fail "alpha journal scaffold lint"
fi

if bash scripts/lint_evidence_doc.sh docs/v2-ga-acceptance-evidence.md; then
  pass "acceptance evidence structure lint"
else
  fail "acceptance evidence structure lint"
fi

echo
for agent in claude codex gemini; do
  if grep -Eq "\\| \`$agent\` \\| pass \\|" docs/e0-3-agent-tmux-sanity.md 2>/dev/null; then
    pass "E0 live tmux sanity for $agent"
  else
    fail "E0 live tmux sanity for $agent is not pass"
  fi
done

phase_e_sources=(
  "docs/v2-ga-acceptance-evidence.md"
  ".omc/specs/alpha-dogfood-log.md"
  ".omc/specs/v2-agent-operator-decisions.md"
  ".omc/specs/v2-team-share-report.md"
  ".omc/specs/v2-adversarial-review.md"
)
# Zero TODO/placeholder matches is the normal (good) path. Under `set -euo
# pipefail`, grep's exit 1 (no match) would otherwise propagate as the
# pipeline's status even though wc/tr succeed, aborting the script here.
# `|| true` neutralizes that so a clean zero count doesn't trip set -e.
todo_count="$(grep -RInE 'TODO|_to be filled|placeholder' "${phase_e_sources[@]}" 2>/dev/null | wc -l | tr -d ' ' || true)"
entry_count="$(grep -Ec '^### [0-9]{4}-[0-9]{2}-[0-9]{2}' .omc/specs/alpha-dogfood-log.md 2>/dev/null || true)"
changed_count="$(awk '
  /^### [0-9]{4}-[0-9]{2}-[0-9]{2}/ { in_entries=1 }
  in_entries && /\[CHANGED-BY-METRIC: [a-z_]+\]/ { count++ }
  END { print count + 0 }
' .omc/specs/alpha-dogfood-log.md 2>/dev/null || true)"

echo
echo "evidence_todo_count: $todo_count"
echo "alpha_dated_entry_count: $entry_count"
echo "alpha_changed_by_metric_count: $changed_count"

if [[ "$todo_count" -gt 0 ]]; then
  if [[ "$STRICT" -eq 1 ]]; then
    fail "TODO/placeholder evidence remains"
  else
    warn "TODO/placeholder evidence remains; this is expected before real Phase E data exists"
  fi
fi

if [[ "$STRICT" -eq 1 ]]; then
  [[ "$entry_count" -ge 12 ]] || fail "strict mode wants at least 12 dated alpha entries"
  [[ "$changed_count" -ge 1 ]] || fail "strict mode wants at least one CHANGED-BY-METRIC entry"
  grep -Eiq 'external .*ack|acknowledg' docs/v2-ga-acceptance-evidence.md || fail "strict mode wants external acknowledgement evidence"
  grep -Eiq 'adversarial review' docs/v2-ga-acceptance-evidence.md || fail "strict mode wants adversarial review evidence"
fi

echo
if [[ "$failures" -eq 0 ]]; then
  pass "Phase E readiness checks completed"
  exit 0
fi

echo "readiness_failures: $failures"
exit 1
