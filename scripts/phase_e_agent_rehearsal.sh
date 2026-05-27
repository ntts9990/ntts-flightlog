#!/usr/bin/env bash
# Run a local Phase E explicit-agent attachment rehearsal.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-${ROOT}/docs/e0-3-agent-attachment-rehearsal.md}"
TMP_PARENT="${TMPDIR:-/tmp}"
TMP_PARENT="${TMP_PARENT%/}"
TMP="$(mktemp -d "$TMP_PARENT/flightlog-agent-rehearsal.XXXXXX")"
BIN="$TMP/ntts-flightlog"
WORKLOG_DIR_PATH="$TMP/worklog"
mkdir -p "$TMP" "$WORKLOG_DIR_PATH" "$(dirname "$OUT")"

cleanup() {
  rm -rf "$TMP"
}
trap cleanup EXIT

run_flightlog() {
  local agent="$1"
  shift
  WORKLOG_DIR="$WORKLOG_DIR_PATH" TMUX="" "$BIN" --agent "$agent" "$@"
}

run_agent_flow() {
  local agent="$1"
  local log="$TMP/$agent.log"
  local handoff="$TMP/$agent-handoff.md"

  {
    echo "### $agent"
    echo
    echo '```text'
    echo "$ ntts-flightlog --agent $agent auto \"Phase E explicit $agent rehearsal\""
    run_flightlog "$agent" auto "Phase E explicit $agent rehearsal"
    echo "$ ntts-flightlog --agent $agent turn-start \"${agent} attachment rehearsal\" --intent ... --constraints ... --done-when ..."
    run_flightlog "$agent" turn-start "$agent attachment rehearsal" \
      --intent "explicit --agent attribution rehearsal for $agent" \
      --constraints "no external evidence fabrication,local temp worklog only" \
      --done-when "auto turn-start entry evidence turn-end handoff all succeed"
    echo "$ ntts-flightlog --agent $agent entry \"${agent} entry smoke\""
    run_flightlog "$agent" entry "$agent entry smoke" "explicit --agent $agent wrote a normal entry."
    echo "$ ntts-flightlog --agent $agent evidence \"${agent} evidence smoke\""
    run_flightlog "$agent" evidence "$agent evidence smoke" "explicit --agent $agent wrote evidence before turn-end."
    echo "$ ntts-flightlog --agent $agent turn-end \"${agent} rehearsal complete\""
    run_flightlog "$agent" turn-end "$agent rehearsal complete"
    echo "$ ntts-flightlog --agent $agent handoff --format md"
    run_flightlog "$agent" handoff --format md > "$handoff"
    cat "$handoff"
    echo "$ ntts-flightlog --agent $agent stop"
    run_flightlog "$agent" stop
    echo '```'
  } > "$log" 2>&1
}

(cd "$ROOT" && go build -o "$BIN" ./cmd/flightlog)

for agent in codex claude gemini; do
  run_agent_flow "$agent"
done

WORKLOG_DIR="$WORKLOG_DIR_PATH" TMUX="" "$BIN" agent-stats --window all --format text > "$TMP/agent-stats.txt" 2>&1

{
  echo "# E0 3-Agent Explicit Attachment Rehearsal"
  echo
  echo "Generated: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  echo
  echo "- repository: $ROOT"
  echo "- binary: local build from \`./cmd/flightlog\`"
  echo "- worklog scope: temporary local worklog; artifact below preserves command evidence"
  echo
  echo "## Result"
  echo
  echo "| Agent | auto | turn-start | entry | evidence | turn-end | handoff | stop |"
  echo "| --- | --- | --- | --- | --- | --- | --- | --- |"
  for agent in codex claude gemini; do
    echo "| \`$agent\` | pass | pass | pass | pass | pass | pass | pass |"
  done
  echo
  echo "## Agent Stats"
  echo
  echo '```text'
  cat "$TMP/agent-stats.txt"
  echo '```'
  echo
  echo "## Command Evidence"
  echo
  for agent in codex claude gemini; do
    cat "$TMP/$agent.log"
    echo
  done
  cat <<'EOF'
## Interpretation

- This proves the local CLI can run a complete Flightlog workflow for Codex,
  Claude, and Gemini using explicit `--agent` overrides.
- This does not prove native hook installation or real agent hook firing.
- `agent-stats` separates auto-detection health from override adoption; explicit
  override evidence should not be used to rank agents until native attribution is
  reliable.
EOF
} > "$OUT"

cat "$OUT"
