package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

var (
	runTTY bool
)

var cmdRun = &cobra.Command{
	Use:   "run <profile> <command> [args...]",
	Short: "Execute a command on a remote host",
	Long: `Execute a command on a remote host via SSH without an interactive shell.

Examples:
  veessh run mybox uptime
  veessh run mybox "df -h"
  veessh run mybox ls -la /var/log
  veessh run mybox --tty top           # Force TTY allocation`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		remoteCmd := args[1:]

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

		if p.Protocol != config.ProtocolSSH {
			return fmt.Errorf("run command only supports SSH profiles (got %s)", p.Protocol)
		}

		return executeRemoteCommand(cmd.Context(), p, remoteCmd)
	},
}

func executeRemoteCommand(ctx context.Context, p config.Profile, remoteCmd []string) error {
	sshArgs := []string{}

	if p.Port > 0 {
		sshArgs = append(sshArgs, "-p", strconv.Itoa(p.Port))
	}
	if p.Username != "" {
		sshArgs = append(sshArgs, "-l", p.Username)
	}
	if p.IdentityFile != "" {
		sshArgs = append(sshArgs, "-i", p.IdentityFile)
	}
	if p.ProxyJump != "" {
		sshArgs = append(sshArgs, "-J", p.ProxyJump)
	}
	if runTTY {
		sshArgs = append(sshArgs, "-t")
	}

	// Add extra args from profile
	if len(p.ExtraArgs) > 0 {
		sshArgs = append(sshArgs, p.ExtraArgs...)
	}

	sshArgs = append(sshArgs, p.Host)

	// Add the remote command
	sshArgs = append(sshArgs, strings.Join(remoteCmd, " "))

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	if err := util.RunAttached(cmd); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		// Check for exit code and surface it
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func init() {
	cmdRun.Flags().BoolVarP(&runTTY, "tty", "t", false, "force TTY allocation (for interactive commands)")
}

