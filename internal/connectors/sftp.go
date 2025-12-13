package connectors

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

type sftpConnector struct{}

func (s *sftpConnector) Name() string { return "sftp" }

func (s *sftpConnector) Exec(ctx context.Context, p config.Profile, _ string) error {
	args := []string{}
	if p.Port > 0 {
		args = append(args, "-P", strconv.Itoa(p.Port))
	}
	if p.IdentityFile != "" {
		args = append(args, "-i", p.IdentityFile)
	}
	if p.ProxyJump != "" {
		args = append(args, "-o", "ProxyJump="+p.ProxyJump)
	}
	for _, lf := range p.LocalForwards {
		if lf != "" {
			args = append(args, "-o", "LocalForward="+lf)
		}
	}
	for _, rf := range p.RemoteForwards {
		if rf != "" {
			args = append(args, "-o", "RemoteForward="+rf)
		}
	}
	for _, df := range p.DynamicForwards {
		if df != "" {
			args = append(args, "-o", "DynamicForward="+df)
		}
	}
	if len(p.ExtraArgs) > 0 {
		args = append(args, p.ExtraArgs...)
	}
	target := p.Host
	if p.Username != "" {
		target = p.Username + "@" + target
	}
	args = append(args, target)
	cmd := exec.CommandContext(ctx, "sftp", args...)
	return util.RunAttached(cmd)
}

func init() {
	Register(config.ProtocolSFTP, &sftpConnector{})
}
