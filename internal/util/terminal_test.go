package util

import (
	"context"
	"os/exec"
	"testing"
)

func TestRunAttachedSuccess(t *testing.T) {
	cmd := exec.Command("echo", "hello")
	err := RunAttached(cmd)
	if err != nil {
		t.Errorf("RunAttached() error = %v, want nil", err)
	}
}

func TestRunAttachedExitCode(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 1")
	err := RunAttached(cmd)
	if err == nil {
		t.Error("RunAttached() should return error for non-zero exit")
	}

	// Should be an ExitError
	if _, ok := err.(*exec.ExitError); !ok {
		t.Errorf("RunAttached() error type = %T, want *exec.ExitError", err)
	}
}

func TestRunAttachedNotFound(t *testing.T) {
	cmd := exec.Command("nonexistent-command-12345")
	err := RunAttached(cmd)
	if err == nil {
		t.Error("RunAttached() should return error for non-existent command")
	}
}

func TestRunAttachedWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "sleep", "10")

	// Cancel immediately
	cancel()

	err := RunAttached(cmd)
	if err == nil {
		t.Error("RunAttached() should return error when context is cancelled")
	}
}

