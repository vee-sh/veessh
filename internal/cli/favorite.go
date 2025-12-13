package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var favUnset bool

var cmdFavorite = &cobra.Command{
	Use:   "favorite <name>",
	Short: "Mark a profile as favorite (or --unset)",
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
		p, ok := cfg.GetProfile(name)
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}
		p.Favorite = !favUnset
		cfg.UpsertProfile(p)
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		state := "favorited"
		if favUnset {
			state = "unfavorited"
		}
		fmt.Printf("%s %q\n", state, name)
		return nil
	},
}

func init() {
	cmdFavorite.Flags().BoolVar(&favUnset, "unset", false, "unset favorite flag")
}
