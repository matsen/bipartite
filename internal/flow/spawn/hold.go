package spawn

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// HoldReason reports whether a conductor holds the slot at dir, and why.
// A hold is the file <clone root>/.holds/<slot name>, where the clone root
// is dir's parent; it sits outside the clone so reclaiming the slot leaves
// it in place. Returns "" when there is no hold, else the file's first
// line, or "(no reason given)" when that is empty.
func HoldReason(dir string) (string, error) {
	slot, err := canonicalizePath(dir)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(slot), ".holds", filepath.Base(slot)))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if reason := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0]); reason != "" {
		return reason, nil
	}
	return "(no reason given)", nil
}
