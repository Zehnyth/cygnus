package cli

import (
	"encoding/json"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/zehnyth/cygnus/services"
	"github.com/zehnyth/cygnus/types"
	"github.com/zehnyth/cygnus/ui"
)

// reportCmd is cygnus's only subcommand: one-shot memory triage. It loads
// config, runs the collection + correlation pipeline via services.Generate,
// then dispatches to one of three output modes (text, --tui, --json).
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "One-shot memory triage: correlated system + per-process view with findings",
	RunE: func(cmd *cobra.Command, args []string) error {
		// The TUI lists every process on the machine and scrolls, so unless
		// the user explicitly set --top, request all processes (topN=0).
		effectiveTop := topN
		if tuiMode && !cmd.Flags().Changed("top") {
			effectiveTop = 0
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		rep, err := services.Generate(cfg, match, effectiveTop)
		if err != nil {
			return err
		}

		switch {
		case jsonOut:
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rep)
		case tuiMode:
			fetch := func() (*types.Report, error) { return services.Generate(cfg, match, effectiveTop) }
			p := tea.NewProgram(ui.New(rep, fetch), tea.WithAltScreen())
			_, err := p.Run()
			return err
		default:
			renderText(os.Stdout, rep)
			return nil
		}
	},
}
