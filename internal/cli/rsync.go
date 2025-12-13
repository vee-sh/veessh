package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

var (
	rsyncDelete   bool
	rsyncDryRun   bool
	rsyncVerbose  bool
	rsyncExclude  []string
	rsyncProgress bool
)

var cmdRsync = &cobra.Command{
	Use:   "rsync <profile>:<remote-path> <local-path> | <local-path> <profile>:<remote-path>",
	Short: "Sync directories with a remote host using rsync",
	Long: `Efficiently sync directories using rsync with profile credentials.

Rsync is more efficient than scp for syncing directories as it only
transfers changed files.

Examples:
  # Sync remote to local
  veessh rsync mybox:/var/www/html/ ./local-www/

  # Sync local to remote
  veessh rsync ./dist/ mybox:/var/www/html/

  # Sync with delete (remove files not in source)
  veessh rsync --delete ./dist/ mybox:/var/www/html/

  # Dry run to preview changes
  veessh rsync --dry-run ./dist/ mybox:/var/www/html/

  # Exclude patterns
  veessh rsync --exclude "*.log" --exclude ".git/" ./project/ mybox:/app/`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dst := args[1]

		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		// Parse source and destination
		srcProfile, srcPath, srcIsRemote := parseScpPath(src)
		dstProfile, dstPath, dstIsRemote := parseScpPath(dst)

		if srcIsRemote && dstIsRemote {
			return fmt.Errorf("cannot sync between two remote hosts")
		}
		if !srcIsRemote && !dstIsRemote {
			return fmt.Errorf("at least one path must be remote (profile:path)")
		}

		var profileName string
		if srcIsRemote {
			profileName = srcProfile
		} else {
			profileName = dstProfile
		}

		p, ok := cfg.GetProfile(profileName)
		if !ok {
			return fmt.Errorf("profile %q not found", profileName)
		}

		if p.Protocol != config.ProtocolSSH && p.Protocol != config.ProtocolSFTP {
			return fmt.Errorf("rsync only works with SSH/SFTP profiles (got %s)", p.Protocol)
		}

		return executeRsync(cmd.Context(), p, srcPath, dstPath, srcIsRemote, dstIsRemote)
	},
}

func executeRsync(ctx context.Context, p config.Profile, srcPath, dstPath string, srcIsRemote, dstIsRemote bool) error {
	rsyncArgs := []string{"-a"} // Archive mode (recursive, preserves permissions, etc.)

	if rsyncVerbose {
		rsyncArgs = append(rsyncArgs, "-v")
	}
	if rsyncProgress {
		rsyncArgs = append(rsyncArgs, "--progress")
	}
	if rsyncDelete {
		rsyncArgs = append(rsyncArgs, "--delete")
	}
	if rsyncDryRun {
		rsyncArgs = append(rsyncArgs, "--dry-run")
	}
	for _, exc := range rsyncExclude {
		rsyncArgs = append(rsyncArgs, "--exclude", exc)
	}

	// Build SSH command for rsync with proper quoting
	sshParts := []string{"ssh"}
	if p.Port > 0 && p.Port != 22 {
		sshParts = append(sshParts, "-p", strconv.Itoa(p.Port))
	}
	if p.IdentityFile != "" {
		sshParts = append(sshParts, "-i", shellQuoteForRsync(p.IdentityFile))
	}
	if p.ProxyJump != "" {
		sshParts = append(sshParts, "-J", shellQuoteForRsync(p.ProxyJump))
	}
	rsyncArgs = append(rsyncArgs, "-e", strings.Join(sshParts, " "))

	// Build remote path with user@host prefix
	remotePrefix := p.Host
	if p.Username != "" {
		remotePrefix = p.Username + "@" + p.Host
	}

	var src, dst string
	if srcIsRemote {
		src = remotePrefix + ":" + srcPath
		dst = dstPath
	} else {
		src = srcPath
		dst = remotePrefix + ":" + dstPath
	}

	// Ensure trailing slash for directory sync if not present
	if !strings.HasSuffix(src, "/") && !strings.HasSuffix(src, ":") {
		// Check if it looks like a directory path
		if strings.HasSuffix(srcPath, "/") || srcPath == "." || srcPath == ".." {
			src += "/"
		}
	}

	rsyncArgs = append(rsyncArgs, src, dst)

	cmd := exec.CommandContext(ctx, "rsync", rsyncArgs...)
	if err := util.RunAttached(cmd); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}
	return nil
}

// shellQuoteForRsync quotes a string for use in rsync's -e ssh command
// Uses single quotes and escapes embedded single quotes
func shellQuoteForRsync(s string) string {
	// If no special characters, return as-is
	if !strings.ContainsAny(s, " \t'\"\\$`!") {
		return s
	}
	// Use single quotes and escape any single quotes within
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func init() {
	cmdRsync.Flags().BoolVar(&rsyncDelete, "delete", false, "delete files in destination not in source")
	cmdRsync.Flags().BoolVarP(&rsyncDryRun, "dry-run", "n", false, "show what would be transferred")
	cmdRsync.Flags().BoolVar(&rsyncVerbose, "verbose", false, "verbose output")
	cmdRsync.Flags().BoolVar(&rsyncProgress, "progress", false, "show transfer progress")
	cmdRsync.Flags().StringSliceVar(&rsyncExclude, "exclude", nil, "exclude pattern (repeatable)")
}

