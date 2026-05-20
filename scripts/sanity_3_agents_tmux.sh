#!/usr/bin/env bash
# Run a local Phase E0 sanity check for tmux + Claude/Codex/Gemini CLIs.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-${ROOT}/docs/e0-3-agent-tmux-sanity.md}"
TMP="${TMPDIR:-/tmp}/flightlog-e0-agents-$$"
SESSION="flightlog-e0-$$"
mkdir -p "$TMP" "$(dirname "$OUT")"

cleanup() {
  tmux kill-session -t "$SESSION" >/dev/null 2>&1 || true
  rm -rf "$TMP"
}
trap cleanup EXIT

status_for() {
  local agent="$1"
  local out="$TMP/$agent.out"
  local code="$TMP/$agent.code"
  if ! command -v "$agent" >/dev/null 2>&1; then
    printf '| `%s` | missing | not on PATH | |\n' "$agent"
    return
  fi

  local bin
  bin="$(command -v "$agent")"
  tmux split-window -t "$SESSION:0" -h "bash" "-lc" \
    "perl -e '\$SIG{ALRM}=sub{exit 124}; alarm shift; system @ARGV; exit(\$? >> 8)' 8 '$bin' --version >'$out' 2>&1; printf '%s' \$? >'$code'"

  local deadline=$((SECONDS + 12))
  while [[ ! -f "$code" && "$SECONDS" -lt "$deadline" ]]; do
    sleep 0.2
  done

  if [[ ! -f "$code" ]]; then
    printf '| `%s` | timeout | `%s` | version smoke did not finish |\n' "$agent" "$bin"
    return
  fi

  local rc first
  rc="$(cat "$code")"
  first="$(head -n 1 "$out" | tr '|' '/' | sed 's/[[:cntrl:]]//g')"
  [[ -n "$first" ]] || first="(no output)"
  if [[ "$rc" == "0" ]]; then
    printf '| `%s` | pass | `%s` | `%s` |\n' "$agent" "$bin" "$first"
  else
    printf '| `%s` | fail(%s) | `%s` | `%s` |\n' "$agent" "$rc" "$bin" "$first"
  fi
}

{
  echo "# E0 3-Agent tmux Sanity"
  echo
  echo "Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo
  if ! command -v tmux >/dev/null 2>&1; then
    echo "Result: FAIL - tmux not on PATH"
    exit 1
  fi
  echo "- tmux: $(tmux -V)"
  echo "- repository: $ROOT"
  echo
  echo "| Agent CLI | Status | Path | Evidence |"
  echo "| --- | --- | --- | --- |"
} > "$OUT"

tmux new-session -d -s "$SESSION" "bash" "-lc" "sleep 60"
for agent in claude codex gemini; do
  status_for "$agent" >> "$OUT"
done

cat >> "$OUT" <<'EOF'

## Interpretation

- `pass` means the CLI exists and returned successfully from `--version` inside a tmux pane.
- `missing` means install or PATH setup is still incomplete.
- `fail` or `timeout` means the CLI exists but the noninteractive version smoke is not currently usable.
EOF

cat "$OUT"
