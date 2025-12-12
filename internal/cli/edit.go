package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/credentials"
)

var (
	editHost        string
	editPort        int
	editUser        string
	editIdentity    string
	editGroup       string
	editDesc        string
	editProxyJump   string
	editTags        []string
	editClearTags   bool
	editAskPassword bool
)

var cmdEdit = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing profile",
	Long: `Edit an existing profile. Only specified flags will be updated.
Examples:
  veessh edit mybox --host newhost.example.com
  veessh edit mybox --port 2222 --user admin
  veessh edit mybox --tags prod,web --group production`,
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
		p, ok := cfg.GetProfile(name)
		if !ok {
			return fmt.Errorf("profile %q not found", name)
		}

		// Update only fields that were explicitly set
		if cmd.Flags().Changed("host") {
			p.Host = editHost
		}
		if cmd.Flags().Changed("port") {
			p.Port = editPort
		}
		if cmd.Flags().Changed("user") {
			p.Username = editUser
		}
		if cmd.Flags().Changed("identity") {
			p.IdentityFile = editIdentity
		}
		if cmd.Flags().Changed("group") {
			p.Group = editGroup
		}
		if cmd.Flags().Changed("desc") {
			p.Description = editDesc
		}
		if cmd.Flags().Changed("proxy-jump") {
			p.ProxyJump = editProxyJump
		}
		if editClearTags {
			p.Tags = nil
		} else if cmd.Flags().Changed("tags") {
			p.Tags = editTags
		}

		if err := (&p).Validate(); err != nil {
			return err
		}

		cfg.UpsertProfile(p)
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}

		if editAskPassword {
			pass, err := promptPassword("Enter new password (leave empty to skip): ")
			if err != nil {
				return err
			}
			if pass != "" {
				if err := credentials.SetPassword(name, pass); err != nil {
					return err
				}
				fmt.Println("Password updated.")
			}
		}

		fmt.Printf("Updated profile %q\n", name)
		return nil
	},
}

func init() {
	cmdEdit.Flags().StringVar(&editHost, "host", "", "host or IP")
	cmdEdit.Flags().IntVar(&editPort, "port", 0, "port")
	cmdEdit.Flags().StringVar(&editUser, "user", "", "username")
	cmdEdit.Flags().StringVar(&editIdentity, "identity", "", "path to identity (private key) file")
	cmdEdit.Flags().StringVar(&editGroup, "group", "", "group name")
	cmdEdit.Flags().StringVar(&editDesc, "desc", "", "description")
	cmdEdit.Flags().StringVar(&editProxyJump, "proxy-jump", "", "proxy jump host")
	cmdEdit.Flags().StringSliceVar(&editTags, "tags", nil, "tags (replaces existing)")
	cmdEdit.Flags().BoolVar(&editClearTags, "clear-tags", false, "remove all tags")
	cmdEdit.Flags().BoolVar(&editAskPassword, "ask-password", false, "prompt to update password")
}

