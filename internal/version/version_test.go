package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	result := String()

	// Should contain version
	if !strings.Contains(result, Version) {
		t.Errorf("String() = %q, should contain Version %q", result, Version)
	}

	// Should contain commit
	if !strings.Contains(result, Commit) {
		t.Errorf("String() = %q, should contain Commit %q", result, Commit)
	}

	// Should contain date
	if !strings.Contains(result, Date) {
		t.Errorf("String() = %q, should contain Date %q", result, Date)
	}
}

func TestDefaultValues(t *testing.T) {
	// Default values should be set
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if Date == "" {
		t.Error("Date should not be empty")
	}
}

