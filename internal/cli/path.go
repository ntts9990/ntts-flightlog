package cli

import (
	"fmt"
	"path/filepath"

	"github.com/ntts9990/ntts-flightlog/internal/worklog"
	"github.com/spf13/cobra"
)

func newPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "워크로그 디렉토리 절대 경로를 출력합니다",
		Long:  ".ntts-flightlog/main.md 파일의 절대 경로를 출력합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := worklog.DefaultConfig()
			if err := cfg.EnsureDir(); err != nil {
				return err
			}
			abs, err := filepath.Abs(cfg.MainMd)
			if err != nil {
				return err
			}
			fmt.Println(abs)
			return nil
		},
	}
	return cmd
}
