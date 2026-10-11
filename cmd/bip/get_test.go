package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPaperNotesFile(t *testing.T) {
	root := t.TempDir()
	if got := paperNotesFile(root, "Smith2024-ab"); got != "" {
		t.Fatalf("no papers/ dir: got %q, want empty", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "papers"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "papers", "Smith2024-ab.md")
	if err := os.WriteFile(want, []byte("# notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := paperNotesFile(root, "Smith2024-ab"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := paperNotesFile(root, "Other2024-cd"); got != "" {
		t.Fatalf("other id: got %q, want empty", got)
	}
}
