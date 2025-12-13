package cli

import (
	"testing"
)

func TestShellQuoteForRsync(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// No special chars - return as-is
		{"simple", "simple"},
		{"/path/to/key", "/path/to/key"},
		{"user@host", "user@host"},

		// With spaces - needs quoting
		{"/path/with spaces", "'/path/with spaces'"},
		{"/path/with\ttab", "'/path/with\ttab'"},

		// With quotes
		{"path'quote", "'path'\"'\"'quote'"},
		{"path\"double", "'path\"double'"},

		// With shell special chars
		{"$(cmd)", "'$(cmd)'"},
		{"`cmd`", "'`cmd`'"},
		{"$HOME", "'$HOME'"},
		{"path!excl", "'path!excl'"},
		{"back\\slash", "'back\\slash'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellQuoteForRsync(tt.input)
			if got != tt.want {
				t.Errorf("shellQuoteForRsync(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

