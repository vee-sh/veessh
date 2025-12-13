package connectors

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

type moshConnector struct{}

func (m *moshConnector) Name() string { return "mosh" }

func (m *moshConnector) Exec(ctx context.Context, p config.Profile, _ string) error {
	args := []string{}

	// Mosh uses --ssh for ssh options
	sshArgs := ""
	if p.Port > 0 && p.Port != 22 {
		sshArgs += " -p " + strconv.Itoa(p.Port)
	}
	if p.IdentityFile != "" {
		sshArgs += " -i " + p.IdentityFile
	}
	if p.ProxyJump != "" {
		sshArgs += " -J " + p.ProxyJump
	}
	if sshArgs != "" {
		args = append(args, "--ssh=ssh"+sshArgs)
	}

	if p.MoshServer != "" {
		args = append(args, "--server="+p.MoshServer)
	}

	if len(p.ExtraArgs) > 0 {
		args = append(args, p.ExtraArgs...)
	}

	target := p.Host
	if p.Username != "" {
		target = p.Username + "@" + target
	}
	args = append(args, target)

	// Mosh can pass a remote command after --
	if p.RemoteCommand != "" {
		args = append(args, "--", p.RemoteCommand)
	}

	cmd := exec.CommandContext(ctx, "mosh", args...)
	return util.RunAttached(cmd)
}

func init() {
	Register(config.ProtocolMosh, &moshConnector{})
}

