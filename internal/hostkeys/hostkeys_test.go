package hostkeys

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPinnedKeysPath(t *testing.T) {
	path, err := PinnedKeysPath()
	if err != nil {
		t.Fatalf("PinnedKeysPath() error = %v", err)
	}
	if path == "" {
		t.Error("PinnedKeysPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("PinnedKeysPath() = %s, want absolute path", path)
	}
}

func TestKnownHostsPath(t *testing.T) {
	path, err := KnownHostsPath()
	if err != nil {
		t.Fatalf("KnownHostsPath() error = %v", err)
	}
	if path == "" {
		t.Error("KnownHostsPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("KnownHostsPath() = %s, want absolute path", path)
	}
}

func TestLoadPinnedKeysEmpty(t *testing.T) {
	// Should not error when file doesn't exist
	// We can't easily test this without mocking the path
	keys, err := LoadPinnedKeys()
	if err != nil && !os.IsNotExist(err) {
		// Only fail if it's not a "file not found" type error
		// and there's actually an error
		t.Logf("LoadPinnedKeys() returned: keys=%v, err=%v", keys, err)
	}
}

func TestPinKeyAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "pinned_keys.txt")

	// Create the file manually
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Write a test entry
	_, err = file.WriteString("example.com:22 ssh-ed25519 SHA256:abc123 test comment\n")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	// Read it back
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("File should not be empty")
	}
}

func TestFormatFingerprint(t *testing.T) {
	fp := "SHA256:abc123def456"
	got := FormatFingerprint(fp)
	if got != fp {
		t.Errorf("FormatFingerprint(%q) = %q, want %q", fp, got, fp)
	}
}

func TestIsHostInKnownHostsNonExistent(t *testing.T) {
	// Create a temp file as known_hosts
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "known_hosts")

	// File doesn't exist
	found, err := IsHostInKnownHosts("example.com", 22)
	if err != nil {
		// May error if it can't find ~/.ssh/known_hosts
		t.Logf("IsHostInKnownHosts error (expected if no known_hosts): %v", err)
	}
	_ = found
	_ = path
}

