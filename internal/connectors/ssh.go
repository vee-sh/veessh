package connectors

import (
	"context"
	"os/exec"
	"strconv"
	"strings"

	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/util"
)

type sshConnector struct{}

func (s *sshConnector) Name() string { return "ssh" }

func (s *sshConnector) Exec(ctx context.Context, p config.Profile, _ string) error {
	args := []string{}
	if p.Port > 0 {
		args = append(args, "-p", strconv.Itoa(p.Port))
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
	for _, lf := range p.LocalForwards {
		if lf != "" {
			args = append(args, "-L", lf)
		}
	}
	for _, rf := range p.RemoteForwards {
		if rf != "" {
			args = append(args, "-R", rf)
		}
	}
	for _, df := range p.DynamicForwards {
		if df != "" {
			args = append(args, "-D", df)
		}
	}

	// On-connect environment variables
	for _, env := range p.SetEnv {
		if env != "" {
			args = append(args, "-o", "SetEnv="+env)
		}
	}

	if len(p.ExtraArgs) > 0 {
		args = append(args, p.ExtraArgs...)
	}

	// Request TTY if we have a remote command
	if p.RemoteCommand != "" || p.RemoteDir != "" {
		args = append(args, "-t")
	}

	args = append(args, p.Host)

	// Build remote command if specified
	remoteCmd := buildRemoteCommand(p)
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}

	cmd := exec.CommandContext(ctx, "ssh", args...)
	return util.RunAttached(cmd)
}

// buildRemoteCommand constructs the command to run on the remote host
func buildRemoteCommand(p config.Profile) string {
	var parts []string

	// Change to remote directory
	if p.RemoteDir != "" {
		parts = append(parts, "cd "+p.RemoteDir)
	}

	// Execute remote command or start shell
	if p.RemoteCommand != "" {
		parts = append(parts, p.RemoteCommand)
	} else if p.RemoteDir != "" {
		// If we changed directory but have no command, start a shell
		parts = append(parts, "exec $SHELL -l")
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " && ")
}

func init() {
	Register(config.ProtocolSSH, &sshConnector{})
}
