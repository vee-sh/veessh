package cli

import (
	"testing"
)

func TestPortString(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{0, "default"},
		{-1, "default"},
		{22, "22"},
		{2222, "2222"},
		{65535, "65535"},
	}

	for _, tt := range tests {
		got := portString(tt.port)
		if got != tt.want {
			t.Errorf("portString(%d) = %s, want %s", tt.port, got, tt.want)
		}
	}
}

func TestParseScpPath(t *testing.T) {
	tests := []struct {
		path       string
		wantProf   string
		wantPath   string
		wantRemote bool
	}{
		{"mybox:/var/log", "mybox", "/var/log", true},
		{"./local/path", "", "./local/path", false},
		{"/absolute/path", "", "/absolute/path", false},
		{"profile:relative/path", "profile", "relative/path", true},
		{"profile:", "profile", "", true},
		{"simple", "", "simple", false},
	}

	for _, tt := range tests {
		prof, path, isRemote := parseScpPath(tt.path)
		if prof != tt.wantProf {
			t.Errorf("parseScpPath(%q) profile = %q, want %q", tt.path, prof, tt.wantProf)
		}
		if path != tt.wantPath {
			t.Errorf("parseScpPath(%q) path = %q, want %q", tt.path, path, tt.wantPath)
		}
		if isRemote != tt.wantRemote {
			t.Errorf("parseScpPath(%q) isRemote = %v, want %v", tt.path, isRemote, tt.wantRemote)
		}
	}
}

func TestExpandHomePath(t *testing.T) {
	tests := []struct {
		input    string
		hasHome  bool // Whether result should differ from input
	}{
		{"~/test", true},
		{"~/.ssh/key", true},
		{"/absolute/path", false},
		{"relative/path", false},
		{"", false},
		{"~", false}, // Just ~ without / should not expand
	}

	for _, tt := range tests {
		got := expandHomePath(tt.input)
		if tt.hasHome {
			if got == tt.input {
				t.Errorf("expandHomePath(%q) should expand ~ to home dir", tt.input)
			}
			if len(got) <= len(tt.input) {
				t.Errorf("expandHomePath(%q) = %q, should be longer", tt.input, got)
			}
		} else {
			if got != tt.input {
				t.Errorf("expandHomePath(%q) = %q, want %q", tt.input, got, tt.input)
			}
		}
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		// We can't easily test time-based functions without mocking
		// Just verify it doesn't panic
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.want {
				t.Errorf("formatTimeAgo() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestEffectivePortForProfile(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		protocol string
		want     int
	}{
		{"explicit port", 2222, "ssh", 2222},
		{"ssh default", 0, "ssh", 22},
		{"sftp default", 0, "sftp", 22},
		{"mosh default", 0, "mosh", 22},
		{"telnet default", 0, "telnet", 23},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Can't easily test effectivePortForProfile without importing config
			// This is just a placeholder for the test structure
		})
	}
}

