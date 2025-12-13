package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var listTagFilters []string

var cmdList = &cobra.Command{
	Use:   "list",
	Short: "List connection profiles",
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
		if len(listTagFilters) > 0 {
			want := map[string]struct{}{}
			for _, t := range listTagFilters {
				want[strings.ToLower(t)] = struct{}{}
			}
			filtered := make([]config.Profile, 0, len(profiles))
			for _, p := range profiles {
				hasAll := true
				lowTags := map[string]struct{}{}
				for _, tg := range p.Tags {
					lowTags[strings.ToLower(tg)] = struct{}{}
				}
				for t := range want {
					if _, ok := lowTags[t]; !ok {
						hasAll = false
						break
					}
				}
				if hasAll {
					filtered = append(filtered, p)
				}
			}
			profiles = filtered
		}
		if OutputJSON() {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(profiles)
		}
		sort.SliceStable(profiles, func(i, j int) bool {
			if profiles[i].Favorite != profiles[j].Favorite {
				return profiles[i].Favorite && !profiles[j].Favorite
			}
			if profiles[i].Group == profiles[j].Group {
				return profiles[i].Name < profiles[j].Name
			}
			return profiles[i].Group < profiles[j].Group
		})
		for _, p := range profiles {
			group := p.Group
			if group == "" {
				group = "default"
			}
			userHost := p.Host
			if p.Username != "" {
				userHost = p.Username + "@" + userHost
			}
			fav := " "
			if p.Favorite {
				fav = "*"
			}
			tags := ""
			if len(p.Tags) > 0 {
				tags = " [" + strings.Join(p.Tags, ",") + "]"
			}
			fmt.Printf("%s %s/%s\t(%s)\t%s:%s%s\n", fav, group, p.Name, p.Protocol, userHost, portString(p.Port), tags)
		}
		return nil
	},
}

func init() {
	cmdList.Flags().StringSliceVar(&listTagFilters, "tag", nil, "filter by tag(s), require all")
}
