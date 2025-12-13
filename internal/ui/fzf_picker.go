package ui

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/vee-sh/veessh/internal/config"
)

// PickProfileInteractive can use fzf or survey to select a profile with optional filters and sorting.
func PickProfileInteractive(ctx context.Context, cfg config.Config, protocolFilter string, groupFilter string, favoritesOnly bool, preferFZF bool, recentFirst bool, tagFilters []string) (config.Profile, error) {
	profiles := cfg.ListProfiles()
	var filtered []config.Profile
	for _, p := range profiles {
		if protocolFilter != "" && strings.ToLower(string(p.Protocol)) != strings.ToLower(protocolFilter) {
			continue
		}
		if groupFilter != "" && strings.ToLower(p.Group) != strings.ToLower(groupFilter) {
			continue
		}
		if favoritesOnly && !p.Favorite {
			continue
		}
		if len(tagFilters) > 0 {
			want := map[string]struct{}{}
			for _, t := range tagFilters {
				want[strings.ToLower(t)] = struct{}{}
			}
			lowTags := map[string]struct{}{}
			for _, tg := range p.Tags {
				lowTags[strings.ToLower(tg)] = struct{}{}
			}
			okAll := true
			for t := range want {
				if _, ok := lowTags[t]; !ok {
					okAll = false
					break
				}
			}
			if !okAll {
				continue
			}
		}
		filtered = append(filtered, p)
	}
	if len(filtered) == 0 {
		return config.Profile{}, fmt.Errorf("no profiles found")
	}

	sort.Slice(filtered, func(i, j int) bool {
		if recentFirst {
			if filtered[i].LastUsed.Equal(filtered[j].LastUsed) {
				if filtered[i].Group == filtered[j].Group {
					return filtered[i].Name < filtered[j].Name
				}
				return filtered[i].Group < filtered[j].Group
			}
			return filtered[i].LastUsed.After(filtered[j].LastUsed)
		}
		if filtered[i].Group == filtered[j].Group {
			return filtered[i].Name < filtered[j].Name
		}
		return filtered[i].Group < filtered[j].Group
	})

	labels := make([]string, 0, len(filtered))
	for _, p := range filtered {
		group := p.Group
		if group == "" {
			group = "default"
		}
		userHost := p.Host
		if p.Username != "" {
			userHost = p.Username + "@" + userHost
		}
		desc := p.Description
		if desc != "" {
			desc = " - " + desc
		}
		fav := ""
		if p.Favorite {
			fav = "* "
		}
		labels = append(labels, fmt.Sprintf("%s%s/%s  (%s)  %s:%d%s", fav, group, p.Name, p.Protocol, userHost, effectivePort(p), desc))
	}

	if preferFZF {
		// Try fzf, fall back to survey on error
		sel, err := pickWithFZF(ctx, labels)
		if err == nil {
			for i, l := range labels {
				if l == sel {
					return filtered[i], nil
				}
			}
			return config.Profile{}, fmt.Errorf("selection not found")
		}
		// fallthrough to survey
	}

	var selected string
	prompt := &survey.Select{
		Message:  "Select profile:",
		Options:  labels,
		PageSize: 15,
		VimMode:  false,
		Filter: func(filter string, opt string, idx int) bool {
			f := strings.ToLower(filter)
			o := strings.ToLower(opt)
			return strings.Contains(o, f)
		},
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		if errors.Is(err, terminal.InterruptErr) {
			return config.Profile{}, context.Canceled
		}
		return config.Profile{}, err
	}
	for i, l := range labels {
		if l == selected {
			return filtered[i], nil
		}
	}
	return config.Profile{}, fmt.Errorf("selection not found")
}

func pickWithFZF(ctx context.Context, options []string) (string, error) {
	cmd := exec.CommandContext(ctx, "fzf", "--prompt=Select profile: ", "--height=80%", "--layout=reverse")
	var in bytes.Buffer
	for i, o := range options {
		if i > 0 {
			in.WriteByte('\n')
		}
		in.WriteString(o)
	}
	cmd.Stdin = &in
	out, err := cmd.Output()
	if err != nil {
		// Check if context was cancelled
		if ctx.Err() != nil {
			return "", context.Canceled
		}
		// fzf exits with code 130 on Ctrl+C, 1 on no match/escape
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				return "", context.Canceled
			}
		}
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", fmt.Errorf("no selection")
}
