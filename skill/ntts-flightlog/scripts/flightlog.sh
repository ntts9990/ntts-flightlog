#!/usr/bin/env bash
set -euo pipefail

WORKLOG_DIR="${WORKLOG_DIR:-.omx/worklog}"
WORKLOG_FILE="${WORKLOG_FILE:-${WORKLOG_DIR}/main.md}"
PANE_FILE="${PANE_FILE:-${WORKLOG_DIR}/pane-id}"
SESSION_START_FILE="${SESSION_START_FILE:-${WORKLOG_DIR}/session-start-epoch}"
TURN_START_FILE="${TURN_START_FILE:-${WORKLOG_DIR}/turn-start-epoch}"
TURN_COUNTER_FILE="${TURN_COUNTER_FILE:-${WORKLOG_DIR}/turn-counter}"
MODE_FILE="${MODE_FILE:-${WORKLOG_DIR}/mode}"
REFRESH_SECONDS="${REFRESH_SECONDS:-2}"
PANE_PERCENT="${PANE_PERCENT:-34}"
WORKLOG_RENDERER="${WORKLOG_RENDERER:-color}"

usage() {
  cat <<'USAGE'
Usage:
  worklog_pane.sh start [title]
  worklog_pane.sh auto [title]
  worklog_pane.sh stop
  worklog_pane.sh status <label> [focus] [next]
  worklog_pane.sh mode <solo|ralph|team|plan|review|autopilot|other> [detail]
  worklog_pane.sh turn-start <title>
  worklog_pane.sh turn-end [summary]
  worklog_pane.sh entry <title> [detail]
  worklog_pane.sh decision <title> [detail]
  worklog_pane.sh evidence <title> [detail]
  worklog_pane.sh blocker <title> [detail]
  worklog_pane.sh path

Environment:
  WORKLOG_DIR       default: .omx/worklog
  WORKLOG_FILE      default: .omx/worklog/main.md
  REFRESH_SECONDS   default: 2
  PANE_PERCENT      default: 34
  WORKLOG_RENDERER  default: color; set to glow to use glow when installed
USAGE
}

timestamp() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

epoch_seconds() {
  date -u +"%s"
}

normalize_korean_worklog() {
  [[ -f "${WORKLOG_FILE}" ]] || return 0
  local tmp
  tmp="$(mktemp)"
  awk '
    /^Updated:/ { sub(/^Updated:/, "업데이트:") }
    /^## Current Status$/ { print "## 현재 상태"; next }
    /^## Milestones$/ { print "## 작업 기록"; next }
    /^- State:/ { sub(/^- State:/, "- 상태:") }
    /^- Mode:/ { sub(/^- Mode:/, "- 모드:") }
    /^- Detail:/ { sub(/^- Detail:/, "- 초점:") }
    /^- Focus:/ { sub(/^- Focus:/, "- 초점:") }
    /^- Next:/ { sub(/^- Next:/, "- 다음:") }
    /^- Started:/ { sub(/^- Started:/, "- 시작:") }
    /^- Elapsed:/ { sub(/^- Elapsed:/, "- 경과:") }
    { print }
  ' "${WORKLOG_FILE}" >"${tmp}"
  mv "${tmp}" "${WORKLOG_FILE}"
}

format_duration() {
  local total_seconds="$1"
  local hours=$((total_seconds / 3600))
  local minutes=$(((total_seconds % 3600) / 60))
  local seconds=$((total_seconds % 60))
  if ((hours > 0)); then
    printf '%dh %02dm %02ds' "${hours}" "${minutes}" "${seconds}"
  elif ((minutes > 0)); then
    printf '%dm %02ds' "${minutes}" "${seconds}"
  else
    printf '%ds' "${seconds}"
  fi
}

ensure_worklog() {
  mkdir -p "${WORKLOG_DIR}"
  if [[ ! -f "${WORKLOG_FILE}" ]]; then
    local title="${1:-Codex 작업 기록}"
    cat >"${WORKLOG_FILE}" <<EOF
# ${title}

업데이트: $(timestamp)

## 현재 상태

- 상태: 초기화됨
- 모드: 미지정
- 초점: 작업 기록 pane이 준비되었습니다.
- 다음: 다음 작업 턴을 시작하세요.
- 시작: $(timestamp)

## 작업 기록

EOF
  fi
  if [[ ! -f "${SESSION_START_FILE}" ]]; then
    epoch_seconds >"${SESSION_START_FILE}"
  fi
  normalize_korean_worklog
}

replace_status() {
  local label="$1"
  local focus="${2:-}"
  local next_step="${3:-}"
  ensure_worklog
  local elapsed="unknown"
  local mode="미지정"
  if [[ -f "${MODE_FILE}" ]]; then
    mode="$(cat "${MODE_FILE}")"
  fi
  if [[ -f "${SESSION_START_FILE}" ]]; then
    elapsed="$(format_duration "$(( $(epoch_seconds) - $(cat "${SESSION_START_FILE}") ))")"
  fi
  local tmp
  tmp="$(mktemp)"
  awk \
    -v ts="$(timestamp)" \
    -v label="${label}" \
    -v mode="${mode}" \
    -v focus="${focus}" \
    -v next_step="${next_step}" \
    -v elapsed="${elapsed}" '
    BEGIN { in_status=0; replaced=0 }
    /^(Updated|업데이트):/ { print "업데이트: " ts; next }
    /^## (Current Status|현재 상태)$/ {
      print
      print ""
      print "- 상태: " label
      print "- 모드: " mode
      if (focus != "") {
        print "- 초점: " focus
      }
      if (next_step != "") {
        print "- 다음: " next_step
      }
      print "- 경과: " elapsed
      print ""
      in_status=1
      replaced=1
      next
    }
    in_status && /^## / {
      in_status=0
      print
      next
    }
    in_status { next }
    { print }
    END {
      if (!replaced) {
        print ""
        print "## 현재 상태"
        print ""
        print "- 상태: " label
        print "- 모드: " mode
        if (focus != "") {
          print "- 초점: " focus
        }
        if (next_step != "") {
          print "- 다음: " next_step
        }
        print "- 경과: " elapsed
      }
    }
  ' "${WORKLOG_FILE}" >"${tmp}"
  mv "${tmp}" "${WORKLOG_FILE}"
}

set_mode() {
  local mode="$1"
  local detail="${2:-}"
  ensure_worklog
  printf '%s\n' "${mode}" >"${MODE_FILE}"
  append_entry "mode" "작업 모드: ${mode}" "${detail}"
  replace_status "모드 설정" "현재 작업 모드: ${mode}" "작업 진행 내용을 턴 단위로 기록하세요."
}

append_entry() {
  local kind="$1"
  local title="$2"
  local detail="${3:-}"
  ensure_worklog
  {
    printf '\n### %s [%s] %s\n' "$(timestamp)" "${kind}" "${title}"
    if [[ -n "${detail}" ]]; then
      printf "%s\n" "${detail}"
    fi
  } >>"${WORKLOG_FILE}"
}

next_turn_number() {
  ensure_worklog
  local current="0"
  if [[ -f "${TURN_COUNTER_FILE}" ]]; then
    current="$(cat "${TURN_COUNTER_FILE}")"
  fi
  local next=$((current + 1))
  printf '%s\n' "${next}" >"${TURN_COUNTER_FILE}"
  printf '%s\n' "${next}"
}

turn_start() {
  local title="$1"
  ensure_worklog
  local turn_number
  turn_number="$(next_turn_number)"
  local now
  now="$(epoch_seconds)"
  printf '%s\n' "${now}" >"${TURN_START_FILE}"
  append_entry "turn-${turn_number}-start" "${title}" "시작 시각: $(timestamp)."
  replace_status "turn-${turn_number}" "${title}" "결정과 검증 근거를 진행 중에 기록하세요."
}

turn_end() {
  local summary="${1:-Turn completed.}"
  ensure_worklog
  local turn_number="?"
  if [[ -f "${TURN_COUNTER_FILE}" ]]; then
    turn_number="$(cat "${TURN_COUNTER_FILE}")"
  fi
  local elapsed="unknown"
  if [[ -f "${TURN_START_FILE}" ]]; then
    elapsed="$(format_duration "$(( $(epoch_seconds) - $(cat "${TURN_START_FILE}") ))")"
  fi
  append_entry "turn-${turn_number}-end" "${summary}" "소요 시간: ${elapsed}."
  replace_status "대기" "마지막 턴 ${turn_number} 완료: ${elapsed}." "다음 작업 턴을 기다립니다."
  rm -f "${TURN_START_FILE}"
}

pane_alive() {
  [[ -f "${PANE_FILE}" ]] || return 1
  local pane_id
  pane_id="$(cat "${PANE_FILE}")"
  tmux has-session 2>/dev/null || return 1
  tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qx "${pane_id}"
}

viewer_command() {
  local abs_file
  abs_file="$(cd "$(dirname "${WORKLOG_FILE}")" && pwd)/$(basename "${WORKLOG_FILE}")"
  cat <<EOF
render_worklog() {
  awk '
    BEGIN {
      reset = sprintf("%c[0m", 27)
      bold = sprintf("%c[1m", 27)
      dim = sprintf("%c[2m", 27)
      colors[0] = sprintf("%c[38;5;81m", 27)
      colors[1] = sprintf("%c[38;5;114m", 27)
      colors[2] = sprintf("%c[38;5;215m", 27)
      colors[3] = sprintf("%c[38;5;177m", 27)
      colors[4] = sprintf("%c[38;5;117m", 27)
      entry = 0
    }
    /^# / { print bold colors[0] \$0 reset; next }
    /^## / { print ""; print bold colors[4] \$0 reset; next }
    /^Updated:/ { print dim \$0 reset; next }
    /^### .*\\[/ {
      line = \$0
      sub(/^### /, "", line)
      ts = line
      sub(/ .*/, "", ts)
      kind = line
      sub(/^[^[]*/, "", kind)
      sub(/].*/, "]", kind)
      title = line
      sub(/^[^]]*] /, "", title)
      if (line ~ /\\[turn-[0-9]+-start\\]/) {
        color = colors[entry % 5]
        print color "■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■" reset
        print bold color "▶ " ts "  " kind reset
        print bold color "  " title reset
        print color "────────────────────────────────" reset
        entry++
        next
      }
      if (line ~ /\\[mode\\]/) {
        print bold colors[3] "▣ " ts "  " kind reset
        print bold colors[3] "  " title reset
        entry++
        next
      }
      if (line ~ /\\[entry\\]/) {
        color = colors[entry % 5]
        print bold color "◆ " ts "  " kind reset
        print bold color "  " title reset
        entry++
        next
      }
      if (line ~ /\\[(turn-[0-9]+-end|evidence)\\]/) {
        print bold colors[1] "■ " ts "  " kind reset
        print bold colors[1] "  " title reset
        entry++
        next
      }
      if (line ~ /\\[decision\\]/) {
        print bold colors[2] "◆ " ts "  " kind reset
        print bold colors[2] "  " title reset
        entry++
        next
      }
      if (line ~ /\\[blocker\\]/) {
        red = sprintf("%c[38;5;203m", 27)
        print bold red "!! " ts "  " kind reset
        print bold red "  " title reset
        entry++
        next
      }
    }
    /^[^#[:space:]][^#]*$/ { print dim \$0 reset; next }
    { print }
  ' '${abs_file}'
}

draw_once() {
  printf '\033[H\033[J'
  printf '\033[1;38;5;81m작업 기록 PANE\033[0m  %s\n' '${abs_file}'
  printf '%*s\n' 80 '' | tr ' ' -
  if [[ '${WORKLOG_RENDERER}' == 'glow' ]] && command -v glow >/dev/null 2>&1; then
    glow -s dark '${abs_file}' 2>/dev/null || sed -n '1,240p' '${abs_file}'
  else
    render_worklog | sed -n '1,240p'
  fi
}

draw_once
if command -v fswatch >/dev/null 2>&1; then
  fswatch -0 '${abs_file}' | while IFS= read -r -d '' _event; do
    draw_once
  done
else
  while true; do
    sleep '${REFRESH_SECONDS}'
    draw_once
  done
fi
EOF
}

start_pane() {
  local title="${1:-Codex 작업 기록}"
  ensure_worklog "${title}"
  replace_status "활성" "작업 기록 pane이 표시됩니다." "turn-start로 시간 측정 작업 턴을 시작하세요."
  if ! command -v tmux >/dev/null 2>&1 || [[ -z "${TMUX:-}" ]]; then
    printf '작업 기록 파일: %s\n' "${WORKLOG_FILE}"
    printf '현재 shell에서는 tmux side pane 렌더링을 사용할 수 없습니다.\n'
    return 0
  fi
  if pane_alive; then
    printf '작업 기록 pane이 이미 실행 중입니다: %s\n' "$(cat "${PANE_FILE}")"
    return 0
  fi
  local pane_id
  pane_id="$(
    tmux split-window -h -p "${PANE_PERCENT}" -P -F '#{pane_id}' \
      "bash -lc $(printf '%q' "$(viewer_command)")"
  )"
  printf '%s\n' "${pane_id}" >"${PANE_FILE}"
  printf '작업 기록 pane 시작됨: %s\n' "${pane_id}"
  printf '작업 기록 파일: %s\n' "${WORKLOG_FILE}"
}

auto_start_pane() {
  local title="${1:-Codex 작업 기록}"
  ensure_worklog "${title}"
  if [[ -z "${TMUX:-}" ]]; then
    printf 'tmux가 활성 상태가 아닙니다. 작업 기록 파일은 준비됨: %s\n' "${WORKLOG_FILE}"
    return 0
  fi
  start_pane "${title}"
}

stop_pane() {
  if pane_alive; then
    tmux kill-pane -t "$(cat "${PANE_FILE}")"
    rm -f "${PANE_FILE}"
    replace_status "중지됨" "작업 기록 pane을 닫았습니다." "필요하면 auto로 다시 시작하세요."
    printf '작업 기록 pane 중지됨.\n'
    return 0
  fi
  rm -f "${PANE_FILE}"
  printf '실행 중인 작업 기록 pane을 찾지 못했습니다.\n'
}

cmd="${1:-}"
case "${cmd}" in
  start)
    shift
    start_pane "${1:-Codex 작업 기록}"
    ;;
  auto)
    shift
    auto_start_pane "${1:-Codex 작업 기록}"
    ;;
  stop)
    stop_pane
    ;;
  status)
    shift
    [[ $# -ge 1 ]] || { usage; exit 2; }
    replace_status "$1" "${2:-}" "${3:-}"
    ;;
  mode)
    shift
    [[ $# -ge 1 ]] || { usage; exit 2; }
    set_mode "$1" "${2:-}"
    ;;
  turn-start)
    shift
    [[ $# -ge 1 ]] || { usage; exit 2; }
    turn_start "$1"
    ;;
  turn-end)
    shift
    turn_end "${1:-작업 턴 완료.}"
    ;;
  entry|decision|evidence|blocker)
    shift
    [[ $# -ge 1 ]] || { usage; exit 2; }
    append_entry "${cmd}" "$1" "${2:-}"
    ;;
  path)
    ensure_worklog
    printf '%s\n' "${WORKLOG_FILE}"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    usage
    exit 2
    ;;
esac
