package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/vee-sh/veessh/internal/config"
	"github.com/vee-sh/veessh/internal/credentials"
)

var (
	addProtocol       string
	addHost           string
	addPort           int
	addUser           string
	addIdentity       string
	addUseAgent       bool
	addExtra          []string
	addGroup          string
	addDesc           string
	addAskPass        bool
	addRemoteCmd      string
	addRemoteDir      string
	addInstanceID     string
	addAWSRegion      string
	addAWSProfile     string
	addTags           []string
	addLocalForward   []string
	addRemoteForward  []string
	addDynamicForward []string
	addGCPProject     string
	addGCPZone        string
	addGCPTunnel      bool
	addExtends        string
)

var cmdAdd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add or update a connection profile",
	Long: `Add a new connection profile or update an existing one.

Supported protocols:
  ssh     - Standard SSH connection
  sftp    - SFTP file transfer
  telnet  - Telnet connection
  mosh    - Mobile shell (persistent SSH)
  ssm     - AWS Systems Manager Session Manager

Examples:
  # SSH profile
  veessh add mybox --host example.com --user alice --identity ~/.ssh/id_ed25519

  # SSH with on-connect automation
  veessh add dev --host dev.example.com --user dev --remote-cmd "tmux attach || tmux new"
  veessh add web --host web.example.com --user deploy --remote-dir /var/www/app

  # Mosh profile
  veessh add unstable --type mosh --host flaky.example.com --user alice

  # AWS SSM profile
  veessh add ec2-prod --type ssm --instance-id i-1234567890abcdef0 --aws-region us-east-1

  # With port forwarding
  veessh add tunnel --host jump.example.com --user admin \
    --local-forward 8080:internal:80 --dynamic-forward 1080

  # GCP Compute Engine
  veessh add gce-web --type gcloud --host my-vm --gcp-project myproject --gcp-zone us-central1-a

  # Profile inheritance (inherit from template)
  veessh add prod-template --host example.com --user deploy --identity ~/.ssh/deploy_key
  veessh add prod-web --extends prod-template --host web.example.com`,
	Args: cobra.ExactArgs(1),
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
			Name:            name,
			Protocol:        config.Protocol(strings.ToLower(addProtocol)),
			Host:            addHost,
			Port:            addPort,
			Username:        addUser,
			IdentityFile:    addIdentity,
			UseAgent:        addUseAgent,
			ExtraArgs:       addExtra,
			Group:           addGroup,
			Description:     addDesc,
			RemoteCommand:   addRemoteCmd,
			RemoteDir:       addRemoteDir,
			InstanceID:      addInstanceID,
			AWSRegion:       addAWSRegion,
			AWSProfile:      addAWSProfile,
			Tags:            addTags,
			LocalForwards:   addLocalForward,
			RemoteForwards:  addRemoteForward,
			DynamicForwards: addDynamicForward,
			GCPProject:      addGCPProject,
			GCPZone:         addGCPZone,
			GCPUseTunnel:    addGCPTunnel,
			Extends:         addExtends,
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
	cmdAdd.Flags().StringVar(&addProtocol, "type", string(config.ProtocolSSH), "protocol: ssh|sftp|telnet|mosh|ssm|gcloud")
	cmdAdd.Flags().StringVar(&addHost, "host", "", "host or IP")
	cmdAdd.Flags().IntVar(&addPort, "port", 0, "port")
	cmdAdd.Flags().StringVar(&addUser, "user", "", "username")
	cmdAdd.Flags().StringVar(&addIdentity, "identity", "", "path to identity (private key) file")
	cmdAdd.Flags().BoolVar(&addUseAgent, "agent", true, "use SSH agent if available")
	cmdAdd.Flags().StringSliceVar(&addExtra, "extra", nil, "extra args to pass to the client (repeatable)")
	cmdAdd.Flags().StringVar(&addGroup, "group", "", "group name for organizing profiles")
	cmdAdd.Flags().StringVar(&addDesc, "desc", "", "description")
	cmdAdd.Flags().BoolVar(&addAskPass, "ask-password", false, "prompt to store password in keychain")
	cmdAdd.Flags().StringSliceVar(&addTags, "tags", nil, "tags for filtering")

	// On-connect automation
	cmdAdd.Flags().StringVar(&addRemoteCmd, "remote-cmd", "", "command to run on connect (e.g., 'tmux attach || tmux new')")
	cmdAdd.Flags().StringVar(&addRemoteDir, "remote-dir", "", "cd to this directory on connect")

	// Port forwarding
	cmdAdd.Flags().StringSliceVar(&addLocalForward, "local-forward", nil, "local port forward (e.g., 8080:localhost:80)")
	cmdAdd.Flags().StringSliceVar(&addRemoteForward, "remote-forward", nil, "remote port forward")
	cmdAdd.Flags().StringSliceVar(&addDynamicForward, "dynamic-forward", nil, "dynamic SOCKS proxy (e.g., 1080)")

	// AWS SSM
	cmdAdd.Flags().StringVar(&addInstanceID, "instance-id", "", "EC2 instance ID (for SSM)")
	cmdAdd.Flags().StringVar(&addAWSRegion, "aws-region", "", "AWS region (for SSM)")
	cmdAdd.Flags().StringVar(&addAWSProfile, "aws-profile", "", "AWS profile name (for SSM)")

	// GCP gcloud
	cmdAdd.Flags().StringVar(&addGCPProject, "gcp-project", "", "GCP project (for gcloud)")
	cmdAdd.Flags().StringVar(&addGCPZone, "gcp-zone", "", "GCP zone (for gcloud)")
	cmdAdd.Flags().BoolVar(&addGCPTunnel, "gcp-tunnel", false, "use IAP tunnel (for gcloud)")

	// Profile inheritance
	cmdAdd.Flags().StringVar(&addExtends, "extends", "", "inherit from another profile")
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
