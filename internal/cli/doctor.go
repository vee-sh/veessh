package cli

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var doctorVerbose bool

var cmdDoctor = &cobra.Command{
	Use:   "doctor [profile]",
	Short: "Diagnose connection issues",
	Long: `Run diagnostics on profiles to identify potential issues.

Checks performed:
  - Identity file exists and has correct permissions
  - Host resolves via DNS
  - Port is reachable (TCP connect)
  - SSH agent is running (if useAgent is enabled)
  - Required tools are installed (ssh, sftp, telnet)

Examples:
  veessh doctor           # Check all profiles
  veessh doctor mybox     # Check specific profile
  veessh doctor -v        # Verbose output`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		// Check global prerequisites first
		fmt.Println("=== System Checks ===")
		checkTools()
		checkSSHAgent()
		fmt.Println()

		if len(args) == 1 {
			name := args[0]
			p, ok := cfg.GetProfile(name)
			if !ok {
				return fmt.Errorf("profile %q not found", name)
			}
			fmt.Printf("=== Profile: %s ===\n", name)
			diagnoseProfile(p)
			return nil
		}

		// Check all profiles
		profiles := cfg.ListProfiles()
		if len(profiles) == 0 {
			fmt.Println("No profiles configured.")
			return nil
		}

		for _, p := range profiles {
			fmt.Printf("=== Profile: %s ===\n", p.Name)
			diagnoseProfile(p)
			fmt.Println()
		}

		return nil
	},
}

func checkTools() {
	tools := []struct {
		name     string
		required bool
	}{
		{"ssh", true},
		{"sftp", true},
		{"telnet", false},
		{"fzf", false},
	}

	for _, t := range tools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			if t.required {
				fmt.Printf("  [FAIL] %s: not found in PATH\n", t.name)
			} else {
				fmt.Printf("  [WARN] %s: not found (optional)\n", t.name)
			}
		} else {
			if doctorVerbose {
				fmt.Printf("  [OK]   %s: %s\n", t.name, path)
			} else {
				fmt.Printf("  [OK]   %s\n", t.name)
			}
		}
	}
}

func checkSSHAgent() {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		fmt.Println("  [WARN] SSH_AUTH_SOCK not set (agent may not be running)")
		return
	}

	// Check if socket exists
	if _, err := os.Stat(authSock); err != nil {
		fmt.Printf("  [WARN] SSH agent socket not accessible: %v\n", err)
		return
	}

	// Try to list keys
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			fmt.Println("  [WARN] SSH agent running but no keys loaded")
		} else {
			fmt.Printf("  [WARN] SSH agent check failed: %v\n", err)
		}
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	keyCount := 0
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "no identities") {
			keyCount++
		}
	}
	fmt.Printf("  [OK]   SSH agent: %d key(s) loaded\n", keyCount)
}

func diagnoseProfile(p config.Profile) {
	issues := 0

	// Check identity file
	if p.IdentityFile != "" {
		expandedPath := expandHomePath(p.IdentityFile)
		info, err := os.Stat(expandedPath)
		if err != nil {
			fmt.Printf("  [FAIL] Identity file: %s (%v)\n", p.IdentityFile, err)
			issues++
		} else {
			// Check permissions (should be 600 or 400)
			mode := info.Mode().Perm()
			if mode&0o077 != 0 {
				fmt.Printf("  [WARN] Identity file permissions too open: %s (%04o, should be 0600)\n", p.IdentityFile, mode)
			} else {
				if doctorVerbose {
					fmt.Printf("  [OK]   Identity file: %s\n", p.IdentityFile)
				} else {
					fmt.Println("  [OK]   Identity file exists with correct permissions")
				}
			}
		}
	}

	// Check DNS resolution
	port := effectivePortForProfile(p)
	addrs, err := net.LookupHost(p.Host)
	if err != nil {
		fmt.Printf("  [FAIL] DNS resolution: %s (%v)\n", p.Host, err)
		issues++
	} else {
		if doctorVerbose {
			fmt.Printf("  [OK]   DNS resolution: %s -> %s\n", p.Host, strings.Join(addrs, ", "))
		} else {
			fmt.Printf("  [OK]   DNS resolution: %s\n", p.Host)
		}

		// Check port connectivity
		addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", port))
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			fmt.Printf("  [FAIL] Port %d: not reachable (%v)\n", port, err)
			issues++
		} else {
			conn.Close()
			fmt.Printf("  [OK]   Port %d: reachable\n", port)
		}
	}

	// Check proxy jump if configured
	if p.ProxyJump != "" {
		fmt.Printf("  [INFO] ProxyJump configured: %s\n", p.ProxyJump)
	}

	// Summary
	if issues == 0 {
		fmt.Println("  [OK]   No issues detected")
	} else {
		fmt.Printf("  [!]    %d issue(s) found\n", issues)
	}
}

func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func init() {
	cmdDoctor.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "verbose output")
}

