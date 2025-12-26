package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var cmdEditConfig = &cobra.Command{
	Use:   "edit-config",
	Short: "Edit configuration file in your default editor",
	Long: `Edit the veessh configuration file in your default editor.

The editor is determined by the EDITOR environment variable, or falls back
to 'vi' if not set. Common values:
  - vi / vim
  - nano
  - code (VS Code)
  - subl (Sublime Text)

Examples:
  EDITOR=nano veessh edit-config
  EDITOR=code veessh edit-config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}

		// Ensure config directory exists
		dir := filepath.Dir(cfgPath)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create empty config file if it doesn't exist
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			cfg := config.Config{Profiles: map[string]config.Profile{}}
			if err := config.Save(cfgPath, cfg); err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
		}

		// Determine editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		// Launch editor
		editorCmd := exec.Command(editor, cfgPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		fmt.Printf("Configuration saved to %s\n", cfgPath)
		return nil
	},
}

func init() {
	// cmdEditConfig is registered in root.go
}

