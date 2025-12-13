package connectors

import (
	"testing"

	"github.com/vee-sh/veessh/internal/config"
)

func TestRegisterAndGet(t *testing.T) {
	// SSH, SFTP, Telnet, Mosh, SSM, GCloud should be registered via init()
	protocols := []config.Protocol{
		config.ProtocolSSH,
		config.ProtocolSFTP,
		config.ProtocolTelnet,
		config.ProtocolMosh,
		config.ProtocolSSM,
		config.ProtocolGCloud,
	}

	for _, proto := range protocols {
		t.Run(string(proto), func(t *testing.T) {
			conn, err := Get(proto)
			if err != nil {
				t.Errorf("Get(%s) error = %v", proto, err)
				return
			}
			if conn == nil {
				t.Errorf("Get(%s) returned nil connector", proto)
			}
			if conn.Name() == "" {
				t.Errorf("Connector for %s has empty name", proto)
			}
		})
	}
}

func TestGetUnregistered(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Error("Get() should error for unregistered protocol")
	}
}

func TestConnectorNames(t *testing.T) {
	expectedNames := map[config.Protocol]string{
		config.ProtocolSSH:    "ssh",
		config.ProtocolSFTP:   "sftp",
		config.ProtocolTelnet: "telnet",
		config.ProtocolMosh:   "mosh",
		config.ProtocolSSM:    "ssm",
		config.ProtocolGCloud: "gcloud",
	}

	for proto, wantName := range expectedNames {
		conn, err := Get(proto)
		if err != nil {
			t.Errorf("Get(%s) error = %v", proto, err)
			continue
		}
		if conn.Name() != wantName {
			t.Errorf("Connector for %s: Name() = %s, want %s", proto, conn.Name(), wantName)
		}
	}
}

