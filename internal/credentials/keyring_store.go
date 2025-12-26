package credentials

import (
	"fmt"
	"os"

	"github.com/99designs/keyring"
)

const serviceName = "veessh"

// BackendType represents the credential storage backend
type BackendType string

const (
	BackendAuto      BackendType = "auto"      // Auto-detect (prefer 1Password if available)
	BackendKeyring   BackendType = "keyring"   // System keyring
	Backend1Password BackendType = "1password" // 1Password CLI
)

var (
	// currentBackend is the active backend (lazy initialized)
	currentBackend Backend
	backendType    = BackendAuto
)

// Backend interface for credential storage
type Backend interface {
	SetPassword(profileName string, password string) error
	GetPassword(profileName string) (string, error)
	DeletePassword(profileName string) error
}

// SetBackendType configures which backend to use
// Can be set via VEESSH_CREDENTIALS_BACKEND environment variable
func SetBackendType(bt BackendType) {
	backendType = bt
	currentBackend = nil // Reset to force re-initialization
}

// getBackend returns the active backend, initializing if needed
func getBackend() (Backend, error) {
	if currentBackend != nil {
		return currentBackend, nil
	}

	// Check environment variable
	if envBackend := os.Getenv("VEESSH_CREDENTIALS_BACKEND"); envBackend != "" {
		backendType = BackendType(envBackend)
	}

	// Initialize backend
	switch backendType {
	case Backend1Password:
		op := NewOnePasswordBackend("")
		if op.IsAvailable() {
			currentBackend = op
			return currentBackend, nil
		}
		return nil, fmt.Errorf("1Password CLI not available (not installed or not signed in)")

	case BackendKeyring:
		kr := &KeyringBackend{}
		currentBackend = kr
		return currentBackend, nil

	case BackendAuto:
		fallthrough
	default:
		// Try 1Password first, fall back to keyring
		op := NewOnePasswordBackend("")
		if op.IsAvailable() {
			currentBackend = op
			return currentBackend, nil
		}
		// Fall back to keyring
		kr := &KeyringBackend{}
		currentBackend = kr
		return currentBackend, nil
	}
}

// KeyringBackend provides password storage using system keyring
type KeyringBackend struct{}

func (k *KeyringBackend) openRing() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{ServiceName: serviceName})
}

func (k *KeyringBackend) SetPassword(profileName string, password string) error {
	if profileName == "" {
		return fmt.Errorf("profile name required")
	}
	r, err := k.openRing()
	if err != nil {
		return err
	}
	return r.Set(keyring.Item{Key: profileName + ":password", Data: []byte(password)})
}

func (k *KeyringBackend) GetPassword(profileName string) (string, error) {
	r, err := k.openRing()
	if err != nil {
		return "", err
	}
	it, err := r.Get(profileName + ":password")
	if err != nil {
		// keyring returns ErrKeyNotFound when key doesn't exist
		if err == keyring.ErrKeyNotFound {
			return "", nil
		}
		return "", err
	}
	return string(it.Data), nil
}

func (k *KeyringBackend) DeletePassword(profileName string) error {
	r, err := k.openRing()
	if err != nil {
		return err
	}
	return r.Remove(profileName + ":password")
}

// SetPassword stores a password for a profile name.
func SetPassword(profileName string, password string) error {
	backend, err := getBackend()
	if err != nil {
		return err
	}
	return backend.SetPassword(profileName, password)
}

// GetPassword retrieves a password for a profile name, empty string if missing.
func GetPassword(profileName string) (string, error) {
	backend, err := getBackend()
	if err != nil {
		return "", err
	}
	return backend.GetPassword(profileName)
}

// DeletePassword removes the stored password for a profile.
func DeletePassword(profileName string) error {
	backend, err := getBackend()
	if err != nil {
		return err
	}
	return backend.DeletePassword(profileName)
}
