package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matsen/bipartite/internal/config"
)

// TestDBStale: a refs.jsonl newer than the query database (a pull with no
// rebuild) is stale; an older one, or a missing database, is not.
func TestDBStale(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(config.DBPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if dbStale(root) {
		t.Error("no database yet, but stale")
	}
	write := func(p string, at time.Time) {
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	write(config.DBPath(root), now)
	write(config.RefsPath(root), now.Add(-time.Hour))
	if dbStale(root) {
		t.Error("refs.jsonl older than the database, but stale")
	}
	write(config.RefsPath(root), now.Add(time.Hour))
	if !dbStale(root) {
		t.Error("refs.jsonl newer than the database, but not stale")
	}
}
