package s2

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func sampleRefs() []reference.Reference {
	return []reference.Reference{
		{
			ID:     "Zhang2018-vi",
			DOI:    "10.1038/Nature12373",
			Title:  "Variational Inference",
			Source: reference.ImportSource{Type: "s2", ID: "s2paper1"},
		},
		{
			ID:     "Doe2020-nn",
			DOI:    "",
			Title:  "Neural Networks",
			Source: reference.ImportSource{Type: "paperpile", ID: "pp123"},
		},
		{
			ID:     "Smith2019-os",
			DOI:    "10.1234/foo",
			Title:  "Origin of Species",
			Source: reference.ImportSource{Type: "s2", ID: "s2paper2"},
		},
	}
}

func TestLocalResolverFindByDOI(t *testing.T) {
	r := NewLocalResolverFromRefs(sampleRefs())

	// DOI lookup is normalized (case + prefix).
	ref, ok := r.FindByDOI("https://doi.org/10.1038/nature12373")
	if !ok {
		t.Fatal("FindByDOI: expected to find normalized DOI")
	}
	if ref.ID != "Zhang2018-vi" {
		t.Errorf("FindByDOI: got %q, want Zhang2018-vi", ref.ID)
	}

	if _, ok := r.FindByDOI("10.9999/missing"); ok {
		t.Error("FindByDOI: expected miss for unknown DOI")
	}
}

func TestLocalResolverFindByID(t *testing.T) {
	r := NewLocalResolverFromRefs(sampleRefs())

	ref, ok := r.FindByID("Doe2020-nn")
	if !ok {
		t.Fatal("FindByID: expected to find Doe2020-nn")
	}
	if ref.Title != "Neural Networks" {
		t.Errorf("FindByID: got title %q", ref.Title)
	}

	if _, ok := r.FindByID("Nobody1999-xx"); ok {
		t.Error("FindByID: expected miss for unknown ID")
	}
}

func TestLocalResolverFindByS2ID(t *testing.T) {
	r := NewLocalResolverFromRefs(sampleRefs())

	ref, ok := r.FindByS2ID("s2paper2")
	if !ok {
		t.Fatal("FindByS2ID: expected to find s2paper2")
	}
	if ref.ID != "Smith2019-os" {
		t.Errorf("FindByS2ID: got %q, want Smith2019-os", ref.ID)
	}

	// Non-s2 sources are not indexed by S2 ID.
	if _, ok := r.FindByS2ID("pp123"); ok {
		t.Error("FindByS2ID: paperpile source should not be indexed by S2 ID")
	}
}

func TestLocalResolverExistsLocally(t *testing.T) {
	r := NewLocalResolverFromRefs(sampleRefs())

	tests := []struct {
		name   string
		paper  S2Paper
		wantOK bool
		wantID string
	}{
		{
			name:   "match by DOI",
			paper:  S2Paper{ExternalIDs: ExternalIDs{DOI: "10.1038/nature12373"}},
			wantOK: true,
			wantID: "Zhang2018-vi",
		},
		{
			name:   "match by S2 ID",
			paper:  S2Paper{PaperID: "s2paper2"},
			wantOK: true,
			wantID: "Smith2019-os",
		},
		{
			name:   "no match",
			paper:  S2Paper{PaperID: "unknown", ExternalIDs: ExternalIDs{DOI: "10.0/none"}},
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, ok := r.ExistsLocally(tt.paper)
			if ok != tt.wantOK {
				t.Fatalf("ExistsLocally ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && ref.ID != tt.wantID {
				t.Errorf("ExistsLocally ID = %q, want %q", ref.ID, tt.wantID)
			}
		})
	}
}

func TestLocalResolverCount(t *testing.T) {
	r := NewLocalResolverFromRefs(sampleRefs())
	if got := r.Count(); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}
	if got := NewLocalResolverFromRefs(nil).Count(); got != 0 {
		t.Errorf("Count() on empty = %d, want 0", got)
	}
}

func TestLocalResolverRefsWithDOI(t *testing.T) {
	r := NewLocalResolverFromRefs(sampleRefs())
	withDOI := r.RefsWithDOI()
	if len(withDOI) != 2 {
		t.Fatalf("RefsWithDOI len = %d, want 2", len(withDOI))
	}
	for _, ref := range withDOI {
		if ref.DOI == "" {
			t.Errorf("RefsWithDOI returned ref with empty DOI: %q", ref.ID)
		}
	}
}
