#!/usr/bin/env bash
set -euo pipefail

find_default_worklog_dir() {
  if [[ -d ".ntts-flightlog" ]]; then
    printf '%s\n' ".ntts-flightlog"; return
  fi
  if [[ -d ".omx/worklog" ]]; then
    printf '%s\n' ".omx/worklog"; return
  fi
  printf '%s\n' ".ntts-flightlog"
}

WORKLOG_DIR="${WORKLOG_DIR:-$(find_default_worklog_dir)}"
WORKLOG_FILE="${WORKLOG_FILE:-${WORKLOG_DIR}/main.md}"
TURNS_DIR="${TURNS_DIR:-${WORKLOG_DIR}/turns}"
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
  ntts-flightlog start [title]
  ntts-flightlog auto [title]
  ntts-flightlog stop
  ntts-flightlog status <label> [focus] [next]
  ntts-flightlog mode <solo|ralph|team|plan|review|autopilot|other> [detail]
  ntts-flightlog turn-start <title>
  ntts-flightlog turn-end [summary]
  ntts-flightlog entry <title> [detail]
  ntts-flightlog decision <title> [detail]
  ntts-flightlog evidence <title> [detail]
  ntts-flightlog blocker <title> [detail]
  ntts-flightlog path                # absolute path of main worklog file
  ntts-flightlog turn-path [N]       # absolute path of turn N (or current turn)
  ntts-flightlog view <flat|turns|decisions|blockers>  # one-shot ANSI render

In-pane view menu (when launched via `auto`/`start` inside tmux):
  [1] flat   [2] turns   [3] decisions   [4] blockers   [r] reload   [q] quit
  Turn-start titles are OSC 8 hyperlinks. cmd/ctrl-click in iTerm2 / WezTerm /
  Kitty / Ghostty / VS Code terminal opens the per-turn markdown file.

Environment:
  WORKLOG_DIR       default: .ntts-flightlog (or .omx/worklog if it already exists)
  WORKLOG_FILE      default: ${WORKLOG_DIR}/main.md
  TURNS_DIR         default: ${WORKLOG_DIR}/turns
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

abs_path() {
  local p="$1"
  case "${p}" in
    /*) printf '%s\n' "${p}" ;;
    *)  printf '%s/%s\n' "$(pwd)" "${p}" ;;
  esac
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

ensure_worklog() {
  mkdir -p "${WORKLOG_DIR}"
  if [[ ! -f "${WORKLOG_FILE}" ]]; then
    local title="${1:-작업 기록}"
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

ensure_turns_dir() {
  mkdir -p "${TURNS_DIR}"
}

current_turn_number_or_zero() {
  if [[ -f "${TURN_COUNTER_FILE}" ]]; then
    cat "${TURN_COUNTER_FILE}"
  else
    printf '0\n'
  fi
}

turn_file_for() {
  local n="$1"
  printf '%s/turn-%s.md\n' "${TURNS_DIR}" "${n}"
}

append_to_current_turn() {
  local kind="$1"
  local title="$2"
  local detail="${3:-}"
  [[ -f "${TURN_START_FILE}" ]] || return 0
  local n
  n="$(current_turn_number_or_zero)"
  [[ "${n}" != "0" ]] || return 0
  ensure_turns_dir
  local turn_file
  turn_file="$(turn_file_for "${n}")"
  {
    printf '\n### %s [%s] %s\n' "$(timestamp)" "${kind}" "${title}"
    if [[ -n "${detail}" ]]; then
      printf '%s\n' "${detail}"
    fi
  } >>"${turn_file}"
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
      printf '%s\n' "${detail}"
    fi
  } >>"${WORKLOG_FILE}"
  append_to_current_turn "${kind}" "${title}" "${detail}"
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
  printf '%s\n' "$(epoch_seconds)" >"${TURN_START_FILE}"
  ensure_turns_dir
  local turn_file
  turn_file="$(turn_file_for "${turn_number}")"
  cat >"${turn_file}" <<EOF
# Turn ${turn_number}: ${title}

시작: $(timestamp)

EOF
  append_entry "turn-${turn_number}-start" "${title}" "시작 시각: $(timestamp)."
  replace_status "turn-${turn_number}" "${title}" "결정과 검증 근거를 진행 중에 기록하세요."
}

turn_end() {
  local summary="${1:-작업 턴 완료.}"
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

render_markdown_ansi() {
  local file="$1"
  [[ -f "${file}" ]] || { printf '(파일이 없습니다: %s)\n' "${file}"; return; }
  local abs_turns
  abs_turns="$(abs_path "${TURNS_DIR}")"
  awk -v turns_dir="${abs_turns}" '
    BEGIN {
      ESC = sprintf("%c", 27)
      RESET = ESC "[0m"
      BOLD = ESC "[1m"
      DIM = ESC "[2m"
      ST = ESC "\\"
      OSC8 = ESC "]8;;"
      colors[0] = ESC "[38;5;81m"
      colors[1] = ESC "[38;5;114m"
      colors[2] = ESC "[38;5;215m"
      colors[3] = ESC "[38;5;177m"
      colors[4] = ESC "[38;5;117m"
      red = ESC "[38;5;203m"
      entry = 0
    }
    function osc_link(url, text) {
      return OSC8 url ST text OSC8 ST
    }
    /^# / { print BOLD colors[0] $0 RESET; next }
    /^## / { print ""; print BOLD colors[4] $0 RESET; next }
    /^업데이트:/ { print DIM $0 RESET; next }
    /^시작:/ { print DIM $0 RESET; next }
    /^### .*\[/ {
      line = $0
      sub(/^### /, "", line)
      ts = line
      sub(/ .*/, "", ts)
      kind = line
      sub(/^[^[]*/, "", kind)
      sub(/].*/, "]", kind)
      title = line
      sub(/^[^]]*] /, "", title)
      if (line ~ /\[turn-[0-9]+-start\]/) {
        n = line
        sub(/.*\[turn-/, "", n)
        sub(/-start\].*/, "", n)
        url = "file://" turns_dir "/turn-" n ".md"
        color = colors[entry % 5]
        print color "■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■" RESET
        printf "%s%s▶ %s  %s%s\n", BOLD, color, ts, kind, RESET
        printf "%s%s  %s%s\n", BOLD, color, osc_link(url, title), RESET
        print color "────────────────────────────────" RESET
        entry++
        next
      }
      if (line ~ /\[mode\]/) {
        printf "%s%s▣ %s  %s%s\n", BOLD, colors[3], ts, kind, RESET
        printf "%s%s  %s%s\n", BOLD, colors[3], title, RESET
        entry++
        next
      }
      if (line ~ /\[entry\]/) {
        color = colors[entry % 5]
        printf "%s%s◆ %s  %s%s\n", BOLD, color, ts, kind, RESET
        printf "%s%s  %s%s\n", BOLD, color, title, RESET
        entry++
        next
      }
      if (line ~ /\[(turn-[0-9]+-end|evidence)\]/) {
        printf "%s%s■ %s  %s%s\n", BOLD, colors[1], ts, kind, RESET
        printf "%s%s  %s%s\n", BOLD, colors[1], title, RESET
        entry++
        next
      }
      if (line ~ /\[decision\]/) {
        printf "%s%s◆ %s  %s%s\n", BOLD, colors[2], ts, kind, RESET
        printf "%s%s  %s%s\n", BOLD, colors[2], title, RESET
        entry++
        next
      }
      if (line ~ /\[blocker\]/) {
        printf "%s%s!! %s  %s%s\n", BOLD, red, ts, kind, RESET
        printf "%s%s  %s%s\n", BOLD, red, title, RESET
        entry++
        next
      }
    }
    /^[^#[:space:]][^#]*$/ { print DIM $0 RESET; next }
    { print }
  ' "${file}"
}

filter_entries_by_kind() {
  local kind="$1"
  [[ -f "${WORKLOG_FILE}" ]] || return 0
  awk -v kind="${kind}" '
    BEGIN {
      ESC = sprintf("%c", 27)
      RESET = ESC "[0m"
      BOLD = ESC "[1m"
      DIM = ESC "[2m"
      amber = ESC "[38;5;215m"
      red = ESC "[38;5;203m"
      printing = 0
    }
    /^### .*\[/ {
      pat = "\\[" kind "\\]"
      if ($0 ~ pat) {
        line = $0
        sub(/^### /, "", line)
        ts = line; sub(/ .*/, "", ts)
        title = line; sub(/^[^]]*] /, "", title)
        if (kind == "blocker") {
          printf "%s%s!! %s%s\n", BOLD, red, ts, RESET
          printf "%s%s  %s%s\n", BOLD, red, title, RESET
        } else {
          printf "%s%s◆ %s%s\n", BOLD, amber, ts, RESET
          printf "%s%s  %s%s\n", BOLD, amber, title, RESET
        }
        printing = 1
        next
      } else {
        printing = 0
        next
      }
    }
    {
      if (printing && $0 !~ /^[[:space:]]*$/ && $0 !~ /^### /) {
        print DIM $0 RESET
      }
    }
  ' "${WORKLOG_FILE}"
}

render_view() {
  local view="$1"
  case "${view}" in
    flat)
      render_markdown_ansi "${WORKLOG_FILE}"
      ;;
    turns)
      if [[ ! -d "${TURNS_DIR}" ]]; then
        printf '(turn 파일이 아직 없습니다. turn-start로 첫 턴을 시작하세요.)\n'
        return 0
      fi
      local any=0 f
      while IFS= read -r f; do
        any=1
        render_markdown_ansi "${f}"
        printf '\n'
      done < <(ls -1 "${TURNS_DIR}"/turn-*.md 2>/dev/null | sort -V)
      if [[ "${any}" -eq 0 ]]; then
        printf '(turn 파일이 아직 없습니다. turn-start로 첫 턴을 시작하세요.)\n'
      fi
      ;;
    decisions)
      filter_entries_by_kind "decision"
      ;;
    blockers)
      filter_entries_by_kind "blocker"
      ;;
    *)
      printf '알 수 없는 view: %s (flat|turns|decisions|blockers)\n' "${view}" >&2
      return 2
      ;;
  esac
}

pane_alive() {
  [[ -f "${PANE_FILE}" ]] || return 1
  local pane_id
  pane_id="$(cat "${PANE_FILE}")"
  tmux has-session 2>/dev/null || return 1
  tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qx "${pane_id}"
}

viewer_command() {
  local abs_script abs_file abs_worklog_dir abs_turns_dir
  abs_script="$(abs_path "$0")"
  abs_file="$(abs_path "${WORKLOG_FILE}")"
  abs_worklog_dir="$(abs_path "${WORKLOG_DIR}")"
  abs_turns_dir="$(abs_path "${TURNS_DIR}")"
  cat <<EOF
view="flat"

print_header() {
  local v="\$1"
  local reset bold dim hl pane_color
  reset=\$(printf '\\033[0m')
  bold=\$(printf '\\033[1m')
  dim=\$(printf '\\033[2m')
  hl=\$(printf '\\033[38;5;226m')
  pane_color=\$(printf '\\033[38;5;81m')
  printf '%s%s작업 기록 PANE%s  %s\\n' "\$bold" "\$pane_color" "\$reset" '${abs_file}'
  local items=("flat:[1] 평면" "turns:[2] 턴별" "decisions:[3] 결정" "blockers:[4] 블로커")
  local label_line="" item id label
  for item in "\${items[@]}"; do
    id="\${item%%:*}"
    label="\${item#*:}"
    if [[ "\$id" == "\$v" ]]; then
      label_line+="\${bold}\${hl}\${label}\${reset}  "
    else
      label_line+="\${dim}\${label}\${reset}  "
    fi
  done
  label_line+="\${dim}[r] 새로고침  [q] 종료\${reset}"
  printf '%b\\n' "\$label_line"
  printf '%*s\\n' 80 '' | tr ' ' -
}

draw_once() {
  printf '\\033[H\\033[J'
  print_header "\$view"
  WORKLOG_DIR='${abs_worklog_dir}' \\
  WORKLOG_FILE='${abs_file}' \\
  TURNS_DIR='${abs_turns_dir}' \\
  '${abs_script}' view "\$view" 2>/dev/null | sed -n '1,240p'
}

draw_once
while true; do
  if read -t '${REFRESH_SECONDS}' -n 1 key 2>/dev/null; then
    case "\$key" in
      1) view="flat";      draw_once ;;
      2) view="turns";     draw_once ;;
      3) view="decisions"; draw_once ;;
      4) view="blockers";  draw_once ;;
      r|R) draw_once ;;
      q|Q) break ;;
      *) : ;;
    esac
  else
    draw_once
  fi
done
EOF
}

start_pane() {
  local title="${1:-작업 기록}"
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
  local title="${1:-작업 기록}"
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
    start_pane "${1:-작업 기록}"
    ;;
  auto)
    shift
    auto_start_pane "${1:-작업 기록}"
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
    abs_path "${WORKLOG_FILE}"
    ;;
  turn-path)
    shift
    turn_arg=""
    if [[ $# -ge 1 ]]; then
      turn_arg="$1"
    else
      turn_arg="$(current_turn_number_or_zero)"
    fi
    if [[ "${turn_arg}" == "0" ]]; then
      printf '활성 turn이 없습니다. turn-start로 시작하세요.\n' >&2
      exit 2
    fi
    abs_path "$(turn_file_for "${turn_arg}")"
    ;;
  view)
    shift
    [[ $# -ge 1 ]] || { usage; exit 2; }
    render_view "$1"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    usage
    exit 2
    ;;
esac
