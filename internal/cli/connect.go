package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/audit"
	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/connectors"
	"github.com/vee-sh/veessh/internal/credentials"
)

var (
	connectNoForward  bool
	connectWithForward bool
)

var cmdConnect = &cobra.Command{
	Use:   "connect <name>",
	Short: "Connect using a profile",
	Long: `Connect to a remote host using a saved profile.

Port forwarding can be toggled at connect time:
  --forward     Enable port forwards defined in profile
  --no-forward  Disable port forwards for this connection

Examples:
  veessh connect mybox
  veessh connect mybox --no-forward    # Skip port forwarding
  veessh connect mybox --forward       # Ensure forwards are enabled`,
	Args: cobra.ExactArgs(1),
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

		// Handle port-forward toggle
		if connectNoForward {
			p.LocalForwards = nil
			p.RemoteForwards = nil
			p.DynamicForwards = nil
		}

		conn, err := connectors.Get(p.Protocol)
		if err != nil {
			return err
		}
		password, err := credentials.GetPassword(name)
		if err != nil {
			// Non-fatal: log but continue (password might not be stored)
			fmt.Fprintf(os.Stderr, "Warning: failed to retrieve password: %v\n", err)
		}

		// Audit log: connection start
		startTime := time.Now()
		audit.LogConnect(p.Name, string(p.Protocol), p.Host, p.Username)

		// Use command context to support Ctrl+C cancel
		var exitCode int
		var connErr error
		if err := conn.Exec(cmd.Context(), p, password); err != nil {
			connErr = err
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			// Audit log: connection end with error
			audit.LogDisconnect(p.Name, string(p.Protocol), p.Host, p.Username, startTime, exitCode, connErr)

			if errors.Is(err, context.Canceled) {
				return context.Canceled
			}
			return err
		}

		// Audit log: successful disconnect
		audit.LogDisconnect(p.Name, string(p.Protocol), p.Host, p.Username, startTime, 0, nil)

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

func init() {
	cmdConnect.Flags().BoolVar(&connectNoForward, "no-forward", false, "disable port forwarding for this connection")
	cmdConnect.Flags().BoolVar(&connectWithForward, "forward", false, "enable port forwarding (default if configured)")
}
