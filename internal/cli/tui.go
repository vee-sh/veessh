package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/tui"
)

var cmdTUI = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI for managing profiles",
	Long: `Launch the Terminal User Interface for managing SSH profiles.
	
The TUI provides an interactive way to:
  - Browse and search profiles
  - Edit and create new profiles
  - Organize profiles into groups
  - Manage favorites and tags
  - Test connections
  - View connection history`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}

		// Create the TUI model
		model, err := tui.New(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to initialize TUI: %w", err)
		}

		// Start the TUI
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		return nil
	},
}
