package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "agent hook starter kit을 출력하고 점검합니다",
		Long:  "Codex, Claude Code, Gemini용 opt-in hook 예시를 출력하거나 현재 환경에서 ingest 연결 가능성을 점검합니다.",
	}
	cmd.AddCommand(newHooksPrintCmd(), newHooksDoctorCmd())
	return cmd
}

func newHooksPrintCmd() *cobra.Command {
	var hookAgent string
	cmd := &cobra.Command{
		Use:   "print --agent <codex|claude|gemini>",
		Short: "복사 가능한 hook 설정 예시를 출력합니다",
		Long:  "설정 파일을 변경하지 않고 stdout에만 redacted ingest hook starter kit을 출력합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validHookAgent(hookAgent) {
				return fmt.Errorf("--agent must be codex, claude, or gemini, got %q", hookAgent)
			}
			bin, err := hookBinaryPath()
			if err != nil {
				return err
			}
			cmd.Print(renderHookStarter(hookAgent, bin))
			return nil
		},
	}
	cmd.Flags().StringVar(&hookAgent, "agent", "", "대상 agent: codex|claude|gemini")
	return cmd
}

func newHooksDoctorCmd() *cobra.Command {
	var hookAgent string
	var format string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "hook starter kit 실행 가능성을 점검합니다",
		Long:  "현재 바이너리, worklog 경로, redacted ingest smoke를 네트워크 없이 점검합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hookAgent != "" && !validHookAgent(hookAgent) {
				return fmt.Errorf("--agent must be codex, claude, gemini, or empty, got %q", hookAgent)
			}
			if format != "text" && format != "json" {
				return fmt.Errorf("--format must be text or json, got %q", format)
			}
			results := runHooksDoctor(hookAgent)
			if format == "json" {
				raw, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					return fmt.Errorf("hooks doctor json: %w", err)
				}
				cmd.Println(string(raw))
			} else {
				cmd.Print(renderHooksDoctor(results))
			}
			if !results.OK {
				return fmt.Errorf("hooks doctor failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&hookAgent, "agent", "", "대상 agent 필터: codex|claude|gemini")
	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	return cmd
}

type hooksDoctorResult struct {
	OK        bool               `json:"ok"`
	Binary    string             `json:"binary"`
	Worklog   string             `json:"worklog_dir"`
	Checks    []hooksDoctorCheck `json:"checks"`
	Agents    []string           `json:"agents"`
	NextSteps []string           `json:"next_steps"`
}

type hooksDoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func runHooksDoctor(agentFilter string) hooksDoctorResult {
	result := hooksDoctorResult{OK: true}
	bin, err := hookBinaryPath()
	if err != nil {
		result.OK = false
		result.Checks = append(result.Checks, hooksDoctorCheck{Name: "binary", OK: false, Detail: err.Error()})
	} else {
		result.Binary = bin
		result.Checks = append(result.Checks, hooksDoctorCheck{Name: "binary", OK: true, Detail: bin})
	}

	cfg := worklog.DefaultConfig()
	result.Worklog = cfg.Dir
	if err := cfg.EnsureDir(); err != nil {
		result.OK = false
		result.Checks = append(result.Checks, hooksDoctorCheck{Name: "worklog_dir", OK: false, Detail: err.Error()})
	} else {
		result.Checks = append(result.Checks, hooksDoctorCheck{Name: "worklog_dir", OK: true, Detail: cfg.Dir})
	}

	if bin != "" {
		smoke := exec.Command(bin, "ingest", "--source", "generic", "--event", "hook.doctor")
		smoke.Env = append(os.Environ(), "WORKLOG_DIR="+filepath.Join(os.TempDir(), "ntts-flightlog-hook-doctor"))
		smoke.Stdin = strings.NewReader(`{"event_name":"hook.doctor","summary":"hook doctor smoke","dedupe_key":"hook-doctor-smoke"}`)
		out, err := smoke.CombinedOutput()
		if err != nil {
			result.OK = false
			result.Checks = append(result.Checks, hooksDoctorCheck{Name: "ingest_smoke", OK: false, Detail: strings.TrimSpace(string(out)) + " " + err.Error()})
		} else {
			result.Checks = append(result.Checks, hooksDoctorCheck{Name: "ingest_smoke", OK: true, Detail: "redacted ingest command accepted JSON stdin"})
		}
	}

	for _, name := range []string{"codex", "claude", "gemini"} {
		if agentFilter != "" && name != agentFilter {
			continue
		}
		result.Agents = append(result.Agents, name)
		result.NextSteps = append(result.NextSteps, fmt.Sprintf("Run: ntts-flightlog hooks print --agent %s", name))
	}
	return result
}

func hookBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		return exe, nil
	}
	if path, err := exec.LookPath("ntts-flightlog"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("cannot resolve current executable and ntts-flightlog is not on PATH")
}

func validHookAgent(name string) bool {
	return name == "codex" || name == "claude" || name == "gemini"
}

func renderHooksDoctor(result hooksDoctorResult) string {
	var b strings.Builder
	b.WriteString("NTTS Flightlog hooks doctor\n")
	for _, check := range result.Checks {
		status := "PASS"
		if !check.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "- %s %s: %s\n", status, check.Name, check.Detail)
	}
	if len(result.NextSteps) > 0 {
		b.WriteString("\nNext steps:\n")
		for _, step := range result.NextSteps {
			fmt.Fprintf(&b, "- %s\n", step)
		}
	}
	return b.String()
}

func renderHookStarter(name, bin string) string {
	switch name {
	case "codex":
		return codexHookStarter(bin)
	case "claude":
		return claudeHookStarter(bin)
	default:
		return geminiHookStarter(bin)
	}
}

func hookIngestShell(bin, source string) string {
	return fmt.Sprintf(`printf '{"source":"%s","event_name":"session.hook","summary":"agent hook fired","dedupe_key":"%s-%%s"}' "$(date +%%s)" | %s ingest --source %s --event session.hook`, source, source, shellQuote(bin), source)
}

func codexHookStarter(bin string) string {
	return fmt.Sprintf(`# Codex hook starter kit for ntts-flightlog
# Copy the command into your Codex hook runner. This command does not mutate config.
# Captured fields: source, event_name, summary, dedupe_key.
# Dropped by ingest: raw stdout/stderr, raw prompts, full environment, secrets.

%s
`, hookIngestShell(bin, "codex"))
}

func claudeHookStarter(bin string) string {
	return fmt.Sprintf(`# Claude Code hook starter kit for ntts-flightlog
# Copy into your Claude Code hook command. This command does not mutate config.
# Captured fields: source, event_name, summary, dedupe_key.
# Dropped by ingest: raw stdout/stderr, raw prompts, full environment, secrets.

%s
`, hookIngestShell(bin, "claude"))
}

func geminiHookStarter(bin string) string {
	return fmt.Sprintf(`# Gemini CLI hook starter kit for ntts-flightlog
# Copy into your Gemini CLI hook command. This command does not mutate config.
# Captured fields: source, event_name, summary, dedupe_key.
# Dropped by ingest: raw stdout/stderr, raw prompts, full environment, secrets.

%s
`, hookIngestShell(bin, "gemini"))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
