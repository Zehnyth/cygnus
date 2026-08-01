// Package cli is the Cobra layer: command and flag definitions only.
// No /proc access and no correlation logic here — everything routes
// through services.
package cli

import "github.com/spf13/cobra"

var (
	jsonOut    bool
	tuiMode    bool
	match      string
	topN       int
	configPath string
)

var rootCmd = &cobra.Command{
	Use:           "cygnus",
	Short:         "Host memory triage: correlated process investigation with a verdict",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root Cobra command. Called once from main.go.
func Execute() error { return rootCmd.Execute() }

// init registers global flags and subcommands on rootCmd.
func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to a JSON config file overriding rule thresholds (default: search XDG/user/system paths)")

	reportCmd.Flags().StringVar(&match, "match", "", "also count processes whose cmdline contains this substring")
	reportCmd.Flags().IntVar(&topN, "top", 20, "number of top consumers to list")
	reportCmd.Flags().BoolVar(&tuiMode, "tui", false, "launch the interactive view instead of static output")

	rootCmd.AddCommand(reportCmd)
}
