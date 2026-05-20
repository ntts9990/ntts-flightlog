// Package cli defines the Cobra CLI root command and all subcommands for
// ntts-flightlog v2. Subcommand implementations live in adjacent files.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion string
	agentFlag  string // --agent <name> global override
)

// rootCmd is the top-level command: `flightlog`.
var rootCmd = &cobra.Command{
	Use:   "flightlog",
	Short: "NTTS Flightlog — turn-by-turn session worklog CLI",
	Long: `ntts-flightlog v2: a static Go binary that records session turns,
entries, decisions, evidence, and blockers into SQLite and mirrors
progress to .ntts-flightlog/main.md for v1 compatibility.`,
	// No Run — subcommand required; prints help when called bare.
	SilenceUsage: true,
}

// Execute runs the root command. Called from main.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion stores the build-time version string for --version output.
func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

func init() {
	// Global flag: agent override (propagates to all subcommands).
	rootCmd.PersistentFlags().StringVar(&agentFlag, "agent", "", "override auto-detected agent name (e.g. claude, codex, gemini)")

	// Register all subcommands.
	rootCmd.AddCommand(
		newStartCmd(),
		newStopCmd(),
		newAutoCmd(),
		newStatusCmd(),
		newModeCmd(),
		newTurnStartCmd(),
		newTurnEndCmd(),
		newEntryCmd(),
		newDecisionCmd(),
		newEvidenceCmd(),
		newBlockerCmd(),
		newPathCmd(),
		newTurnPathCmd(),
		newViewCmd(),
		newMigrateCmd(),
		newReportCmd(),
		newAgentStatsCmd(),
		newRefreshAnchorCmd(),
		newDriftCheckCmd(),
		newSelfUpgradeCmd(),
	)
}

// versionString returns the formatted version line.
func versionString() string {
	return fmt.Sprintf("flightlog %s", appVersion)
}
