package cli

import (
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var (
	testTimeout int
	testAll     bool
)

var cmdTest = &cobra.Command{
	Use:   "test [name]",
	Short: "Test connectivity to a profile's host",
	Long: `Test if a host is reachable by attempting a TCP connection.

Examples:
  veessh test mybox              # Test single profile
  veessh test mybox --timeout 5  # Custom timeout (seconds)
  veessh test --all              # Test all profiles`,
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

		timeout := time.Duration(testTimeout) * time.Second

		if testAll {
			return testAllProfiles(cfg, timeout)
		}

		if len(args) == 0 {
			return fmt.Errorf("profile name required (or use --all)")
		}

		name := args[0]
		p, ok := cfg.GetProfile(name)
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}

		return testProfile(p, timeout)
	},
}

func testProfile(p config.Profile, timeout time.Duration) error {
	port := effectivePortForProfile(p)
	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", port))

	fmt.Printf("Testing %s (%s)... ", p.Name, addr)

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("FAILED (%v)\n", err)
		return nil // Don't return error, just report
	}
	conn.Close()

	fmt.Printf("OK (%s)\n", elapsed.Round(time.Millisecond))
	return nil
}

func testAllProfiles(cfg config.Config, timeout time.Duration) error {
	profiles := cfg.ListProfiles()
	if len(profiles) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}

	passed := 0
	failed := 0

	for _, p := range profiles {
		port := effectivePortForProfile(p)
		addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", port))

		fmt.Printf("Testing %s (%s)... ", p.Name, addr)

		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, timeout)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("FAILED\n")
			failed++
		} else {
			conn.Close()
			fmt.Printf("OK (%s)\n", elapsed.Round(time.Millisecond))
			passed++
		}
	}

	fmt.Printf("\nResults: %d passed, %d failed, %d total\n", passed, failed, len(profiles))
	return nil
}

func effectivePortForProfile(p config.Profile) int {
	if p.Port > 0 {
		return p.Port
	}
	switch p.Protocol {
	case config.ProtocolSSH, config.ProtocolSFTP:
		return 22
	case config.ProtocolTelnet:
		return 23
	default:
		return 22
	}
}

func init() {
	cmdTest.Flags().IntVar(&testTimeout, "timeout", 10, "connection timeout in seconds")
	cmdTest.Flags().BoolVar(&testAll, "all", false, "test all profiles")
}

