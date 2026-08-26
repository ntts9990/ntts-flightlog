package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/tui"
	tuiviews "github.com/ntts9990/ntts-flightlog/internal/tui/views"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

func newViewCmd() *cobra.Command {
	var nonInteractive bool
	var tuiView string

	cmd := &cobra.Command{
		Use:   "view [flat|turns|decisions|blockers|report|visual|tui]",
		Short: "워크로그를 ANSI 색상으로 출력합니다",
		Long: `워크로그를 지정한 뷰 모드로 ANSI 렌더링하여 출력합니다.

Modes:
  flat        전체 워크로그 (기본값, main.md 기반)
  turns       턴 인덱스 (SQLite 기반 요약)
  decisions   결정 기록 (SQLite 기반)
  blockers    블로커/리스크 (SQLite 기반)
  report      운영 리포트 (SQLite 기반)
  visual      리포트 ASCII 시각화 (SQLite 기반)
  tui         Bubble Tea 인터랙티브 TUI (SQLite 기반, Phase B)

TUI flags:
  --noninteractive   TUI를 시작하지 않고 지정 뷰를 stdout으로 출력 (기본: flat)
  --view <name>      noninteractive 모드에서 출력할 뷰 (flat|turns|decisions|blockers|report|visual)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := "flat"
			if len(args) > 0 {
				mode = args[0]
			}
			cfg := worklog.DefaultConfig()
			w := os.Stdout

			switch mode {
			case "flat":
				return worklog.RenderFlat(cfg, w)
			case "turns":
				return renderSQLiteView(cfg, "turns", w)
			case "decisions":
				return renderSQLiteView(cfg, "decisions", w)
			case "blockers":
				return renderSQLiteView(cfg, "blockers", w)
			case "report":
				return renderSQLiteView(cfg, "report", w)
			case "visual":
				return renderSQLiteView(cfg, "visual", w)

			case "tui":
				d, err := db.Open(cfg.DBPath)
				if err != nil {
					return fmt.Errorf("view tui: open db: %w", err)
				}
				defer func() { _ = d.Close() }()

				if nonInteractive {
					// Render a single view to stdout and exit — used for
					// byte-equality tests (B2) and CI capture.
					data, err := tuiviews.LoadAll(d)
					if err != nil {
						return fmt.Errorf("view tui --noninteractive: load: %w", err)
					}
					content := tui.RenderView(data, tuiView, cfg.TurnsDir)
					_, _ = fmt.Fprint(w, content)
					return nil
				}

				// Launch the interactive Bubble Tea TUI.
				return tui.Run(d, cfg.TurnsDir)

			default:
				return fmt.Errorf("알 수 없는 view: %s (flat|turns|decisions|blockers|report|visual|tui)", mode)
			}
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "noninteractive", false,
		"TUI 없이 지정 뷰를 stdout으로 출력하고 종료합니다 (tui 모드 전용)")
	cmd.Flags().StringVar(&tuiView, "view", "flat",
		"noninteractive 모드에서 출력할 뷰 (flat|turns|decisions|blockers|report|visual)")

	return cmd
}

func renderSQLiteView(cfg *worklog.Config, viewName string, w *os.File) error {
	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("view %s: open db: %w", viewName, err)
	}
	defer func() { _ = d.Close() }()

	data, err := tuiviews.LoadAll(d)
	if err != nil {
		return fmt.Errorf("view %s: load: %w", viewName, err)
	}
	_, _ = fmt.Fprint(w, tui.RenderView(data, viewName, cfg.TurnsDir))
	return nil
}
