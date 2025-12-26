package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

var cmdSetBackend = &cobra.Command{
	Use:   "set-backend [backend]",
	Short: "Set the default credential backend in config",
	Long: `Set the default credential backend in the config file.

Backends:
  - auto: Auto-detect (prefer 1Password, then keyring, then file)
  - 1password: Use 1Password CLI
  - keyring: Use system keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager)
  - file: Use encrypted file backend (works on all platforms)

The environment variable VEESSH_CREDENTIALS_BACKEND takes precedence over this setting.

Examples:
  # Use 1Password as default
  veessh set-backend 1password

  # Use file backend as default
  veessh set-backend file

  # Use auto-detection (default)
  veessh set-backend auto`,
	Args: cobra.ExactArgs(1),
	RunE: runSetBackend,
}

func runSetBackend(cmd *cobra.Command, args []string) error {
	backendStr := args[0]
	backendType := credentials.BackendType(backendStr)

	// Validate backend type
	validBackends := map[credentials.BackendType]bool{
		credentials.BackendAuto:      true,
		credentials.Backend1Password:  true,
		credentials.BackendKeyring:    true,
		credentials.BackendFile:      true,
	}

	if !validBackends[backendType] {
		return fmt.Errorf("invalid backend: %s (must be: auto, 1password, keyring, or file)", backendStr)
	}

	// Load config
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set default backend
	cfg.DefaultBackend = backendStr

	// Save config
	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Default backend set to: %s\n", backendStr)
	fmt.Printf("Config saved to: %s\n", cfgPath)
	fmt.Printf("\nNote: VEESSH_CREDENTIALS_BACKEND environment variable takes precedence over this setting.\n")

	return nil
}

