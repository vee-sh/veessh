package connectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/util"
)

type sshConnector struct{}

func (s *sshConnector) Name() string { return "ssh" }

func (s *sshConnector) Exec(ctx context.Context, p config.Profile, password string) error {
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

	// If password is provided, try to use sshpass or SSH_ASKPASS
	if password != "" && p.IdentityFile == "" && !p.UseAgent {
		return s.execWithPassword(ctx, args, password)
	}

	cmd := exec.CommandContext(ctx, "ssh", args...)
	return util.RunAttached(cmd)
}

// execWithPassword executes SSH with password authentication
func (s *sshConnector) execWithPassword(ctx context.Context, sshArgs []string, password string) error {
	// Try sshpass first (if available) - this is the most reliable method
	if sshpassPath := findExecutable("sshpass"); sshpassPath != "" {
		args := []string{"-e", "ssh"}
		args = append(args, sshArgs...)
		cmd := exec.CommandContext(ctx, sshpassPath, args...)
		cmd.Env = append(os.Environ(), "SSHPASS="+password)
		return util.RunAttached(cmd)
	}

	// sshpass not available - fall back to interactive prompt
	// Note: SSH doesn't support password via command line for security reasons
	fmt.Fprintf(os.Stderr, "Note: Password is stored but sshpass is not installed.\n")
	fmt.Fprintf(os.Stderr, "Install sshpass for automatic password injection: brew install hudochenkov/sshpass/sshpass\n")
	fmt.Fprintf(os.Stderr, "You will be prompted for the password below.\n\n")
	
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	return util.RunAttached(cmd)
}

// findExecutable finds an executable in PATH
func findExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// createSSHAskPassScript creates a temporary script that outputs the password
func createSSHAskPassScript(password string) (string, error) {
	tmpFile, err := os.CreateTemp("", "veessh-askpass-*")
	if err != nil {
		return "", err
	}
	scriptPath := tmpFile.Name()
	tmpFile.Close()

	// Write the script
	script := "#!/bin/sh\necho \"" + strings.ReplaceAll(password, "\"", "\\\"") + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		os.Remove(scriptPath)
		return "", err
	}

	return scriptPath, nil
}

// buildRemoteCommand constructs the command to run on the remote host
func buildRemoteCommand(p config.Profile) string {
	var parts []string

	// Change to remote directory (properly quoted for spaces/special chars)
	if p.RemoteDir != "" {
		parts = append(parts, "cd "+shellQuote(p.RemoteDir))
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

// shellQuote quotes a string for safe use in a shell command.
// Uses single quotes with proper escaping for embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func init() {
	Register(config.ProtocolSSH, &sshConnector{})
}
