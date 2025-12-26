package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

var cmdMigrate = &cobra.Command{
	Use:   "migrate [from-backend] [to-backend]",
	Short: "Migrate passwords from one backend to another",
	Long: `Migrate passwords from one backend to another.

Backends: 1password, keyring, file, auto

Examples:
  # Migrate from keyring to 1Password
  veessh migrate keyring 1password

  # Migrate from file to keyring
  veessh migrate file keyring

  # Migrate all passwords to file backend
  veessh migrate auto file`,
	Args: cobra.ExactArgs(2),
	RunE: runMigrate,
}

func runMigrate(cmd *cobra.Command, args []string) error {
	fromBackend := credentials.BackendType(args[0])
	toBackend := credentials.BackendType(args[1])

	// Validate backend types
	validBackends := map[credentials.BackendType]bool{
		credentials.Backend1Password: true,
		credentials.BackendKeyring:   true,
		credentials.BackendFile:       true,
		credentials.BackendAuto:       false, // Can't use auto as source/dest
	}

	if !validBackends[fromBackend] {
		return fmt.Errorf("invalid source backend: %s (must be: 1password, keyring, or file)", fromBackend)
	}
	if !validBackends[toBackend] {
		return fmt.Errorf("invalid destination backend: %s (must be: 1password, keyring, or file)", toBackend)
	}

	if fromBackend == toBackend {
		return fmt.Errorf("source and destination backends are the same: %s", fromBackend)
	}

	// Load config to get all profile names
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles found. Nothing to migrate.")
		return nil
	}

	// Get source backend by directly creating it
	var sourceBackend credentials.Backend
	switch fromBackend {
	case credentials.Backend1Password:
		op := credentials.NewOnePasswordBackend("")
		if !op.IsAvailable() {
			return fmt.Errorf("1Password CLI not available (not installed or not signed in)")
		}
		sourceBackend = op
	case credentials.BackendKeyring:
		sourceBackend = credentials.NewKeyringBackend()
	case credentials.BackendFile:
		fileBackend, err := credentials.NewFileBackend()
		if err != nil {
			return fmt.Errorf("failed to initialize file backend: %w", err)
		}
		sourceBackend = fileBackend
	default:
		return fmt.Errorf("invalid source backend: %s", fromBackend)
	}

	// Get destination backend by directly creating it
	var destBackend credentials.Backend
	switch toBackend {
	case credentials.Backend1Password:
		op := credentials.NewOnePasswordBackend("")
		if !op.IsAvailable() {
			return fmt.Errorf("1Password CLI not available (not installed or not signed in)")
		}
		destBackend = op
	case credentials.BackendKeyring:
		destBackend = credentials.NewKeyringBackend()
	case credentials.BackendFile:
		fileBackend, err := credentials.NewFileBackend()
		if err != nil {
			return fmt.Errorf("failed to initialize file backend: %w", err)
		}
		destBackend = fileBackend
	default:
		return fmt.Errorf("invalid destination backend: %s", toBackend)
	}

	// Migrate passwords
	migrated := 0
	failed := 0
	skipped := 0

	fmt.Printf("Migrating passwords from %s to %s...\n\n", fromBackend, toBackend)

	for profileName := range cfg.Profiles {
		// Get password from source
		password, err := sourceBackend.GetPassword(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to read password for %s from %s: %v\n", profileName, fromBackend, err)
			failed++
			continue
		}

		if password == "" {
			// No password stored, skip
			skipped++
			continue
		}

		// Check if password already exists in destination
		existing, err := destBackend.GetPassword(profileName)
		if err == nil && existing != "" {
			fmt.Printf("⏭️  %s: Password already exists in destination, skipping\n", profileName)
			skipped++
			continue
		}

		// Write to destination
		if err := destBackend.SetPassword(profileName, password); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to write password for %s to %s: %v\n", profileName, toBackend, err)
			failed++
			continue
		}

		fmt.Printf("✅ %s: Migrated successfully\n", profileName)
		migrated++
	}

	fmt.Printf("\nMigration complete:\n")
	fmt.Printf("  ✅ Migrated: %d\n", migrated)
	fmt.Printf("  ⏭️  Skipped:  %d\n", skipped)
	fmt.Printf("  ❌ Failed:   %d\n", failed)

	if failed > 0 {
		return fmt.Errorf("migration completed with %d errors", failed)
	}

	return nil
}


