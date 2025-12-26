package credentials

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// OnePasswordBackend provides password storage using 1Password CLI
type OnePasswordBackend struct {
	vault string // Optional vault name
}

// NewOnePasswordBackend creates a new 1Password backend
// If vault is empty, uses the default vault
func NewOnePasswordBackend(vault string) *OnePasswordBackend {
	return &OnePasswordBackend{vault: vault}
}

// IsAvailable checks if 1Password CLI is installed and authenticated
func (op *OnePasswordBackend) IsAvailable() bool {
	cmd := exec.Command("op", "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	// Check if signed in
	cmd = exec.Command("op", "account", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// SetPassword stores a password in 1Password
// Uses item title: "veessh - profile-name"
func (op *OnePasswordBackend) SetPassword(profileName string, password string) error {
	if profileName == "" {
		return fmt.Errorf("profile name required")
	}

	itemTitle := fmt.Sprintf("veessh - %s", profileName)
	itemRef := op.getItemRef(profileName)

	// Check if item already exists
	if op.itemExists(profileName) {
		// Update existing item
		args := []string{"item", "edit", itemRef}
		if op.vault != "" {
			args = append(args, "--vault", op.vault)
		}
		args = append(args, fmt.Sprintf("password=%s", password))

		cmd := exec.Command("op", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("1password CLI error: %w (output: %s)", err, string(output))
		}
		return nil
	}

	// Create new item
	args := []string{"item", "create"}
	if op.vault != "" {
		args = append(args, "--vault", op.vault)
	}
	args = append(args,
		"--category", "password",
		"--title", itemTitle,
		fmt.Sprintf("password=%s", password),
		fmt.Sprintf("notesPlain=SSH connection profile: %s\n\nManaged by veessh", profileName),
	)

	cmd := exec.Command("op", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("1password CLI error: %w (output: %s)", err, string(output))
	}

	return nil
}

// GetPassword retrieves a password from 1Password
func (op *OnePasswordBackend) GetPassword(profileName string) (string, error) {
	if profileName == "" {
		return "", fmt.Errorf("profile name required")
	}

	itemRef := op.getItemRef(profileName)
	args := []string{"item", "get", itemRef, "--fields", "label=password", "--reveal"}
	if op.vault != "" {
		args = append(args, "--vault", op.vault)
	}

	cmd := exec.Command("op", args...)
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(string(output), "isn't in") || strings.Contains(string(output), "not found") || strings.Contains(string(output), "No item found") {
			return "", nil // Not found, return empty (similar to keyring behavior)
		}
		return "", fmt.Errorf("1password CLI error: %w (output: %s)", err, string(output))
	}

	// Parse JSON output
	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		// Try plain text if JSON parsing fails
		return strings.TrimSpace(string(output)), nil
	}

	return strings.TrimSpace(result.Value), nil
}

// DeletePassword removes a password from 1Password
func (op *OnePasswordBackend) DeletePassword(profileName string) error {
	if profileName == "" {
		return fmt.Errorf("profile name required")
	}

	itemRef := op.getItemRef(profileName)
	args := []string{"item", "delete", itemRef}
	if op.vault != "" {
		args = append(args, "--vault", op.vault)
	}

	cmd := exec.Command("op", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("1password CLI error: %w", err)
	}

	return nil
}

// itemExists checks if an item exists in 1Password
func (op *OnePasswordBackend) itemExists(profileName string) bool {
	itemRef := op.getItemRef(profileName)
	args := []string{"item", "get", itemRef}
	if op.vault != "" {
		args = append(args, "--vault", op.vault)
	}

	cmd := exec.Command("op", args...)
	return cmd.Run() == nil
}

// getItemRef returns the 1Password item reference (title search)
func (op *OnePasswordBackend) getItemRef(profileName string) string {
	itemTitle := fmt.Sprintf("veessh - %s", profileName)
	// Use title search - 1Password CLI supports searching by title
	return itemTitle
}

