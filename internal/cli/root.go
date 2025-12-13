package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/audit"
	"github.com/alex-vee-sh/veessh/internal/config"
	"github.com/alex-vee-sh/veessh/internal/connectors"
	"github.com/alex-vee-sh/veessh/internal/credentials"
	"github.com/alex-vee-sh/veessh/internal/ui"
	"github.com/alex-vee-sh/veessh/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "veessh",
	Short: "Console connection manager for SSH/SFTP/Telnet and more",
	Long: `veessh - Console connection manager for SSH/SFTP/Telnet and more.

Run without arguments to launch interactive profile picker.`,
	RunE: runInteractive,
}

// Execute is the entrypoint for the Cobra command tree.
func Execute() error {
	addSubcommands()
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.Version = version.String()
	rootCmd.SetVersionTemplate(versionTemplate)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return rootCmd.ExecuteContext(ctx)
}

func addSubcommands() {
	rootCmd.AddCommand(cmdAdd)
	rootCmd.AddCommand(cmdEdit)
	rootCmd.AddCommand(cmdClone)
	rootCmd.AddCommand(cmdList)
	rootCmd.AddCommand(cmdShow)
	rootCmd.AddCommand(cmdConnect)
	rootCmd.AddCommand(cmdRun)
	rootCmd.AddCommand(cmdTest)
	rootCmd.AddCommand(cmdScp)
	rootCmd.AddCommand(cmdRsync)
	rootCmd.AddCommand(cmdCopyId)
	rootCmd.AddCommand(cmdSession)
	rootCmd.AddCommand(cmdRemove)
	rootCmd.AddCommand(cmdPick)
	rootCmd.AddCommand(cmdFavorite)
	rootCmd.AddCommand(cmdHistory)
	rootCmd.AddCommand(cmdAudit)
	rootCmd.AddCommand(cmdHostkey)
	rootCmd.AddCommand(cmdDoctor)
	rootCmd.AddCommand(cmdExport)
	rootCmd.AddCommand(cmdImport)
	rootCmd.AddCommand(cmdImportSSH)
	rootCmd.AddCommand(cmdCompletion)
	rootCmd.AddCommand(cmdVersion)
}

var flagJSON bool
var flagVersionShort bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output JSON where supported")
	rootCmd.PersistentFlags().BoolVarP(&flagVersionShort, "version", "v", false, "show version and exit")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flagVersionShort {
			fmt.Fprint(cmd.OutOrStdout(), renderVersion())
			os.Exit(0)
		}
		return nil
	}
}

func OutputJSON() bool { return flagJSON }

const versionTemplate = `{{.Name}} {{.Version}}

 /\_/\   veessh
( o.o )  {{.Version}}
 > ^ <

`

func renderVersion() string {
	return fmt.Sprintf(`%s %s

 /\\_/\\   veessh
( o.o )  %s
 > ^ <

`, rootCmd.Name(), rootCmd.Version, rootCmd.Version)
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// runInteractive launches the interactive picker when veessh is run with no arguments
func runInteractive(cmd *cobra.Command, args []string) error {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Add one with:")
		fmt.Println("  veessh add <name> --host <host> --user <user>")
		return nil
	}

	// Launch interactive picker (prefer fzf if available)
	p, err := ui.PickProfileInteractive(cmd.Context(), cfg, "", "", false, true, true, nil)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}

	conn, err := connectors.Get(p.Protocol)
	if err != nil {
		return err
	}

	password, _ := credentials.GetPassword(p.Name)

	// Audit log: connection start
	startTime := time.Now()
	audit.LogConnect(p.Name, string(p.Protocol), p.Host, p.Username)

	if err := conn.Exec(cmd.Context(), p, password); err != nil {
		audit.LogDisconnect(p.Name, string(p.Protocol), p.Host, p.Username, startTime, 1, err)
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}

	// Audit log: successful disconnect
	audit.LogDisconnect(p.Name, string(p.Protocol), p.Host, p.Username, startTime, 0, nil)

	// Update usage tracking
	p.LastUsed = time.Now()
	p.UseCount++
	cfg.UpsertProfile(p)
	if err := config.Save(cfgPath, cfg); err != nil {
		fmt.Printf("warning: failed to update usage stats: %v\n", err)
	}

	return nil
}
