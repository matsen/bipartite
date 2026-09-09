package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

// linkpubRefsPath sets up a temp repo containing refs and returns its path.
func linkpubRefsPath(t *testing.T, refs []reference.Reference) string {
	t.Helper()
	repoRoot := t.TempDir()
	refsPath := config.RefsPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(refsPath), 0o755); err != nil {
		t.Fatalf("creating refs dir: %v", err)
	}
	if err := storage.WriteAll(refsPath, refs); err != nil {
		t.Fatalf("writing refs: %v", err)
	}
	return refsPath
}

// TestWriteUpdatedRefsPersistsPartialProgress covers the mid-run rate-limit
// path: link-published calls writeUpdatedRefs before bailing out on a 429, so
// links already found in that run survive. Before this fix a mid-run rate
// limit discarded them.
func TestWriteUpdatedRefsPersistsPartialProgress(t *testing.T) {
	refs := []reference.Reference{
		{ID: "P1", DOI: "10.1101/preprint-1", Venue: "bioRxiv"},
		{ID: "P2", DOI: "10.1101/preprint-2", Venue: "bioRxiv"},
		{ID: "P3", DOI: "10.1101/preprint-3", Venue: "bioRxiv"},
	}
	refsPath := linkpubRefsPath(t, refs)

	// P1 was linked before the rate limit hit; P2 and P3 were never reached.
	linked := refs[0]
	linked.Supersedes = "10.1038/published-1"
	updated := map[int]reference.Reference{0: linked}

	if err := writeUpdatedRefs(refsPath, refs, updated); err != nil {
		t.Fatalf("writeUpdatedRefs: %v", err)
	}

	got, err := storage.ReadAll(refsPath)
	if err != nil {
		t.Fatalf("reading refs back: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d refs, want 3", len(got))
	}
	if got[0].Supersedes != "10.1038/published-1" {
		t.Errorf("P1.Supersedes = %q, want the link found before the rate limit", got[0].Supersedes)
	}
	for _, ref := range got[1:] {
		if ref.Supersedes != "" {
			t.Errorf("%s.Supersedes = %q, want empty (never reached)", ref.ID, ref.Supersedes)
		}
	}
}

// TestWriteUpdatedRefsNoopWhenNothingLinked guards the other direction: a run
// that linked nothing must not rewrite refs.jsonl at all.
func TestWriteUpdatedRefsNoopWhenNothingLinked(t *testing.T) {
	refs := []reference.Reference{{ID: "P1", DOI: "10.1101/preprint-1", Venue: "bioRxiv"}}
	refsPath := linkpubRefsPath(t, refs)

	before, err := os.Stat(refsPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if err := writeUpdatedRefs(refsPath, refs, map[int]reference.Reference{}); err != nil {
		t.Fatalf("writeUpdatedRefs: %v", err)
	}

	after, err := os.Stat(refsPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("refs.jsonl was rewritten despite no links being found")
	}
}

func TestIsPreprint(t *testing.T) {
	tests := []struct {
		venue string
		want  bool
	}{
		{"bioRxiv", true},
		{"biorxiv", true},
		{"medRxiv", true},
		{"arXiv", true},
		{"arXiv preprint arXiv:2301.00001", true},
		{"Nature", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.venue, func(t *testing.T) {
			if got := isPreprint(reference.Reference{Venue: tt.venue}); got != tt.want {
				t.Errorf("isPreprint(venue=%q) = %v, want %v", tt.venue, got, tt.want)
			}
		})
	}
}
