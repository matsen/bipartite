package flow

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndReadLastCheckin(t *testing.T) {
	// Work in a temp directory so we don't clobber real state
	dir := t.TempDir()

	now := time.Now().Truncate(time.Millisecond) // JSON loses sub-ms precision

	if err := WriteLastCheckin(dir, now); err != nil {
		t.Fatalf("WriteLastCheckin: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, StateFile)); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	got := ReadLastCheckin(dir)
	if !got.Equal(now) {
		t.Errorf("ReadLastCheckin = %v, want %v", got, now)
	}
}

func TestReadLastCheckinMissingFile(t *testing.T) {
	dir := t.TempDir()

	got := ReadLastCheckin(dir)
	if !got.IsZero() {
		t.Errorf("ReadLastCheckin with no file = %v, want zero time", got)
	}
}

func TestReadLastCheckinInvalidJSON(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, StateFile), []byte("not json"), 0644)

	got := ReadLastCheckin(dir)
	if !got.IsZero() {
		t.Errorf("ReadLastCheckin with invalid JSON = %v, want zero time", got)
	}
}
