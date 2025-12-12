package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/credentials"
)

var (
	addProtocol string
	addHost     string
	addPort     int
	addUser     string
	addIdentity string
	addUseAgent bool
	addExtra    []string
	addGroup    string
	addDesc     string
	addAskPass  bool
)

var cmdAdd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add or update a connection profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
	p := config.Profile{
		Name:         name,
		Protocol:     config.Protocol(strings.ToLower(addProtocol)),
		Host:         addHost,
		Port:         addPort,
		Username:     addUser,
		IdentityFile: addIdentity,
		UseAgent:     addUseAgent,
		ExtraArgs:    addExtra,
		Group:        addGroup,
		Description:  addDesc,
	}
	if err := (&p).Validate(); err != nil {
		return err
	}
		cfg.UpsertProfile(p)
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		if addAskPass {
			pass, err := promptPassword("Enter password (leave empty to skip): ")
			if err != nil {
				return err
			}
			if pass != "" {
				if err := credentials.SetPassword(name, pass); err != nil {
					return err
				}
			}
		}
		fmt.Printf("Saved profile %q (%s %s@%s:%s)\n", name, p.Protocol, p.Username, p.Host, portString(p.Port))
		return nil
	},
}

func init() {
	cmdAdd.Flags().StringVar(&addProtocol, "type", string(config.ProtocolSSH), "protocol: ssh|sftp|telnet")
	cmdAdd.Flags().StringVar(&addHost, "host", "", "host or IP")
	cmdAdd.Flags().IntVar(&addPort, "port", 0, "port")
	cmdAdd.Flags().StringVar(&addUser, "user", "", "username")
	cmdAdd.Flags().StringVar(&addIdentity, "identity", "", "path to identity (private key) file")
	cmdAdd.Flags().BoolVar(&addUseAgent, "agent", true, "use SSH agent if available")
	cmdAdd.Flags().StringSliceVar(&addExtra, "extra", nil, "extra args to pass to the client (repeatable)")
	cmdAdd.Flags().StringVar(&addGroup, "group", "", "group name for organizing profiles")
	cmdAdd.Flags().StringVar(&addDesc, "desc", "", "description")
	cmdAdd.Flags().BoolVar(&addAskPass, "ask-password", false, "prompt to store password in keychain")
}

func promptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fd := int(os.Stdin.Fd())
	bytePassword, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePassword)), nil
}

func portString(p int) string {
	if p <= 0 {
		return "default"
	}
	return strconv.Itoa(p)
}
