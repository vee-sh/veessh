package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var (
	cloneHost  string
	clonePort  int
	cloneUser  string
	cloneGroup string
)

var cmdClone = &cobra.Command{
	Use:   "clone <source> <new-name>",
	Short: "Clone an existing profile with a new name",
	Long: `Clone an existing profile to create a new one.
Optionally override host, port, user, or group in the new profile.

Examples:
  veessh clone prod-server staging-server
  veessh clone prod-server staging-server --host staging.example.com
  veessh clone mybox mybox-alt --port 2222 --user admin`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceName := args[0]
		newName := args[1]

		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		source, ok := cfg.GetProfile(sourceName)
		if !ok {
			return fmt.Errorf("source profile %q not found", sourceName)
		}

		if _, exists := cfg.GetProfile(newName); exists {
			return fmt.Errorf("profile %q already exists", newName)
		}

		// Create new profile based on source
		newProfile := source
		newProfile.Name = newName
		newProfile.UseCount = 0
		newProfile.LastUsed = source.LastUsed // Reset or keep? Let's reset
		newProfile.Favorite = false

		// Apply overrides
		if cmd.Flags().Changed("host") {
			newProfile.Host = cloneHost
		}
		if cmd.Flags().Changed("port") {
			newProfile.Port = clonePort
		}
		if cmd.Flags().Changed("user") {
			newProfile.Username = cloneUser
		}
		if cmd.Flags().Changed("group") {
			newProfile.Group = cloneGroup
		}

		if err := (&newProfile).Validate(); err != nil {
			return err
		}

		cfg.UpsertProfile(newProfile)
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}

		fmt.Printf("Cloned %q -> %q\n", sourceName, newName)
		return nil
	},
}

func init() {
	cmdClone.Flags().StringVar(&cloneHost, "host", "", "override host in new profile")
	cmdClone.Flags().IntVar(&clonePort, "port", 0, "override port in new profile")
	cmdClone.Flags().StringVar(&cloneUser, "user", "", "override username in new profile")
	cmdClone.Flags().StringVar(&cloneGroup, "group", "", "override group in new profile")
}

