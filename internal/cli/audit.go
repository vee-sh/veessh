package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/audit"
)

var auditLimit int

var cmdAudit = &cobra.Command{
	Use:   "audit",
	Short: "View connection audit log",
	Long: `Display the connection audit log showing connect/disconnect events.

The audit log records:
  - Connection start times
  - Disconnection times and duration
  - Profile, host, and user
  - Exit codes and errors

Examples:
  veessh audit             # Show last 50 entries
  veessh audit -n 10       # Show last 10 entries
  veessh audit --json      # JSON output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := audit.ReadEntries(auditLimit)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("No audit entries found.")
			return nil
		}

		if OutputJSON() {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		fmt.Println("Connection Audit Log:")
		fmt.Println()
		for _, e := range entries {
			ts := e.Timestamp.Format("2006-01-02 15:04:05")
			switch e.Action {
			case "connect":
				fmt.Printf("  %s  [CONNECT]     %s -> %s@%s\n", ts, e.Profile, e.User, e.Host)
			case "disconnect":
				fmt.Printf("  %s  [DISCONNECT]  %s (duration: %s, exit: %d)\n", ts, e.Profile, e.Duration, e.ExitCode)
			case "error":
				fmt.Printf("  %s  [ERROR]       %s: %s\n", ts, e.Profile, e.Error)
			}
		}

		return nil
	},
}

func init() {
	cmdAudit.Flags().IntVarP(&auditLimit, "limit", "n", 50, "number of entries to show")
}

