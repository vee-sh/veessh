package ui

import (
	"testing"

	"github.com/vee-sh/veessh/internal/config"
)

func TestEffectivePort(t *testing.T) {
	tests := []struct {
		name     string
		profile  config.Profile
		wantPort int
	}{
		{
			name:     "explicit port",
			profile:  config.Profile{Port: 2222, Protocol: config.ProtocolSSH},
			wantPort: 2222,
		},
		{
			name:     "SSH default",
			profile:  config.Profile{Port: 0, Protocol: config.ProtocolSSH},
			wantPort: 22,
		},
		{
			name:     "SFTP default",
			profile:  config.Profile{Port: 0, Protocol: config.ProtocolSFTP},
			wantPort: 22,
		},
		{
			name:     "Telnet default",
			profile:  config.Profile{Port: 0, Protocol: config.ProtocolTelnet},
			wantPort: 23,
		},
		{
			name:     "unknown protocol",
			profile:  config.Profile{Port: 0, Protocol: "unknown"},
			wantPort: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectivePort(tt.profile)
			if got != tt.wantPort {
				t.Errorf("effectivePort() = %d, want %d", got, tt.wantPort)
			}
		})
	}
}

