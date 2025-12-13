package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

var copyIdKey string

var cmdCopyId = &cobra.Command{
	Use:   "copy-id <profile>",
	Short: "Copy SSH public key to remote host",
	Long: `Deploy your SSH public key to a remote host's authorized_keys.

This is equivalent to ssh-copy-id but uses veessh profile credentials.

Examples:
  # Copy default key (~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub)
  veessh copy-id mybox

  # Copy specific key
  veessh copy-id mybox --key ~/.ssh/mykey.pub`,
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

		if p.Protocol != config.ProtocolSSH {
			return fmt.Errorf("copy-id only works with SSH profiles (got %s)", p.Protocol)
		}

		keyPath, err := findPublicKey(copyIdKey)
		if err != nil {
			return err
		}

		fmt.Printf("Copying %s to %s...\n", keyPath, name)
		return executeCopyId(cmd.Context(), p, keyPath)
	},
}

func findPublicKey(specified string) (string, error) {
	if specified != "" {
		expanded := expandHomePath(specified)
		if _, err := os.Stat(expanded); err != nil {
			return "", fmt.Errorf("key file not found: %s", specified)
		}
		return expanded, nil
	}

	// Try common key locations
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_ecdsa.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
		filepath.Join(home, ".ssh", "id_dsa.pub"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no SSH public key found. Specify one with --key or generate with: ssh-keygen -t ed25519")
}

func executeCopyId(ctx context.Context, p config.Profile, keyPath string) error {
	// Read the public key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key: %w", err)
	}

	// Build the remote command to append to authorized_keys
	// This is what ssh-copy-id does internally
	remoteCmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo %q >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`,
		string(keyData),
	)

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

	sshArgs = append(sshArgs, p.Host, remoteCmd)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	if err := util.RunAttached(cmd); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}

	fmt.Println("Key installed successfully!")
	return nil
}

func init() {
	cmdCopyId.Flags().StringVar(&copyIdKey, "key", "", "path to public key file (default: auto-detect)")
}

