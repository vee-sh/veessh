package connectors

import (
	"context"
	"os/exec"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

type gcloudConnector struct{}

func (g *gcloudConnector) Name() string { return "gcloud" }

func (g *gcloudConnector) Exec(ctx context.Context, p config.Profile, _ string) error {
	args := []string{"compute", "ssh"}

	// Instance name is stored in Host field for gcloud
	if p.Username != "" {
		args = append(args, p.Username+"@"+p.Host)
	} else {
		args = append(args, p.Host)
	}

	if p.GCPProject != "" {
		args = append(args, "--project", p.GCPProject)
	}
	if p.GCPZone != "" {
		args = append(args, "--zone", p.GCPZone)
	}

	// Add tunnel through IAP if configured
	if p.GCPUseTunnel {
		args = append(args, "--tunnel-through-iap")
	}

	if len(p.ExtraArgs) > 0 {
		args = append(args, p.ExtraArgs...)
	}

	// Pass remote command if specified
	if p.RemoteCommand != "" {
		args = append(args, "--command", p.RemoteCommand)
	}

	cmd := exec.CommandContext(ctx, "gcloud", args...)
	return util.RunAttached(cmd)
}

func init() {
	Register(config.ProtocolGCloud, &gcloudConnector{})
}

