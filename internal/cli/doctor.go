package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "로컬 flightlog 설치와 worklog 상태를 점검합니다",
		Long:  "네트워크 없이 현재 바이너리, DB 마이그레이션, tmux pane, skill wrapper 위임 상태를 점검합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			exe, exeErr := os.Executable()
			if exeErr != nil {
				exe = "(확인 실패: " + exeErr.Error() + ")"
			}
			pathBin, pathErr := exec.LookPath("ntts-flightlog")
			if pathErr != nil {
				pathBin = "(PATH에서 ntts-flightlog를 찾지 못함)"
			}

			sqliteVersion, err := s.store.Version()
			if err != nil {
				return fmt.Errorf("doctor: sqlite version: %w", err)
			}
			migrations, err := appliedMigrationCount(s)
			if err != nil {
				return fmt.Errorf("doctor: migration count: %w", err)
			}

			cmd.Println("NTTS Flightlog doctor")
			cmd.Printf("binary: %s\n", exe)
			cmd.Printf("path: %s\n", pathBin)
			cmd.Printf("version: %s\n", versionString())
			cmd.Printf("worklog_dir: %s\n", s.cfg.Dir)
			cmd.Printf("db: ok (%s)\n", s.cfg.DBPath)
			cmd.Printf("sqlite: %s\n", sqliteVersion)
			cmd.Printf("migrations: %d applied\n", migrations)
			cmd.Printf("tmux_pane: %s\n", paneDoctorStatus(s.cfg))

			cmd.Println("skill_wrappers:")
			for _, status := range skillWrapperStatuses() {
				cmd.Printf("  %s: %s\n", status.agent, status.summary)
			}
			return nil
		},
	}
	return cmd
}

func appliedMigrationCount(s *session) (int, error) {
	var count int
	err := s.store.QueryRow("SELECT COUNT(*) FROM " + db.TableSchemaMigrations).Scan(&count)
	return count, err
}

func paneDoctorStatus(cfg *worklog.Config) string {
	paneID := worklog.ReadFile(cfg.PaneFile)
	if paneID == "" {
		return "not running"
	}
	if paneAlive(cfg) {
		return "alive (" + paneID + ")"
	}
	return "recorded but not running (" + paneID + ")"
}

type skillWrapperStatus struct {
	agent   string
	summary string
}

func skillWrapperStatuses() []skillWrapperStatus {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return []skillWrapperStatus{{agent: "home", summary: "home directory 확인 실패"}}
	}

	paths := []struct {
		agent string
		path  string
	}{
		{"codex", filepath.Join(home, ".codex", "skills", "ntts-flightlog", "scripts", "flightlog.sh")},
		{"claude", filepath.Join(home, ".claude", "skills", "ntts-flightlog", "scripts", "flightlog.sh")},
		{"gemini", filepath.Join(home, ".gemini", "skills", "ntts-flightlog", "scripts", "flightlog.sh")},
		{"agents", filepath.Join(home, ".agents", "skills", "ntts-flightlog", "scripts", "flightlog.sh")},
	}

	statuses := make([]skillWrapperStatus, 0, len(paths))
	for _, item := range paths {
		statuses = append(statuses, skillWrapperStatus{
			agent:   item.agent,
			summary: describeSkillWrapper(item.path),
		})
	}
	return statuses
}

func describeSkillWrapper(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "not installed"
		}
		return "unreadable: " + err.Error()
	}
	text := string(data)
	if strings.Contains(text, "NTTS_FLIGHTLOG_BIN") &&
		strings.Contains(text, "command -v ntts-flightlog") &&
		strings.Contains(text, "exec") {
		return "delegates to Go CLI"
	}
	return "installed but delegation pattern not recognized"
}
