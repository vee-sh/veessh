package cli

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/hostkeys"
)

var cmdHostkey = &cobra.Command{
	Use:   "hostkey",
	Short: "Manage host key verification",
	Long: `Manage host key fingerprints for verification.

Subcommands:
  show    - Display a host's current key fingerprint
  pin     - Pin a host's current key for future verification
  verify  - Verify a host's key against pinned fingerprint
  list    - List all pinned keys`,
}

var cmdHostkeyShow = &cobra.Command{
	Use:   "show <profile|host[:port]>",
	Short: "Show a host's SSH key fingerprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		host, port, err := resolveHostPort(target)
		if err != nil {
			return err
		}

		fmt.Printf("Fetching host key from %s:%d...\n", host, port)
		keyType, fingerprint, err := hostkeys.GetHostFingerprint(host, port)
		if err != nil {
			return err
		}

		fmt.Printf("\nHost:        %s:%d\n", host, port)
		fmt.Printf("Key Type:    %s\n", keyType)
		fmt.Printf("Fingerprint: %s\n", fingerprint)

		// Check if in known_hosts
		inKnown, _ := hostkeys.IsHostInKnownHosts(host, port)
		if inKnown {
			fmt.Println("Status:      In known_hosts")
		} else {
			fmt.Println("Status:      Not in known_hosts")
		}

		return nil
	},
}

var cmdHostkeyPin = &cobra.Command{
	Use:   "pin <profile|host[:port]>",
	Short: "Pin a host's current key fingerprint",
	Long: `Pin a host's SSH key fingerprint for future verification.

When connecting to a host with a pinned key, veessh can verify
the key hasn't changed (protection against MITM attacks).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		host, port, err := resolveHostPort(target)
		if err != nil {
			return err
		}

		fmt.Printf("Fetching host key from %s:%d...\n", host, port)
		keyType, fingerprint, err := hostkeys.GetHostFingerprint(host, port)
		if err != nil {
			return err
		}

		fmt.Printf("Key Type:    %s\n", keyType)
		fmt.Printf("Fingerprint: %s\n", fingerprint)

		if err := hostkeys.PinKey(host, port, keyType, fingerprint, ""); err != nil {
			return fmt.Errorf("failed to pin key: %w", err)
		}

		fmt.Println("\nKey pinned successfully!")
		return nil
	},
}

var cmdHostkeyVerify = &cobra.Command{
	Use:   "verify <profile|host[:port]>",
	Short: "Verify a host's key against pinned fingerprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		host, port, err := resolveHostPort(target)
		if err != nil {
			return err
		}

		fmt.Printf("Verifying host key for %s:%d...\n", host, port)

		matched, pinnedFP, currentFP, err := hostkeys.VerifyPinnedKey(host, port)
		if err != nil {
			return err
		}

		if pinnedFP == "" {
			fmt.Println("No pinned key found for this host.")
			fmt.Println("Use 'veessh hostkey pin' to pin the current key.")
			return nil
		}

		fmt.Printf("Pinned:  %s\n", pinnedFP)
		fmt.Printf("Current: %s\n", currentFP)

		if matched {
			fmt.Println("\n[OK] Host key matches pinned fingerprint!")
		} else {
			fmt.Println("\n[WARNING] Host key does NOT match pinned fingerprint!")
			fmt.Println("This could indicate a man-in-the-middle attack or server key change.")
		}

		return nil
	},
}

var cmdHostkeyList = &cobra.Command{
	Use:   "list",
	Short: "List all pinned host keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := hostkeys.LoadPinnedKeys()
		if err != nil {
			return err
		}

		if len(keys) == 0 {
			fmt.Println("No pinned keys.")
			return nil
		}

		fmt.Println("Pinned host keys:")
		fmt.Println()
		for _, k := range keys {
			fmt.Printf("  %s:%d\n", k.Host, k.Port)
			fmt.Printf("    Type: %s\n", k.KeyType)
			fmt.Printf("    Fingerprint: %s\n", k.Fingerprint)
			if k.Comment != "" {
				fmt.Printf("    Comment: %s\n", k.Comment)
			}
			fmt.Println()
		}

		return nil
	},
}

func resolveHostPort(target string) (host string, port int, err error) {
	port = 22

	// Check if it's a profile name
	cfgPath, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(cfgPath)
		if err == nil {
			if p, ok := cfg.GetProfile(target); ok {
				if p.Port > 0 {
					port = p.Port
				}
				return p.Host, port, nil
			}
		}
	}

	// Parse as host:port
	if h, p, err := net.SplitHostPort(target); err == nil {
		fmt.Sscanf(p, "%d", &port)
		return h, port, nil
	}

	// Just a hostname
	return target, port, nil
}

func init() {
	cmdHostkey.AddCommand(cmdHostkeyShow)
	cmdHostkey.AddCommand(cmdHostkeyPin)
	cmdHostkey.AddCommand(cmdHostkeyVerify)
	cmdHostkey.AddCommand(cmdHostkeyList)
}

