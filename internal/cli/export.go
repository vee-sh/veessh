package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vee-sh/veessh/internal/config"
)

var exportFile string

var cmdExport = &cobra.Command{
	Use:   "export",
	Short: "Export profiles to a YAML file (no passwords)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if exportFile == "" {
			return errors.New("--file is required")
		}
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := os.WriteFile(exportFile, data, 0o600); err != nil {
			return err
		}
		fmt.Printf("exported %d profiles to %s\n", len(cfg.Profiles), exportFile)
		return nil
	},
}

func init() {
	cmdExport.Flags().StringVar(&exportFile, "file", "", "output file path")
}
