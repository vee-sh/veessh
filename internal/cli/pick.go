package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/connectors"
	"github.com/vee-sh/veessh/internal/credentials"
	"github.com/vee-sh/veessh/internal/ui"
)

var (
	pickProtocol    string
	pickGroup       string
	pickPrint       bool
	pickFavorites   bool
	pickUseFZF      bool
	pickRecentFirst bool
	pickTags        []string
)

var cmdPick = &cobra.Command{
	Use:   "pick",
	Short: "Interactively pick a profile and connect",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		p, err := ui.PickProfileInteractive(cmd.Context(), cfg, pickProtocol, pickGroup, pickFavorites, pickUseFZF, pickRecentFirst, pickTags)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return context.Canceled
			}
			return err
		}
		if pickPrint {
			fmt.Println(p.Name)
			return nil
		}
		conn, err := connectors.Get(p.Protocol)
		if err != nil {
			return err
		}
		password, err := credentials.GetPassword(p.Name)
		if err != nil {
			// Non-fatal: log but continue (password might not be stored)
			fmt.Fprintf(os.Stderr, "Warning: failed to retrieve password: %v\n", err)
		}
		if err := conn.Exec(cmd.Context(), p, password); err != nil {
			if errors.Is(err, context.Canceled) {
				return context.Canceled
			}
			return err
		}
		// Update usage tracking
		p.LastUsed = time.Now()
		p.UseCount++
		cfg.UpsertProfile(p)
		if err := config.Save(cfgPath, cfg); err != nil {
			fmt.Printf("warning: failed to update usage stats: %v\n", err)
		}
		return nil
	},
}

func init() {
	cmdPick.Flags().StringVar(&pickProtocol, "type", "", "filter by protocol: ssh|sftp|telnet")
	cmdPick.Flags().StringVar(&pickGroup, "group", "", "filter by group")
	cmdPick.Flags().BoolVar(&pickFavorites, "favorites", false, "show only favorites")
	cmdPick.Flags().BoolVar(&pickUseFZF, "fzf", false, "use fzf if available; falls back to survey")
	cmdPick.Flags().BoolVar(&pickRecentFirst, "recent-first", false, "sort by last used desc")
	cmdPick.Flags().StringSliceVar(&pickTags, "tag", nil, "filter by tag(s), require all")
	cmdPick.Flags().BoolVar(&pickPrint, "print", false, "print selected profile name instead of connecting")
}
