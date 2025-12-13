package hostkeys

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// PinnedKey represents a pinned host key
type PinnedKey struct {
	Host        string    `yaml:"host"`
	Port        int       `yaml:"port"`
	KeyType     string    `yaml:"keyType"`
	Fingerprint string    `yaml:"fingerprint"`
	PinnedAt    time.Time `yaml:"pinnedAt"`
	Comment     string    `yaml:"comment,omitempty"`
}

// KnownHostsPath returns the path to the known_hosts file
func KnownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// PinnedKeysPath returns the path to veessh's pinned keys file
func PinnedKeysPath() (string, error) {
	cfgHome := os.Getenv("XDG_CONFIG_HOME")
	if cfgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cfgHome = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgHome, "veessh", "pinned_keys.txt"), nil
}

// GetHostFingerprint connects to a host and returns the server's key fingerprint
func GetHostFingerprint(host string, port int) (keyType, fingerprint string, err error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	config := &ssh.ClientConfig{
		User: "probe",
		Auth: []ssh.AuthMethod{},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			keyType = key.Type()
			hash := sha256.Sum256(key.Marshal())
			fingerprint = "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
			return nil
		},
		Timeout: 10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if conn != nil {
		conn.Close()
	}

	// We expect auth to fail, but we should have captured the fingerprint
	if fingerprint != "" {
		return keyType, fingerprint, nil
	}

	if err != nil {
		return "", "", fmt.Errorf("failed to get host key: %w", err)
	}
	return "", "", fmt.Errorf("no host key received")
}

// IsHostInKnownHosts checks if a host is in the SSH known_hosts file
func IsHostInKnownHosts(host string, port int) (bool, error) {
	path, err := KnownHostsPath()
	if err != nil {
		return false, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	// Build the host pattern to search for
	hostPattern := host
	if port != 22 {
		hostPattern = fmt.Sprintf("[%s]:%d", host, port)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		hosts := strings.Split(fields[0], ",")
		for _, h := range hosts {
			if h == hostPattern || h == host {
				return true, nil
			}
		}
	}
	return false, scanner.Err()
}

// LoadPinnedKeys loads the list of pinned keys
func LoadPinnedKeys() ([]PinnedKey, error) {
	path, err := PinnedKeysPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var keys []PinnedKey
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: host:port keyType fingerprint [comment]
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		hostPort := fields[0]
		parts := strings.Split(hostPort, ":")
		host := parts[0]
		port := 22
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &port)
		}
		key := PinnedKey{
			Host:        host,
			Port:        port,
			KeyType:     fields[1],
			Fingerprint: fields[2],
		}
		if len(fields) > 3 {
			key.Comment = strings.Join(fields[3:], " ")
		}
		keys = append(keys, key)
	}
	return keys, scanner.Err()
}

// PinKey adds a key to the pinned keys file
func PinKey(host string, port int, keyType, fingerprint, comment string) error {
	path, err := PinnedKeysPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	line := fmt.Sprintf("%s:%d %s %s", host, port, keyType, fingerprint)
	if comment != "" {
		line += " " + comment
	}
	_, err = fmt.Fprintln(file, line)
	return err
}

// VerifyPinnedKey checks if a host's current key matches the pinned key
func VerifyPinnedKey(host string, port int) (matched bool, pinnedFP, currentFP string, err error) {
	keys, err := LoadPinnedKeys()
	if err != nil {
		return false, "", "", err
	}

	// Find pinned key for this host
	for _, k := range keys {
		if k.Host == host && k.Port == port {
			pinnedFP = k.Fingerprint
			break
		}
	}

	if pinnedFP == "" {
		return false, "", "", nil // No pinned key
	}

	// Get current fingerprint
	_, currentFP, err = GetHostFingerprint(host, port)
	if err != nil {
		return false, pinnedFP, "", err
	}

	return pinnedFP == currentFP, pinnedFP, currentFP, nil
}

// FormatFingerprint formats a fingerprint for display
func FormatFingerprint(fp string) string {
	return fp
}

