// Package clipboard provides cross-platform clipboard access via shell commands.
package clipboard

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// ErrClipboardUnavailable is returned when clipboard access is not available.
var ErrClipboardUnavailable = errors.New("clipboard unavailable")

// IsAvailable checks if clipboard functionality is available on this system.
func IsAvailable() bool {
	switch runtime.GOOS {
	case "darwin":
		// macOS always has pbcopy
		_, err := exec.LookPath("pbcopy")
		return err == nil
	case "linux":
		// Check for xclip or xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			return true
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return true
		}
		return false
	default:
		return false
	}
}

// Copy copies the given text to the system clipboard.
// Returns ErrClipboardUnavailable if clipboard access is not available.
func Copy(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return ErrClipboardUnavailable
		}
	default:
		return ErrClipboardUnavailable
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
