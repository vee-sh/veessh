package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

var cmdBackends = &cobra.Command{
	Use:   "backends",
	Short: "List available credential storage backends",
	Long: `List all available credential storage backends and their status.
Shows which backends are available on your system and which one is currently active.

The active backend is determined by:
  1. VEESSH_CREDENTIALS_BACKEND environment variable (highest priority)
  2. defaultBackend in config file
  3. Auto-detection (1Password → keyring → file)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check current backend
		currentBackend, err := credentials.GetBackend()
		currentBackendName := "unknown"
		if err == nil && currentBackend != nil {
			switch currentBackend.(type) {
			case *credentials.OnePasswordBackend:
				currentBackendName = "1password"
			case *credentials.KeyringBackend:
				currentBackendName = "keyring"
			case *credentials.FileBackend:
				currentBackendName = "file"
			}
		}

		// Check config file for default
		cfgPath, _ := config.DefaultPath()
		cfg, _ := config.Load(cfgPath)
		configDefault := cfg.DefaultBackend
		if configDefault == "" {
			configDefault = "auto"
		}

		// Check environment variable
		envBackend := os.Getenv("VEESSH_CREDENTIALS_BACKEND")

		fmt.Println("Available credential backends:")
		fmt.Println()

		// 1Password
		fmt.Print("  1password  - 1Password CLI integration")
		op := credentials.NewOnePasswordBackend("")
		if op.IsAvailable() {
			fmt.Print(" [✓ Available]")
			if currentBackendName == "1password" {
				fmt.Print(" [ACTIVE]")
			}
		} else {
			fmt.Print(" [✗ Not available]")
			// Check if op CLI is installed
			if _, err := exec.LookPath("op"); err != nil {
				fmt.Print(" (install: brew install --cask 1password-cli)")
			} else {
				fmt.Print(" (not signed in, run: op signin)")
			}
		}
		fmt.Println()

		// System Keyring
		fmt.Print("  keyring    - System keyring")
		kr := credentials.NewKeyringBackend()
		// Try to test keyring availability
		testProfile := "__veessh_test_backend__"
		if err := kr.SetPassword(testProfile, "test"); err == nil {
			kr.DeletePassword(testProfile)
			fmt.Print(" [✓ Available]")
			if currentBackendName == "keyring" {
				fmt.Print(" [ACTIVE]")
			}
		} else {
			fmt.Print(" [✗ Not available]")
			// Platform-specific hints
			if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("DISPLAY") == "" {
				fmt.Print(" (no display/GUI session)")
			}
		}
		fmt.Println()

		// File backend
		fmt.Print("  file       - Encrypted file (~/.config/veessh/passwords.enc)")
		fmt.Print(" [✓ Always available]")
		if currentBackendName == "file" {
			fmt.Print(" [ACTIVE]")
		}
		fmt.Println()

		fmt.Println()
		fmt.Println("Current configuration:")
		fmt.Printf("  Active backend:       %s\n", currentBackendName)
		if envBackend != "" {
			fmt.Printf("  Environment override: %s (VEESSH_CREDENTIALS_BACKEND)\n", envBackend)
		}
		fmt.Printf("  Config default:       %s\n", configDefault)
		
		fmt.Println()
		fmt.Println("To set a default backend:")
		fmt.Println("  veessh set-backend <backend>")
		fmt.Println()
		fmt.Println("To temporarily override:")
		fmt.Println("  VEESSH_CREDENTIALS_BACKEND=file veessh connect myserver")

		return nil
	},
}

func init() {
	// cmdBackends is registered in root.go
}
