package cli

import (
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [up|down]",
		Short: "데이터베이스 스키마 마이그레이션을 실행합니다",
		Long: `데이터베이스 스키마 마이그레이션을 실행합니다.
  up   — 최신 스키마로 업그레이드 (기본값)
  down — 이전 스키마로 롤백`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			direction := "up"
			if len(args) > 0 {
				direction = args[0]
			}

			cfg := worklog.DefaultConfig()
			if err := cfg.EnsureDir(); err != nil {
				return err
			}

			// db.Open already runs pending migrations on every open.
			// For "up", opening the DB is sufficient.
			s, err := openSession()
			if err != nil {
				return err
			}
			defer s.close()

			switch direction {
			case "up":
				cmd.Printf("migrate up: schema applied to %s\n", cfg.DBPath)
			case "down":
				// A5 implements v1→v2 migration parsing; schema rollback deferred to v2.1.
				cmd.Println("migrate down: schema rollback not yet implemented (Phase A5)")
			default:
				return cmd.Help()
			}
			return nil
		},
	}
	return cmd
}
