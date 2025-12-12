package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/connectors"
	"github.com/alex-vee-sh/veessh/internal/credentials"
)

var cmdConnect = &cobra.Command{
	Use:   "connect <name>",
	Short: "Connect using a profile",
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
		conn, err := connectors.Get(p.Protocol)
		if err != nil {
			return err
		}
		password, _ := credentials.GetPassword(name)
		// Use command context to support Ctrl+C cancel
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
			// Non-fatal, but report it
			fmt.Printf("warning: failed to update usage stats: %v\n", err)
		}
		return nil
	},
}
