package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vee-sh/veessh/internal/config"
)

// FileBackend provides password storage using an encrypted file
// This backend works on all platforms (Linux, Windows, macOS)
type FileBackend struct {
	filePath string
	key      []byte
}

// NewFileBackend creates a new file-based backend
func NewFileBackend() (*FileBackend, error) {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}
	cfgDir := filepath.Dir(cfgPath)
	filePath := filepath.Join(cfgDir, "passwords.enc")

	// Derive encryption key from user's home directory
	// This ensures the key is unique per user but doesn't require a master password
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	key := deriveKey(home)

	return &FileBackend{
		filePath: filePath,
		key:      key,
	}, nil
}

// deriveKey creates a 32-byte key from the user's home directory
func deriveKey(home string) []byte {
	// Use a constant salt combined with home directory
	// This ensures the key is deterministic per user
	salt := []byte("veessh-password-encryption-salt-v1")
	data := append([]byte(home), salt...)
	hash := sha256.Sum256(data)
	return hash[:]
}

// loadPasswords loads and decrypts the password file
func (f *FileBackend) loadPasswords() (map[string]string, error) {
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to read password file: %w", err)
	}

	// Decrypt
	plaintext, err := f.decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password file: %w", err)
	}

	// Parse JSON
	var passwords map[string]string
	if err := json.Unmarshal(plaintext, &passwords); err != nil {
		return nil, fmt.Errorf("failed to parse password file: %w", err)
	}

	if passwords == nil {
		passwords = make(map[string]string)
	}

	return passwords, nil
}

// savePasswords encrypts and saves the password file
func (f *FileBackend) savePasswords(passwords map[string]string) error {
	// Serialize to JSON
	data, err := json.Marshal(passwords)
	if err != nil {
		return fmt.Errorf("failed to marshal passwords: %w", err)
	}

	// Encrypt
	encrypted, err := f.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt passwords: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(f.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to temp file first
	tmpPath := f.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, encrypted, 0o600); err != nil {
		return fmt.Errorf("failed to write password file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, f.filePath); err != nil {
		return fmt.Errorf("failed to rename password file: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-256-GCM
func (f *FileBackend) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func (f *FileBackend) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// SetPassword stores a password in the encrypted file
func (f *FileBackend) SetPassword(profileName string, password string) error {
	if profileName == "" {
		return fmt.Errorf("profile name required")
	}

	passwords, err := f.loadPasswords()
	if err != nil {
		return err
	}

	passwords[profileName] = password

	return f.savePasswords(passwords)
}

// GetPassword retrieves a password from the encrypted file
func (f *FileBackend) GetPassword(profileName string) (string, error) {
	if profileName == "" {
		return "", fmt.Errorf("profile name required")
	}

	passwords, err := f.loadPasswords()
	if err != nil {
		return "", err
	}

	password, ok := passwords[profileName]
	if !ok {
		return "", nil // Not found, return empty (consistent with other backends)
	}

	return password, nil
}

// DeletePassword removes a password from the encrypted file
func (f *FileBackend) DeletePassword(profileName string) error {
	if profileName == "" {
		return fmt.Errorf("profile name required")
	}

	passwords, err := f.loadPasswords()
	if err != nil {
		return err
	}

	delete(passwords, profileName)

	return f.savePasswords(passwords)
}

