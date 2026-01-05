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

	// If password is provided and no identity file, configure SSH for password auth
	// Note: We inject password even if UseAgent is true, as SSH may fall back
	// to password auth if the agent doesn't have the right key
	if password != "" && p.IdentityFile == "" {
		// Check if sshpass is available and warn if not
		if findExecutable("sshpass") == "" {
			fmt.Fprintf(os.Stderr, "⚠️  Password is stored but 'sshpass' is not installed.\n")
			fmt.Fprintf(os.Stderr, "   Install it for automatic password injection:\n")
			fmt.Fprintf(os.Stderr, "   macOS:   brew install hudochenkov/sshpass/sshpass\n")
			fmt.Fprintf(os.Stderr, "   Linux:   sudo apt-get install sshpass  (or sudo yum install sshpass)\n")
			fmt.Fprintf(os.Stderr, "   You will be prompted for the password below.\n\n")
		}
		// When using password auth, prefer password over agent/keyboard-interactive
		// This ensures sshpass can inject the password reliably
		// IMPORTANT: These options must come BEFORE the host argument
		// Disable all other auth methods to force password-only
		args = append(args, "-o", "PreferredAuthentications=password")
		args = append(args, "-o", "PubkeyAuthentication=no")
		args = append(args, "-o", "ChallengeResponseAuthentication=no")
		// Add timeout to prevent hanging
		args = append(args, "-o", "ConnectTimeout=10")
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

	// If password is provided, use execWithPassword
	if password != "" && p.IdentityFile == "" {
		return s.execWithPassword(ctx, args, password, p.Name)
	}

	cmd := exec.CommandContext(ctx, "ssh", args...)
	return util.RunAttached(cmd)
}

// execWithPassword executes SSH with password authentication
func (s *sshConnector) execWithPassword(ctx context.Context, sshArgs []string, password string, profileName string) error {
	// Verify password is not empty
	if password == "" {
		fmt.Fprintf(os.Stderr, "Warning: Password is empty. You will be prompted.\n")
		cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
		return util.RunAttached(cmd)
	}

	// Try sshpass first (if available) - this is the most reliable method
	if sshpassPath := findExecutable("sshpass"); sshpassPath != "" {
		// Clean password: remove any trailing newlines/whitespace that might cause issues
		cleanPassword := strings.TrimSpace(password)
		
		// Use -e flag with environment variable (more secure, avoids password in process list)
		args := []string{"-e", "ssh"}
		args = append(args, sshArgs...)
		cmd := exec.CommandContext(ctx, sshpassPath, args...)
		cmd.Env = append(os.Environ(), "SSHPASS="+cleanPassword)
		err := util.RunAttached(cmd)
		if err != nil {
			// Check if it's an authentication failure
			if exitErr, ok := err.(*exec.ExitError); ok {
				switch exitErr.ExitCode() {
				case 5:
					fmt.Fprintf(os.Stderr, "\n⚠️  Authentication failed. The stored password may be incorrect.\n")
					fmt.Fprintf(os.Stderr, "   Update the password with: veessh edit %s --ask-password\n", profileName)
				case 6:
					fmt.Fprintf(os.Stderr, "\n⚠️  Host key verification failed.\n")
				}
			}
		}
		return err
	}

	// Fallback: Use SSH_ASKPASS with a helper script
	// This works by creating a temporary script that outputs the password
	tmpScript, err := createSSHAskPassScript(password)
	if err != nil {
		// If we can't create the script, fall back to interactive prompt
		fmt.Fprintf(os.Stderr, "Warning: Could not create SSH_ASKPASS script. Password will be prompted.\n")
		cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
		return util.RunAttached(cmd)
	}
	defer os.Remove(tmpScript)

	// SSH_ASKPASS requires:
	// 1. DISPLAY environment variable (even on non-X11 systems)
	// 2. SSH_ASKPASS pointing to our script
	// 3. stdin must NOT be a TTY (SSH won't use SSH_ASKPASS if stdin is a TTY)
	// 4. We need to detach from the controlling terminal
	
	// Try using 'setsid' or 'nohup' to detach from TTY
	var cmd *exec.Cmd
	if setsidPath := findExecutable("setsid"); setsidPath != "" {
		// Use setsid to create a new session (detaches from TTY)
		cmd = exec.CommandContext(ctx, setsidPath, "ssh")
		cmd.Args = append(cmd.Args, sshArgs...)
	} else {
		// Fallback: try with nohup or just ssh
		// Note: This may not work if we're attached to a TTY
		cmd = exec.CommandContext(ctx, "ssh", sshArgs...)
	}

	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+tmpScript,
		"DISPLAY=:0", // Required even on non-X11 systems
		"SSH_ASKPASS_REQUIRE=force", // Force SSH to use SSH_ASKPASS
	)
	
	// Don't attach stdin - SSH_ASKPASS only works when stdin is not a TTY
	// We'll attach stdout/stderr for interactive session
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If SSH_ASKPASS failed, fall back to interactive prompt
		// This can happen if we're attached to a TTY
		fmt.Fprintf(os.Stderr, "Note: Automatic password injection failed. You will be prompted for the password.\n")
		fmt.Fprintf(os.Stderr, "Tip: Install 'sshpass' for more reliable automatic password injection:\n")
		fmt.Fprintf(os.Stderr, "  brew install hudochenkov/sshpass/sshpass\n\n")
		
		cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
		return util.RunAttached(cmd)
	}

	return nil
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
	if err := tmpFile.Close(); err != nil {
		os.Remove(scriptPath)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Write the script with proper escaping for shell special characters
	// Use printf instead of echo for better control over special characters
	escapedPassword := strings.ReplaceAll(password, "\\", "\\\\")
	escapedPassword = strings.ReplaceAll(escapedPassword, "\"", "\\\"")
	escapedPassword = strings.ReplaceAll(escapedPassword, "$", "\\$")
	escapedPassword = strings.ReplaceAll(escapedPassword, "`", "\\`")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"%s\"\n", escapedPassword)
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
