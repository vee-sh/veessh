package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"golang.org/x/term"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

// findExecutable finds an executable in PATH
func findExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// runOnboardingWizard guides first-time users through creating their first profile
func runOnboardingWizard(cfgPath string) error {
	fmt.Println("Welcome to veessh!")
	fmt.Println("Let's create your first connection profile.")
	fmt.Println()

	var answers struct {
		Name     string
		Protocol string
		Host     string
		Port     string
		Username string
		Identity string
		UseAgent bool
		Password string
		Group    string
		Desc     string
	}

	// Profile name
	if err := survey.AskOne(&survey.Input{
		Message: "Profile name:",
		Help:    "A short name to identify this connection (e.g., 'myserver', 'prod-web')",
	}, &answers.Name, survey.WithValidator(func(val interface{}) error {
		if str, ok := val.(string); !ok || strings.TrimSpace(str) == "" {
			return fmt.Errorf("profile name is required")
		}
		return nil
	})); err != nil {
		return err
	}

	// Protocol
	if err := survey.AskOne(&survey.Select{
		Message: "Protocol:",
		Options: []string{"ssh", "sftp", "telnet", "mosh", "ssm", "gcloud"},
		Default: "ssh",
		Help:    "Connection protocol to use",
	}, &answers.Protocol); err != nil {
		return err
	}

	// Host
	if err := survey.AskOne(&survey.Input{
		Message: "Host:",
		Help:    "Hostname or IP address of the remote server",
	}, &answers.Host, survey.WithValidator(func(val interface{}) error {
		if str, ok := val.(string); !ok || strings.TrimSpace(str) == "" {
			return fmt.Errorf("host is required")
		}
		return nil
	})); err != nil {
		return err
	}

	// Port (optional, with defaults)
	defaultPort := "22"
	if answers.Protocol == "telnet" {
		defaultPort = "23"
	} else if answers.Protocol == "ssm" {
		defaultPort = "" // SSM doesn't use ports
	}

	if defaultPort != "" {
		if err := survey.AskOne(&survey.Input{
			Message: "Port:",
			Default: defaultPort,
			Help:    fmt.Sprintf("Port number (default: %s)", defaultPort),
		}, &answers.Port); err != nil {
			return err
		}
	}

	// Username
	if err := survey.AskOne(&survey.Input{
		Message: "Username:",
		Help:    "Username for authentication",
	}, &answers.Username, survey.WithValidator(func(val interface{}) error {
		if str, ok := val.(string); !ok || strings.TrimSpace(str) == "" {
			return fmt.Errorf("username is required")
		}
		return nil
	})); err != nil {
		return err
	}

	// Identity file (optional)
	if err := survey.AskOne(&survey.Input{
		Message: "SSH key file (optional):",
		Help:    "Path to your private key file (e.g., ~/.ssh/id_ed25519). Leave empty to use SSH agent or password.",
	}, &answers.Identity); err != nil {
		return err
	}

	// Use SSH agent
	if answers.Identity == "" {
		if err := survey.AskOne(&survey.Confirm{
			Message: "Use SSH agent?",
			Default: true,
			Help:    "Use SSH agent for authentication if available",
		}, &answers.UseAgent); err != nil {
			return err
		}
	} else {
		answers.UseAgent = true // Default to true if key is specified
	}

	// Password (optional)
	askPassword := false
	if answers.Identity == "" && !answers.UseAgent {
		askPassword = true
	} else {
		helpMsg := "Store password securely in system keychain"
		// Check if 1Password is available
		if opPath := findExecutable("op"); opPath != "" {
			// Check if signed in
			cmd := exec.Command("op", "account", "list")
			if cmd.Run() == nil {
				helpMsg = "Store password securely in 1Password (detected) or system keychain"
			}
		}
		if err := survey.AskOne(&survey.Confirm{
			Message: "Store password in keychain?",
			Default: false,
			Help:    helpMsg,
		}, &askPassword); err != nil {
			return err
		}
	}

	if askPassword {
		fmt.Fprint(os.Stderr, "Password (hidden): ")
		fd := int(os.Stdin.Fd())
		bytePassword, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		answers.Password = strings.TrimSpace(string(bytePassword))
	}

	// Group (optional)
	if err := survey.AskOne(&survey.Input{
		Message: "Group (optional):",
		Help:    "Organize profiles into groups (e.g., 'production', 'development')",
	}, &answers.Group); err != nil {
		return err
	}

	// Description (optional)
	if err := survey.AskOne(&survey.Input{
		Message: "Description (optional):",
		Help:    "A brief description of this connection",
	}, &answers.Desc); err != nil {
		return err
	}

	// Build profile
	port := 0
	if answers.Port != "" {
		var err error
		port, err = strconv.Atoi(answers.Port)
		if err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
	}

	profile := config.Profile{
		Name:         answers.Name,
		Protocol:     config.Protocol(strings.ToLower(answers.Protocol)),
		Host:         strings.TrimSpace(answers.Host),
		Port:         port,
		Username:     strings.TrimSpace(answers.Username),
		IdentityFile: strings.TrimSpace(answers.Identity),
		UseAgent:     answers.UseAgent,
		Group:        strings.TrimSpace(answers.Group),
		Description:  strings.TrimSpace(answers.Desc),
	}

	if err := (&profile).Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	// Load existing config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	// Save profile
	cfg.UpsertProfile(profile)
	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	// Save password if provided
	if answers.Password != "" {
		if err := credentials.SetPassword(answers.Name, answers.Password); err != nil {
			fmt.Printf("Warning: failed to store password: %v\n", err)
		}
	}

	fmt.Printf("\n✓ Profile '%s' created successfully!\n", answers.Name)
	
	// Show helpful tips
	fmt.Println("\nYou can now:")
	fmt.Printf("  veessh connect %s\n", answers.Name)
	fmt.Println("  veessh                    # Interactive picker")
	fmt.Println("  veessh list                # List all profiles")
	fmt.Println("  veessh edit-config         # Edit config file directly")
	
	// Show password usage tip if password was stored
	if answers.Password != "" {
		fmt.Println("\n💡 Password tips:")
		if findExecutable("sshpass") == "" {
			fmt.Println("  - Install 'sshpass' for automatic password injection:")
			fmt.Println("    macOS:   brew install hudochenkov/sshpass/sshpass")
			fmt.Println("    Linux:   sudo apt-get install sshpass")
			fmt.Println("  - Without sshpass, you'll be prompted for the password")
		} else {
			fmt.Println("  - Password will be injected automatically (sshpass detected)")
		}
	}
	
	// Show 1Password tip if not using it
	if opPath := findExecutable("op"); opPath == "" {
		fmt.Println("\n💡 1Password integration:")
		fmt.Println("  - Install 1Password CLI: brew install --cask 1password-cli")
		fmt.Println("  - Sign in: op signin")
		fmt.Println("  - veessh will automatically use 1Password for password storage")
	}

	return nil
}

