package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [제목]",
		Short: "새 세션을 시작합니다",
		Long:  "새 작업 세션을 시작하고 SQLite에 기록하며 main.md를 초기화합니다.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := "작업 기록"
			if len(args) > 0 {
				title = args[0]
			}
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			if err := worklog.EnsureMainMd(s.cfg, title); err != nil {
				return err
			}

			// Insert new session into SQLite.
			sessionID, err := insertSession(s, title, "solo")
			if err != nil {
				return err
			}
			if err := worklog.WriteFile(s.cfg.SessionIDFile, sessionID); err != nil {
				return err
			}
			if err := worklog.WriteFile(s.cfg.SessionStart, worklog.EpochSeconds()); err != nil {
				return err
			}

			if err := worklog.ReplaceStatus(s.cfg, "활성",
				"작업 기록 pane이 표시됩니다.",
				"turn-start로 시간 측정 작업 턴을 시작하세요."); err != nil {
				return err
			}

			return startPane(s.cfg, title, cmd)
		},
	}
	return cmd
}

// insertSession inserts a new session row and returns its ID.
func insertSession(s *session, title, mode string) (string, error) {
	const q = `INSERT INTO sessions (started_at, mode, agent_id, agent_detected, agent_override, title)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id`
	var id string
	err := s.store.QueryRow(q,
		now(), mode,
		nullStr(s.agentID), nullStr(s.detected), nullStr(s.override),
		nullStr(title),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return id, nil
}

// ensureActiveSession returns a usable session ID, creating and persisting one
// when the v2 session-id file is missing. This keeps append-style commands from
// writing NULL session_id rows when users call them before start/auto completes.
func ensureActiveSession(s *session, title, mode string) (string, error) {
	if sessionID := s.cfg.ActiveSessionID(); sessionID != "" {
		return sessionID, nil
	}
	if title == "" {
		title = "작업 기록"
	}
	if mode == "" {
		mode = "solo"
	}
	sessionID, err := insertSession(s, title, mode)
	if err != nil {
		return "", err
	}
	if err := worklog.WriteFile(s.cfg.SessionIDFile, sessionID); err != nil {
		return "", err
	}
	if worklog.ReadFile(s.cfg.SessionStart) == "" {
		if err := worklog.WriteFile(s.cfg.SessionStart, worklog.EpochSeconds()); err != nil {
			return "", err
		}
	}
	return sessionID, nil
}

// startPane spawns a tmux side pane running the interactive viewer loop.
// If not in tmux or tmux is unavailable, prints an informational message.
func startPane(cfg *worklog.Config, title string, cmd *cobra.Command) error {
	if os.Getenv("TMUX") == "" {
		cmd.Printf("작업 기록 파일: %s\n", cfg.MainMd)
		cmd.Println("현재 shell에서는 tmux side pane 렌더링을 사용할 수 없습니다.")
		return nil
	}
	if paneAlive(cfg) {
		cmd.Printf("작업 기록 pane이 이미 실행 중입니다: %s\n", worklog.ReadFile(cfg.PaneFile))
		return nil
	}

	absFile, _ := filepath.Abs(cfg.MainMd)
	absWorklogDir, _ := filepath.Abs(cfg.Dir)
	absTurnsDir, _ := filepath.Abs(cfg.TurnsDir)

	// Find our own binary path.
	self, err := os.Executable()
	if err != nil {
		self = "flightlog"
	}

	script := viewerScript(self, absFile, absWorklogDir, absTurnsDir)
	pct := os.Getenv("PANE_PERCENT")
	if pct == "" {
		pct = "34"
	}

	out, err := exec.Command("tmux", "split-window", "-h",
		"-p", pct, "-P", "-F", "#{pane_id}",
		"bash", "-c", script).Output()
	if err != nil {
		return fmt.Errorf("tmux split-window: %w", err)
	}
	paneID := strings.TrimSpace(string(out))
	if err := worklog.WriteFile(cfg.PaneFile, paneID); err != nil {
		return err
	}
	cmd.Printf("작업 기록 pane 시작됨: %s\n", paneID)
	cmd.Printf("작업 기록 파일: %s\n", absFile)
	return nil
}

// paneAlive checks whether the stored pane ID is still alive in tmux.
func paneAlive(cfg *worklog.Config) bool {
	paneID := worklog.ReadFile(cfg.PaneFile)
	if paneID == "" {
		return false
	}
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == paneID {
			return true
		}
	}
	return false
}

// viewerScript returns the inline bash loop script for the tmux pane.
// Mirrors v1's viewer_command() output exactly, adapted to call the v2 binary.
func viewerScript(self, absFile, absWorklogDir, absTurnsDir string) string {
	refresh := os.Getenv("REFRESH_SECONDS")
	if refresh == "" {
		refresh = "2"
	}
	return fmt.Sprintf(`
view="flat"

print_header() {
  local v="$1"
  local reset bold dim hl
  reset=$(printf '\033[0m')
  bold=$(printf '\033[1m')
  dim=$(printf '\033[2m')
  hl=$(printf '\033[38;5;226m')
  local items=("flat:[1]평면" "turns:[2]턴별" "decisions:[3]결정" "blockers:[4]블로커" "report:[5]리포트" "visual:[6]시각화")
  local label_line="" item id label
  for item in "${items[@]}"; do
    id="${item%%:*}"
    label="${item#*:}"
    if [[ "$id" == "$v" ]]; then
      label_line+="${bold}${hl}${label}${reset} "
    else
      label_line+="${dim}${label}${reset} "
    fi
  done
  printf '%%b\n' "$label_line"
  printf '%%b\n' "${dim}[r]새로고침 [q]종료${reset}"
  printf '%%b\n' "${dim}스크롤 후 키가 안 먹으면 Esc/q로 복귀${reset}"
  local cols
  cols=$(tput cols 2>/dev/null || printf 80)
  printf '%%*s\n' $cols '' | tr ' ' -
}

draw_once() {
  printf '\033[?7h\033[H\033[J\033[3J'
  print_header "$view"
  local term_h content_h
  term_h=$(tput lines 2>/dev/null || printf 30)
  content_h=$((term_h - 5))
  [ $content_h -lt 5 ] && content_h=5
  case "$view" in
    flat)
      WORKLOG_DIR='%s' \
      WORKLOG_FILE='%s' \
      TURNS_DIR='%s' \
      '%s' view "$view" 2>/dev/null | tail -n "$content_h"
      ;;
    *)
      WORKLOG_DIR='%s' \
      WORKLOG_FILE='%s' \
      TURNS_DIR='%s' \
      '%s' view "$view" 2>/dev/null | sed -n "1,${content_h}p"
      ;;
  esac
}

get_mtime() {
  stat -f "%%m" '%s' 2>/dev/null || stat -c "%%Y" '%s' 2>/dev/null || printf 0
}

draw_once
last_mtime=$(get_mtime)
while true; do
  if read -t '%s' -n 1 key 2>/dev/null; then
    case "$key" in
      1) view="flat";      draw_once; last_mtime=$(get_mtime) ;;
      2) view="turns";     draw_once; last_mtime=$(get_mtime) ;;
      3) view="decisions"; draw_once; last_mtime=$(get_mtime) ;;
      4) view="blockers";  draw_once; last_mtime=$(get_mtime) ;;
      5) view="report";    draw_once; last_mtime=$(get_mtime) ;;
      6) view="visual";    draw_once; last_mtime=$(get_mtime) ;;
      r|R|ㄱ) draw_once; last_mtime=$(get_mtime) ;;
      q|Q|ㅂ) break ;;
      *) : ;;
    esac
  else
    current_mtime=$(get_mtime)
    if [[ "$current_mtime" != "$last_mtime" ]]; then
      draw_once
      last_mtime=$current_mtime
    fi
  fi
done
`, absWorklogDir, absFile, absTurnsDir, self, absWorklogDir, absFile, absTurnsDir, self, absFile, absFile, refresh)
}
