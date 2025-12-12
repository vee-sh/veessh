package credentials

import (
	"fmt"

	"github.com/99designs/keyring"
)

const serviceName = "veessh"

func openRing() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{ServiceName: serviceName})
}

// SetPassword stores a password for a profile name.
func SetPassword(profileName string, password string) error {
	if profileName == "" {
		return fmt.Errorf("profile name required")
	}
	r, err := openRing()
	if err != nil {
		return err
	}
	return r.Set(keyring.Item{Key: profileName + ":password", Data: []byte(password)})
}

// GetPassword retrieves a password for a profile name, empty string if missing.
func GetPassword(profileName string) (string, error) {
	r, err := openRing()
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

// DeletePassword removes the stored password for a profile.
func DeletePassword(profileName string) error {
	r, err := openRing()
	if err != nil {
		return err
	}
	return r.Remove(profileName + ":password")
}
