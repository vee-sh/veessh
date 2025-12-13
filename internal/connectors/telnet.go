package connectors

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

type telnetConnector struct{}

func (t *telnetConnector) Name() string { return "telnet" }

func (t *telnetConnector) Exec(ctx context.Context, p config.Profile, _ string) error {
	args := []string{p.Host}
	if p.Port > 0 {
		args = append(args, strconv.Itoa(p.Port))
	}
	cmd := exec.CommandContext(ctx, "telnet", args...)
	return util.RunAttached(cmd)
}

func init() {
	Register(config.ProtocolTelnet, &telnetConnector{})
}
