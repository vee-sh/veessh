package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
	"github.com/spf13/cobra"

	"github.com/vee-sh/veessh/internal/config"
)

var importSSHFile string
var importSSHGroup string
var importSSHPrefix string
var importSSHOverwrite bool
var importSSHDryRun bool

var cmdImportSSH = &cobra.Command{
	Use:   "import-ssh",
	Short: "Import profiles from an OpenSSH config file (~/.ssh/config)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if importSSHFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			importSSHFile = filepath.Join(home, ".ssh", "config")
		}
		f, err := os.Open(importSSHFile)
		if err != nil {
			return err
		}
		defer f.Close()
		cfgSSH, err := ssh_config.Decode(f)
		if err != nil {
			return err
		}

		cfgPath, err := config.DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		if cfg.Profiles == nil {
			cfg.Profiles = map[string]config.Profile{}
		}

		imported := 0
		skipped := 0
		for _, pat := range cfgSSH.Hosts {
			for _, pattern := range pat.Patterns {
				host := pattern.String()
				if strings.ContainsAny(host, "*? ") { // skip wildcards for now
					continue
				}
				name := importSSHPrefix + host
				if _, exists := cfg.Profiles[name]; exists && !importSSHOverwrite {
					skipped++
					continue
				}
				get := func(key string) string { v, _ := cfgSSH.Get(host, key); return v }
				p := config.Profile{
					Name:         name,
					Protocol:     config.ProtocolSSH,
					Host:         valueOr(get("Hostname"), host),
					Username:     get("User"),
					Group:        importSSHGroup,
					IdentityFile: expandTilde(get("IdentityFile")),
					Description:  "imported from ssh config",
				}
				if port := get("Port"); port != "" {
					if v, err := strconv.Atoi(port); err == nil {
						p.Port = v
					}
				}
				if pj := get("ProxyJump"); pj != "" {
					p.ProxyJump = pj
				}
				if !importSSHDryRun {
					cfg.UpsertProfile(p)
					imported++
				}
			}
		}
		if !importSSHDryRun {
			if err := config.Save(cfgPath, cfg); err != nil {
				return err
			}
		}
		fmt.Printf("imported %d, skipped %d\n", imported, skipped)
		return nil
	},
}

func init() {
	cmdImportSSH.Flags().StringVar(&importSSHFile, "file", "", "ssh config file (default: ~/.ssh/config)")
	cmdImportSSH.Flags().StringVar(&importSSHGroup, "group", "", "group to assign to imported profiles")
	cmdImportSSH.Flags().StringVar(&importSSHPrefix, "prefix", "", "name prefix for imported profiles")
	cmdImportSSH.Flags().BoolVar(&importSSHOverwrite, "overwrite", false, "overwrite existing profiles")
	cmdImportSSH.Flags().BoolVar(&importSSHDryRun, "dry-run", false, "parse and show stats without writing")
}

func valueOr(v string, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func expandTilde(p string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

