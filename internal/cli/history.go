package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/alex-vee-sh/veessh/internal/config"
)

var (
	historyLimit int
	historyStats bool
)

var cmdHistory = &cobra.Command{
	Use:   "history",
	Short: "Show connection history and statistics",
	Long: `Display recent connections and usage statistics.

Examples:
  veessh history            # Show recent connections
  veessh history -n 5       # Show last 5 connections
  veessh history --stats    # Show usage statistics
  veessh history --json     # JSON output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		profiles := cfg.ListProfiles()

		if historyStats {
			return showStats(cmd, profiles)
		}

		return showHistory(cmd, profiles)
	},
}

func showHistory(cmd *cobra.Command, profiles []config.Profile) error {
	// Filter profiles that have been used
	var used []config.Profile
	for _, p := range profiles {
		if p.UseCount > 0 {
			used = append(used, p)
		}
	}

	if len(used) == 0 {
		fmt.Println("No connection history.")
		return nil
	}

	// Sort by last used (most recent first)
	sort.Slice(used, func(i, j int) bool {
		return used[i].LastUsed.After(used[j].LastUsed)
	})

	// Apply limit
	if historyLimit > 0 && historyLimit < len(used) {
		used = used[:historyLimit]
	}

	if OutputJSON() {
		type historyEntry struct {
			Name     string    `json:"name"`
			Host     string    `json:"host"`
			Protocol string    `json:"protocol"`
			LastUsed time.Time `json:"lastUsed"`
			UseCount int       `json:"useCount"`
		}
		entries := make([]historyEntry, len(used))
		for i, p := range used {
			entries[i] = historyEntry{
				Name:     p.Name,
				Host:     p.Host,
				Protocol: string(p.Protocol),
				LastUsed: p.LastUsed,
				UseCount: p.UseCount,
			}
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	fmt.Println("Recent connections:")
	fmt.Println()
	for _, p := range used {
		ago := formatTimeAgo(p.LastUsed)
		fmt.Printf("  %-20s  %s@%s  (used %dx, last: %s)\n",
			p.Name,
			p.Username,
			p.Host,
			p.UseCount,
			ago,
		)
	}

	return nil
}

func showStats(cmd *cobra.Command, profiles []config.Profile) error {
	if len(profiles) == 0 {
		fmt.Println("No profiles configured.")
		return nil
	}

	totalProfiles := len(profiles)
	totalConnections := 0
	favorites := 0
	usedProfiles := 0

	protocolCounts := map[config.Protocol]int{}
	groupCounts := map[string]int{}

	var mostUsed config.Profile
	var mostRecent config.Profile

	for _, p := range profiles {
		totalConnections += p.UseCount
		protocolCounts[p.Protocol]++

		group := p.Group
		if group == "" {
			group = "(default)"
		}
		groupCounts[group]++

		if p.Favorite {
			favorites++
		}
		if p.UseCount > 0 {
			usedProfiles++
		}
		if p.UseCount > mostUsed.UseCount {
			mostUsed = p
		}
		if p.LastUsed.After(mostRecent.LastUsed) {
			mostRecent = p
		}
	}

	if OutputJSON() {
		stats := map[string]interface{}{
			"totalProfiles":    totalProfiles,
			"usedProfiles":     usedProfiles,
			"totalConnections": totalConnections,
			"favorites":        favorites,
			"protocolCounts":   protocolCounts,
			"groupCounts":      groupCounts,
		}
		if mostUsed.Name != "" {
			stats["mostUsed"] = map[string]interface{}{
				"name":     mostUsed.Name,
				"useCount": mostUsed.UseCount,
			}
		}
		if !mostRecent.LastUsed.IsZero() {
			stats["mostRecent"] = map[string]interface{}{
				"name":     mostRecent.Name,
				"lastUsed": mostRecent.LastUsed,
			}
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(stats)
	}

	fmt.Println("=== Connection Statistics ===")
	fmt.Println()
	fmt.Printf("  Total profiles:     %d\n", totalProfiles)
	fmt.Printf("  Used profiles:      %d\n", usedProfiles)
	fmt.Printf("  Total connections:  %d\n", totalConnections)
	fmt.Printf("  Favorites:          %d\n", favorites)
	fmt.Println()

	fmt.Println("By protocol:")
	for proto, count := range protocolCounts {
		fmt.Printf("  %-10s %d\n", proto, count)
	}
	fmt.Println()

	fmt.Println("By group:")
	for group, count := range groupCounts {
		fmt.Printf("  %-15s %d\n", group, count)
	}
	fmt.Println()

	if mostUsed.Name != "" && mostUsed.UseCount > 0 {
		fmt.Printf("Most used:    %s (%d connections)\n", mostUsed.Name, mostUsed.UseCount)
	}
	if !mostRecent.LastUsed.IsZero() {
		fmt.Printf("Most recent:  %s (%s)\n", mostRecent.Name, formatTimeAgo(mostRecent.LastUsed))
	}

	return nil
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

func init() {
	cmdHistory.Flags().IntVarP(&historyLimit, "limit", "n", 10, "number of entries to show")
	cmdHistory.Flags().BoolVar(&historyStats, "stats", false, "show usage statistics")
}

