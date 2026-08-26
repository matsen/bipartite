package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = orig

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stderr: %v", err)
	}
	return string(out)
}

func TestWarnSuspectPages_FiresForPlaceholder(t *testing.T) {
	refs := []reference.Reference{
		{ID: "Pae2025-kp", Pages: "1-9", Volume: ""},
	}

	stderr := captureStderr(t, func() { warnSuspectPages(refs) })

	if !strings.Contains(stderr, "Pae2025-kp") || !strings.Contains(stderr, "warning:") {
		t.Errorf("warnSuspectPages() should warn about placeholder pages, got stderr:\n%s", stderr)
	}
}

func TestWarnSuspectPages_SilentAfterVolumeAssigned(t *testing.T) {
	refs := []reference.Reference{
		{ID: "Pae2025-kp", Pages: "486-494", Volume: "641", Issue: "8062"},
	}

	stderr := captureStderr(t, func() { warnSuspectPages(refs) })

	if stderr != "" {
		t.Errorf("warnSuspectPages() should not warn once volume/issue are set, got stderr:\n%s", stderr)
	}
}

func TestWarnSuspectPages_SilentForLegitimatePagination(t *testing.T) {
	refs := []reference.Reference{
		{ID: "Zhang2024-uh", Pages: "1-56", Volume: "25"},
	}

	stderr := captureStderr(t, func() { warnSuspectPages(refs) })

	if stderr != "" {
		t.Errorf("warnSuspectPages() should not warn for legitimate 1-N pagination with a volume, got stderr:\n%s", stderr)
	}
}
