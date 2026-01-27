package clipboard

import (
	"testing"
)

func TestIsAvailable(t *testing.T) {
	// This test just verifies the function doesn't panic
	// Actual availability depends on the system
	_ = IsAvailable()
}

func TestCopy(t *testing.T) {
	if !IsAvailable() {
		t.Skip("clipboard not available on this system")
	}

	// Test that Copy doesn't error with valid text
	testText := "test clipboard content"
	if err := Copy(testText); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	// Note: We can't easily verify clipboard contents in automated tests
	// but at least verify the operation doesn't error
}

func TestCopyEmptyString(t *testing.T) {
	if !IsAvailable() {
		t.Skip("clipboard not available on this system")
	}

	// Test that Copy handles empty string
	if err := Copy(""); err != nil {
		t.Fatalf("Copy of empty string failed: %v", err)
	}
}

func TestGetClipboardCommand(t *testing.T) {
	// Test that getClipboardCommand returns a consistent result
	// (either a valid command or an error, but not both)
	cmd, err := getClipboardCommand()
	if err != nil {
		// Error is acceptable (clipboard may not be available)
		if cmd != nil {
			t.Error("getClipboardCommand returned both command and error")
		}
	} else {
		// Command should be non-nil when no error
		if cmd == nil {
			t.Error("getClipboardCommand returned nil command with no error")
		}
	}
}
