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
	scpRecursive bool
	scpPreserve  bool
)

var cmdScp = &cobra.Command{
	Use:   "scp <profile>:<remote-path> <local-path> | <local-path> <profile>:<remote-path>",
	Short: "Copy files to/from a remote host using profile credentials",
	Long: `Transfer files using scp with profile credentials.

The profile name is used as a prefix before the colon, similar to standard scp syntax.

Examples:
  # Download from remote
  veessh scp mybox:/var/log/app.log ./app.log
  veessh scp mybox:/etc/nginx/ ./nginx-backup/ -r

  # Upload to remote
  veessh scp ./config.yaml mybox:/app/config.yaml
  veessh scp ./dist/ mybox:/var/www/html/ -r

  # Preserve timestamps and permissions
  veessh scp -rp mybox:/backup/ ./local-backup/`,
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

		// Parse source and destination to find profile
		srcProfile, srcPath, srcIsRemote := parseScpPath(src)
		dstProfile, dstPath, dstIsRemote := parseScpPath(dst)

		if srcIsRemote && dstIsRemote {
			return fmt.Errorf("cannot copy between two remote hosts")
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
			return fmt.Errorf("scp only works with SSH/SFTP profiles (got %s)", p.Protocol)
		}

		return executeScp(cmd.Context(), p, srcPath, dstPath, srcIsRemote, dstIsRemote)
	},
}

func parseScpPath(path string) (profile, remotePath string, isRemote bool) {
	// Check for profile:path format (but not Windows drive letters like C:\)
	if idx := strings.Index(path, ":"); idx > 0 {
		// Make sure it's not a Windows path (single letter before colon)
		if idx > 1 || (idx == 1 && len(path) > 2 && path[2] != '\\' && path[2] != '/') {
			return path[:idx], path[idx+1:], true
		}
	}
	return "", path, false
}

func executeScp(ctx context.Context, p config.Profile, srcPath, dstPath string, srcIsRemote, dstIsRemote bool) error {
	scpArgs := []string{}

	if scpRecursive {
		scpArgs = append(scpArgs, "-r")
	}
	if scpPreserve {
		scpArgs = append(scpArgs, "-p")
	}

	if p.Port > 0 && p.Port != 22 {
		scpArgs = append(scpArgs, "-P", strconv.Itoa(p.Port))
	}
	if p.IdentityFile != "" {
		scpArgs = append(scpArgs, "-i", p.IdentityFile)
	}
	if p.ProxyJump != "" {
		scpArgs = append(scpArgs, "-o", "ProxyJump="+p.ProxyJump)
	}

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

	scpArgs = append(scpArgs, src, dst)

	cmd := exec.CommandContext(ctx, "scp", scpArgs...)
	if err := util.RunAttached(cmd); err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}
	return nil
}

func init() {
	cmdScp.Flags().BoolVarP(&scpRecursive, "recursive", "r", false, "recursively copy directories")
	cmdScp.Flags().BoolVarP(&scpPreserve, "preserve", "p", false, "preserve timestamps and permissions")
}

