package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

var rmDeletePassword bool

var cmdRemove = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		if !cfg.DeleteProfile(name) {
			return fmt.Errorf("profile %q not found", name)
		}
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		if rmDeletePassword {
			_ = credentials.DeletePassword(name)
		}
		fmt.Printf("Removed profile %q\n", name)
		return nil
	},
}

func init() {
	cmdRemove.Flags().BoolVar(&rmDeletePassword, "delete-password", false, "also delete any stored password from keychain")
}
