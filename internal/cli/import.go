package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/alex-vee-sh/veessh/internal/config"
)

var importFile string
var importOverwrite bool

var cmdImport = &cobra.Command{
	Use:   "import",
	Short: "Import profiles from a YAML file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFile == "" {
			return errors.New("--file is required")
		}
		data, err := os.ReadFile(importFile)
		if err != nil {
			return err
		}
		var incoming config.Config
		if err := yaml.Unmarshal(data, &incoming); err != nil {
			return err
		}
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		if cfg.Profiles == nil {
			cfg.Profiles = map[string]config.Profile{}
		}
		imported := 0
		skipped := 0
		for name, p := range incoming.Profiles {
			if _, exists := cfg.Profiles[name]; exists && !importOverwrite {
				skipped++
				continue
			}
			cfg.Profiles[name] = p
			imported++
		}
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Printf("imported %d, skipped %d\n", imported, skipped)
		return nil
	},
}

func init() {
	cmdImport.Flags().StringVar(&importFile, "file", "", "input YAML file path")
	cmdImport.Flags().BoolVar(&importOverwrite, "overwrite", false, "overwrite existing profiles")
}
