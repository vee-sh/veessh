package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Protocol string

const (
	ProtocolSSH    Protocol = "ssh"
	ProtocolSFTP   Protocol = "sftp"
	ProtocolTelnet Protocol = "telnet"
	ProtocolMosh   Protocol = "mosh"
	ProtocolSSM    Protocol = "ssm"
)

type Profile struct {
	Name            string    `yaml:"name"`
	Protocol        Protocol  `yaml:"protocol"`
	Host            string    `yaml:"host"`
	Port            int       `yaml:"port"`
	Username        string    `yaml:"username"`
	IdentityFile    string    `yaml:"identityFile"`
	UseAgent        bool      `yaml:"useAgent"`
	ExtraArgs       []string  `yaml:"extraArgs"`
	Group           string    `yaml:"group"`
	Description     string    `yaml:"description"`
	Favorite        bool      `yaml:"favorite"`
	LastUsed        time.Time `yaml:"lastUsed"`
	UseCount        int       `yaml:"useCount"`
	ProxyJump       string    `yaml:"proxyJump"`
	Tags            []string  `yaml:"tags"`
	LocalForwards   []string  `yaml:"localForwards"`
	RemoteForwards  []string  `yaml:"remoteForwards"`
	DynamicForwards []string  `yaml:"dynamicForwards"`

	// On-connect automation
	RemoteCommand string   `yaml:"remoteCommand,omitempty"` // Command to run on connect (e.g., "tmux attach || tmux new")
	RemoteDir     string   `yaml:"remoteDir,omitempty"`     // cd to this directory on connect
	SetEnv        []string `yaml:"setEnv,omitempty"`        // Environment variables to set (KEY=VALUE)

	// AWS SSM specific
	AWSRegion    string `yaml:"awsRegion,omitempty"`
	AWSProfile   string `yaml:"awsProfile,omitempty"`
	InstanceID   string `yaml:"instanceId,omitempty"` // EC2 instance ID for SSM

	// Mosh specific
	MoshServer string `yaml:"moshServer,omitempty"` // Path to mosh-server on remote
}

type Config struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

func DefaultPath() (string, error) {
	cfgHome := os.Getenv("XDG_CONFIG_HOME")
	if strings.TrimSpace(cfgHome) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cfgHome = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgHome, "veessh", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{Profiles: map[string]Profile{}}, nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (c *Config) UpsertProfile(p Profile) {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[p.Name] = p
}

func (c *Config) DeleteProfile(name string) bool {
	if c.Profiles == nil {
		return false
	}
	if _, ok := c.Profiles[name]; ok {
		delete(c.Profiles, name)
		return true
	}
	return false
}

func (c *Config) GetProfile(name string) (Profile, bool) {
	if c.Profiles == nil {
		return Profile{}, false
	}
	p, ok := c.Profiles[name]
	return p, ok
}

func (c *Config) ListProfiles() []Profile {
	list := make([]Profile, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Group == list[j].Group {
			return list[i].Name < list[j].Name
		}
		return list[i].Group < list[j].Group
	})
	return list
}

func (p *Profile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("profile name is required")
	}
	switch p.Protocol {
	case ProtocolSSH, ProtocolSFTP, ProtocolTelnet, ProtocolMosh, ProtocolSSM:
		// ok
	default:
		return fmt.Errorf("unsupported protocol: %s", p.Protocol)
	}
	if strings.TrimSpace(p.Host) == "" {
		return errors.New("host is required")
	}
	if p.Port <= 0 {
		switch p.Protocol {
		case ProtocolSSH, ProtocolSFTP, ProtocolMosh:
			p.Port = 22
		case ProtocolTelnet:
			p.Port = 23
		case ProtocolSSM:
			// SSM doesn't use ports
		}
	}
	return nil
}
