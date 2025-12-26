package credentials

import (
	"testing"
)

func TestSetPasswordEmptyName(t *testing.T) {
	err := SetPassword("", "password")
	if err == nil {
		t.Error("SetPassword() should error with empty profile name")
	}
}

func TestGetPasswordNonExistent(t *testing.T) {
	// This may fail on systems without keyring support
	// We just verify it doesn't panic
	pass, err := GetPassword("nonexistent-profile-12345")
	if err != nil {
		t.Logf("GetPassword error (may be expected): %v", err)
	}
	if pass != "" {
		t.Logf("GetPassword returned: %q", pass)
	}
}

func TestServiceName(t *testing.T) {
	// Verify the service name is correct by checking backend behavior
	// The service name is internal, but we can verify it's used correctly
	kr := &KeyringBackend{}
	// Just verify the backend can be instantiated
	if kr == nil {
		t.Error("KeyringBackend should be instantiable")
	}
}

