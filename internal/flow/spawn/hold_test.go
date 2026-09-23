package spawn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHoldReason(t *testing.T) {
	root := t.TempDir()
	slot := filepath.Join(root, "alder")
	if err := os.MkdirAll(filepath.Join(root, ".holds"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(slot, 0755); err != nil {
		t.Fatal(err)
	}
	hold := filepath.Join(root, ".holds", "alder")

	if got, err := HoldReason(slot); err != nil || got != "" {
		t.Errorf("no hold: got %q, %v", got, err)
	}
	if err := os.WriteFile(hold, []byte("cedar #2943 reads .preserved/asr-2939\nmore\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got, _ := HoldReason(slot); got != "cedar #2943 reads .preserved/asr-2939" {
		t.Errorf("held: got %q", got)
	}
	// Tab completion adds a trailing slash; it names the same hold.
	if got, _ := HoldReason(slot + "/"); got == "" {
		t.Error("trailing slash: hold not found")
	}
	// A symlinked path to the slot names the same hold.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(slot, link); err != nil {
		t.Fatal(err)
	}
	if got, _ := HoldReason(link); got == "" {
		t.Error("symlinked path: hold not found")
	}
	if err := os.WriteFile(hold, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got, _ := HoldReason(slot); got != "(no reason given)" {
		t.Errorf("empty hold: got %q", got)
	}
	// Another slot in the same root is not held.
	if got, _ := HoldReason(filepath.Join(root, "cedar")); got != "" {
		t.Errorf("other slot: got %q", got)
	}
}
