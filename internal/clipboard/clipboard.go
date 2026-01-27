// Package clipboard provides cross-platform clipboard access via shell commands.
package clipboard

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// ErrClipboardUnavailable is returned when clipboard access is not available.
// On Linux, this typically means xclip or xsel is not installed.
var ErrClipboardUnavailable = errors.New("clipboard unavailable")

// getClipboardCommand returns the appropriate clipboard command for the current platform.
// Returns ErrClipboardUnavailable if no clipboard tool is found.
func getClipboardCommand() (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		// macOS always has pbcopy
		if _, err := exec.LookPath("pbcopy"); err != nil {
			return nil, ErrClipboardUnavailable
		}
		return exec.Command("pbcopy"), nil
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			return exec.Command("xclip", "-selection", "clipboard"), nil
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return exec.Command("xsel", "--clipboard", "--input"), nil
		}
		return nil, ErrClipboardUnavailable
	default:
		return nil, ErrClipboardUnavailable
	}
}

// IsAvailable checks if clipboard functionality is available on this system.
func IsAvailable() bool {
	_, err := getClipboardCommand()
	return err == nil
}

// Copy copies the given text to the system clipboard.
// Returns ErrClipboardUnavailable if clipboard access is not available.
func Copy(text string) error {
	cmd, err := getClipboardCommand()
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
