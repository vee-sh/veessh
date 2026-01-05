package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/vee-sh/veessh/internal/audit"
	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/connectors"
	"github.com/vee-sh/veessh/internal/credentials"
)

// executeConnection handles common connection logic for connect, pick, and root commands
func executeConnection(ctx context.Context, p config.Profile, updateUsageStats bool) error {
	conn, err := connectors.Get(p.Protocol)
	if err != nil {
		return err
	}

	// Retrieve password (non-fatal if fails)
	password, err := credentials.GetPassword(p.Name)
	if err != nil {
		// Non-fatal: log but continue (password might not be stored)
		fmt.Fprintf(os.Stderr, "Warning: failed to retrieve password: %v\n", err)
		password = ""
	}

	// Audit log: connection start
	startTime := time.Now()
	audit.LogConnect(p.Name, string(p.Protocol), p.Host, p.Username)

	// Execute connection
	var exitCode int
	var connErr error
	if err := conn.Exec(ctx, p, password); err != nil {
		connErr = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		// Audit log: connection end with error
		audit.LogDisconnect(p.Name, string(p.Protocol), p.Host, p.Username, startTime, exitCode, connErr)

		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}

	// Audit log: successful disconnect
	audit.LogDisconnect(p.Name, string(p.Protocol), p.Host, p.Username, startTime, 0, nil)

	// Update usage tracking if requested
	if updateUsageStats {
		p.LastUsed = time.Now()
		p.UseCount++
		
		cfgPath, err := config.DefaultPath()
		if err == nil {
			cfg, err := config.Load(cfgPath)
			if err == nil {
				cfg.UpsertProfile(p)
				if err := config.Save(cfgPath, cfg); err != nil {
					// Non-fatal, but report it
					fmt.Printf("warning: failed to update usage stats: %v\n", err)
				}
			}
		}
	}

	return nil
}
