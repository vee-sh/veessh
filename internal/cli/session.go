package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/config"
)

var (
	sessionName   string
	sessionLayout string
)

var cmdSession = &cobra.Command{
	Use:   "session <profile1> [profile2] [profile3] ...",
	Short: "Open multiple profiles in tmux windows/panes",
	Long: `Open multiple SSH connections in a tmux session.

Each profile becomes a separate tmux window. Use --layout to create panes instead.

Examples:
  # Open profiles in separate tmux windows
  veessh session web-server db-server cache-server

  # Open in a named session
  veessh session prod-web prod-db --name prod-cluster

  # Open profiles as panes in a single window (tiled layout)
  veessh session web1 web2 web3 --layout tiled

  # Horizontal split layout
  veessh session master worker --layout even-horizontal`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if tmux is available
		if _, err := exec.LookPath("tmux"); err != nil {
			return fmt.Errorf("tmux is required for session command")
		}

		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		// Validate all profiles exist
		var profiles []config.Profile
		for _, name := range args {
			p, ok := cfg.GetProfile(name)
			if !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			if p.Protocol != config.ProtocolSSH {
				return fmt.Errorf("session only supports SSH profiles (got %s for %s)", p.Protocol, name)
			}
			profiles = append(profiles, p)
		}

		sessName := sessionName
		if sessName == "" {
			sessName = "veessh-" + profiles[0].Name
		}

		if sessionLayout != "" {
			return createTmuxPanes(cmd.Context(), sessName, profiles)
		}
		return createTmuxWindows(cmd.Context(), sessName, profiles)
	},
}

func buildSSHCommand(p config.Profile) string {
	args := []string{"ssh"}
	if p.Port > 0 && p.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", p.Port))
	}
	if p.Username != "" {
		args = append(args, "-l", p.Username)
	}
	if p.IdentityFile != "" {
		args = append(args, "-i", p.IdentityFile)
	}
	if p.ProxyJump != "" {
		args = append(args, "-J", p.ProxyJump)
	}
	args = append(args, p.Host)
	return strings.Join(args, " ")
}

func createTmuxWindows(ctx context.Context, sessName string, profiles []config.Profile) error {
	// Check if session already exists
	checkCmd := exec.CommandContext(ctx, "tmux", "has-session", "-t", sessName)
	sessionExists := checkCmd.Run() == nil

	if sessionExists {
		fmt.Printf("Session %q already exists. Attaching...\n", sessName)
		return attachTmux(ctx, sessName)
	}

	// Create new session with first profile
	firstSSH := buildSSHCommand(profiles[0])
	newCmd := exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", sessName, "-n", profiles[0].Name, firstSSH)
	if err := newCmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Add windows for remaining profiles
	for _, p := range profiles[1:] {
		sshCmd := buildSSHCommand(p)
		winCmd := exec.CommandContext(ctx, "tmux", "new-window", "-t", sessName, "-n", p.Name, sshCmd)
		if err := winCmd.Run(); err != nil {
			return fmt.Errorf("failed to create window for %s: %w", p.Name, err)
		}
	}

	// Select first window
	exec.CommandContext(ctx, "tmux", "select-window", "-t", sessName+":0").Run()

	fmt.Printf("Created session %q with %d windows\n", sessName, len(profiles))
	return attachTmux(ctx, sessName)
}

func createTmuxPanes(ctx context.Context, sessName string, profiles []config.Profile) error {
	// Check if session already exists
	checkCmd := exec.CommandContext(ctx, "tmux", "has-session", "-t", sessName)
	sessionExists := checkCmd.Run() == nil

	if sessionExists {
		fmt.Printf("Session %q already exists. Attaching...\n", sessName)
		return attachTmux(ctx, sessName)
	}

	// Create new session with first profile
	firstSSH := buildSSHCommand(profiles[0])
	newCmd := exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", sessName, firstSSH)
	if err := newCmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Split panes for remaining profiles
	for _, p := range profiles[1:] {
		sshCmd := buildSSHCommand(p)
		splitCmd := exec.CommandContext(ctx, "tmux", "split-window", "-t", sessName, sshCmd)
		if err := splitCmd.Run(); err != nil {
			return fmt.Errorf("failed to create pane for %s: %w", p.Name, err)
		}
	}

	// Apply layout
	layoutCmd := exec.CommandContext(ctx, "tmux", "select-layout", "-t", sessName, sessionLayout)
	layoutCmd.Run()

	fmt.Printf("Created session %q with %d panes (%s layout)\n", sessName, len(profiles), sessionLayout)
	return attachTmux(ctx, sessName)
}

func attachTmux(ctx context.Context, sessName string) error {
	// Check if we're already in tmux
	if os.Getenv("TMUX") != "" {
		// Switch to session instead of attach
		switchCmd := exec.CommandContext(ctx, "tmux", "switch-client", "-t", sessName)
		switchCmd.Stdin = os.Stdin
		switchCmd.Stdout = os.Stdout
		switchCmd.Stderr = os.Stderr
		if err := switchCmd.Run(); err != nil {
			if errors.Is(err, context.Canceled) {
				return context.Canceled
			}
			return err
		}
		return nil
	}

	attachCmd := exec.CommandContext(ctx, "tmux", "attach-session", "-t", sessName)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	if err := attachCmd.Run(); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}
	return nil
}

func init() {
	cmdSession.Flags().StringVarP(&sessionName, "name", "n", "", "tmux session name (default: veessh-<first-profile>)")
	cmdSession.Flags().StringVarP(&sessionLayout, "layout", "l", "", "create panes with layout: tiled, even-horizontal, even-vertical, main-horizontal, main-vertical")
}

