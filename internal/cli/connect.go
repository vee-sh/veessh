package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
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

		return executeConnection(cmd.Context(), p, true)
	},
}

func init() {
	cmdConnect.Flags().BoolVar(&connectNoForward, "no-forward", false, "disable port forwarding for this connection")
	cmdConnect.Flags().BoolVar(&connectWithForward, "forward", false, "enable port forwarding (default if configured)")
}
