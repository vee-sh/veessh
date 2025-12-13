package connectors

import (
	"context"
	"os"
	"os/exec"

	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/util"
)

type ssmConnector struct{}

func (s *ssmConnector) Name() string { return "ssm" }

func (s *ssmConnector) Exec(ctx context.Context, p config.Profile, _ string) error {
	args := []string{"ssm", "start-session", "--target", p.InstanceID}

	if p.AWSRegion != "" {
		args = append(args, "--region", p.AWSRegion)
	}

	if len(p.ExtraArgs) > 0 {
		args = append(args, p.ExtraArgs...)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)

	// Set AWS_PROFILE if specified
	if p.AWSProfile != "" {
		cmd.Env = append(os.Environ(), "AWS_PROFILE="+p.AWSProfile)
	}

	return util.RunAttached(cmd)
}

func init() {
	Register(config.ProtocolSSM, &ssmConnector{})
}

