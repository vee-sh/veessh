package connectors

import (
	"testing"

	"github.com/vee-sh/veessh/internal/config"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"/path/to/dir", "'/path/to/dir'"},
		{"/path/with spaces", "'/path/with spaces'"},
		{"/path/with\ttab", "'/path/with\ttab'"},
		{"path'with'quotes", "'path'\"'\"'with'\"'\"'quotes'"},
		{"$(command)", "'$(command)'"},
		{"`backticks`", "'`backticks`'"},
		{"semi;colon", "'semi;colon'"},
		{"pipe|char", "'pipe|char'"},
		{"dollar$var", "'dollar$var'"},
		{"", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellQuote(tt.input)
			if got != tt.want {
				t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildRemoteCommand(t *testing.T) {
	tests := []struct {
		name    string
		profile config.Profile
		want    string
	}{
		{
			name:    "empty",
			profile: config.Profile{},
			want:    "",
		},
		{
			name:    "remote dir only",
			profile: config.Profile{RemoteDir: "/app"},
			want:    "cd '/app' && exec $SHELL -l",
		},
		{
			name:    "remote dir with spaces",
			profile: config.Profile{RemoteDir: "/path/with spaces"},
			want:    "cd '/path/with spaces' && exec $SHELL -l",
		},
		{
			name:    "remote command only",
			profile: config.Profile{RemoteCommand: "tmux attach"},
			want:    "tmux attach",
		},
		{
			name:    "dir and command",
			profile: config.Profile{RemoteDir: "/app", RemoteCommand: "make run"},
			want:    "cd '/app' && make run",
		},
		{
			name:    "dir with single quotes",
			profile: config.Profile{RemoteDir: "/path/it's here"},
			want:    "cd '/path/it'\"'\"'s here' && exec $SHELL -l",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRemoteCommand(tt.profile)
			if got != tt.want {
				t.Errorf("buildRemoteCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

