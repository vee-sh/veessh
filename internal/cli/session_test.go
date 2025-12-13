package cli

import (
	"testing"

	"github.com/alex-vee-sh/veessh/internal/config"
)

func TestBuildSSHCommand(t *testing.T) {
	tests := []struct {
		name    string
		profile config.Profile
		want    string
	}{
		{
			name: "simple",
			profile: config.Profile{
				Host: "example.com",
			},
			want: "ssh example.com",
		},
		{
			name: "with user and port",
			profile: config.Profile{
				Host:     "example.com",
				Port:     2222,
				Username: "alice",
			},
			want: "ssh -p 2222 -l alice example.com",
		},
		{
			name: "with identity file",
			profile: config.Profile{
				Host:         "example.com",
				IdentityFile: "/path/to/key",
			},
			want: "ssh -i /path/to/key example.com",
		},
		{
			name: "identity file with spaces",
			profile: config.Profile{
				Host:         "example.com",
				IdentityFile: "/path/with spaces/key",
			},
			want: "ssh -i '/path/with spaces/key' example.com",
		},
		{
			name: "proxy jump with spaces",
			profile: config.Profile{
				Host:      "example.com",
				ProxyJump: "user@jump host",
			},
			want: "ssh -J 'user@jump host' example.com",
		},
		{
			name: "host with special chars",
			profile: config.Profile{
				Host: "example.com",
			},
			want: "ssh example.com",
		},
		{
			name: "all options with spaces",
			profile: config.Profile{
				Host:         "my host.com",
				Port:         22,
				Username:     "my user",
				IdentityFile: "/my path/key",
				ProxyJump:    "jump server",
			},
			want: "ssh -l 'my user' -i '/my path/key' -J 'jump server' 'my host.com'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSSHCommand(tt.profile)
			if got != tt.want {
				t.Errorf("buildSSHCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShellQuoteArg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// No quoting needed
		{"simple", "simple"},
		{"/path/to/file", "/path/to/file"},
		{"user@host", "user@host"},
		{"host.example.com", "host.example.com"},

		// Needs quoting
		{"/path/with spaces", "'/path/with spaces'"},
		{"it's here", "'it'\"'\"'s here'"},
		{"$(cmd)", "'$(cmd)'"},
		{"`cmd`", "'`cmd`'"},
		{"a;b", "'a;b'"},
		{"a|b", "'a|b'"},
		{"a&b", "'a&b'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellQuoteArg(tt.input)
			if got != tt.want {
				t.Errorf("shellQuoteArg(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

