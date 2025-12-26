package credentials

import (
	"fmt"
	"os"

	"github.com/99designs/keyring"
	"github.com/vee-sh/veessh/internal/config"
)

const serviceName = "veessh"

// BackendType represents the credential storage backend
type BackendType string

const (
	BackendAuto      BackendType = "auto"      // Auto-detect (prefer 1Password if available, then keyring, then file)
	BackendKeyring   BackendType = "keyring"   // System keyring
	Backend1Password BackendType = "1password" // 1Password CLI
	BackendFile     BackendType = "file"      // Encrypted file (works on all platforms)
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

	// Priority: environment variable > config file > default (auto)
	// Check environment variable first (highest priority)
	if envBackend := os.Getenv("VEESSH_CREDENTIALS_BACKEND"); envBackend != "" {
		backendType = BackendType(envBackend)
	} else {
		// Check config file for default backend
		cfgPath, err := config.DefaultPath()
		if err == nil {
			cfg, err := config.Load(cfgPath)
			if err == nil && cfg.DefaultBackend != "" {
				backendType = BackendType(cfg.DefaultBackend)
			}
		}
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
		// Test if keyring works, fall back to file if it fails
		if _, err := kr.openRing(); err != nil {
			// Keyring not available, fall back to file backend
			fileBackend, err := NewFileBackend()
			if err != nil {
				return nil, fmt.Errorf("keyring not available and file backend failed: %w", err)
			}
			currentBackend = fileBackend
			return currentBackend, nil
		}
		currentBackend = kr
		return currentBackend, nil

	case BackendFile:
		fileBackend, err := NewFileBackend()
		if err != nil {
			return nil, fmt.Errorf("file backend failed: %w", err)
		}
		currentBackend = fileBackend
		return currentBackend, nil

	case BackendAuto:
		fallthrough
	default:
		// Try 1Password first
		op := NewOnePasswordBackend("")
		if op.IsAvailable() {
			currentBackend = op
			return currentBackend, nil
		}
		// Try keyring second
		kr := &KeyringBackend{}
		if _, err := kr.openRing(); err == nil {
			currentBackend = kr
			return currentBackend, nil
		}
		// Fall back to file backend (works everywhere)
		fileBackend, err := NewFileBackend()
		if err != nil {
			return nil, fmt.Errorf("all backends failed, file backend error: %w", err)
		}
		currentBackend = fileBackend
		return currentBackend, nil
	}
}

// KeyringBackend provides password storage using system keyring
type KeyringBackend struct{}

// NewKeyringBackend creates a new keyring backend
func NewKeyringBackend() *KeyringBackend {
	return &KeyringBackend{}
}

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

// GetBackend returns the current backend instance (for migration/testing)
func GetBackend() (Backend, error) {
	return getBackend()
}
