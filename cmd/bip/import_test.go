package main

import (
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
)

func TestClassifyImport(t *testing.T) {
	existing := []reference.Reference{
		{ID: "ref1", DOI: "10.1234/abc", Title: "Paper One", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-1"}},
		{ID: "ref2", DOI: "", Title: "Paper Two (no DOI)", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-2"}},
		{ID: "ref3", DOI: "10.5678/xyz", Title: "Paper Three", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-3"}},
	}

	tests := []struct {
		name       string
		newRef     reference.Reference
		wantAction string
		wantReason string
		wantIdx    int
	}{
		{
			name:       "source ID match takes highest priority",
			newRef:     reference.Reference{ID: "different-id", DOI: "10.9999/different", Title: "Same source", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-1"}},
			wantAction: "update",
			wantReason: "source_id_match",
			wantIdx:    0,
		},
		{
			name:       "source ID match with different DOI still matches by source",
			newRef:     reference.Reference{ID: "new-id", DOI: "10.5678/xyz", Title: "Has ref3 DOI but ref1 source", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-1"}},
			wantAction: "update",
			wantReason: "source_id_match",
			wantIdx:    0, // Matches ref1 by source, not ref3 by DOI
		},
		{
			name:       "DOI match when no source ID match",
			newRef:     reference.Reference{ID: "new-id", DOI: "10.1234/abc", Title: "Updated Paper One", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-new"}},
			wantAction: "update",
			wantReason: "doi_match",
			wantIdx:    0,
		},
		{
			name:       "ID match without DOI returns update with id_match reason",
			newRef:     reference.Reference{ID: "ref2", DOI: "", Title: "Updated Paper Two", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-new"}},
			wantAction: "update",
			wantReason: "id_match",
			wantIdx:    1,
		},
		{
			name:       "ID match with different DOI still matches by DOI first",
			newRef:     reference.Reference{ID: "ref1", DOI: "10.5678/xyz", Title: "Matches ref3 by DOI", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-new"}},
			wantAction: "update",
			wantReason: "doi_match",
			wantIdx:    2, // Matches ref3 by DOI, not ref1 by ID
		},
		{
			name:       "ID match when new ref has DOI but no DOI match",
			newRef:     reference.Reference{ID: "ref2", DOI: "10.9999/new", Title: "New DOI for existing ID", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-new"}},
			wantAction: "update",
			wantReason: "id_match",
			wantIdx:    1,
		},
		{
			name:       "no match returns new",
			newRef:     reference.Reference{ID: "brand-new", DOI: "10.9999/brand-new", Title: "Brand New Paper", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-new"}},
			wantAction: "new",
			wantReason: "",
			wantIdx:    0,
		},
		{
			name:       "no match without DOI returns new",
			newRef:     reference.Reference{ID: "another-new", DOI: "", Title: "Another New Paper", Source: reference.ImportSource{Type: "paperpile", ID: "pp-uuid-new"}},
			wantAction: "new",
			wantReason: "",
			wantIdx:    0,
		},
		{
			name:       "empty source ID does not match",
			newRef:     reference.Reference{ID: "new-ref", DOI: "", Title: "No source ID", Source: reference.ImportSource{Type: "paperpile", ID: ""}},
			wantAction: "new",
			wantReason: "",
			wantIdx:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyImport(existing, tt.newRef)

			if got.action != tt.wantAction {
				t.Errorf("action = %q, want %q", got.action, tt.wantAction)
			}
			if got.reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", got.reason, tt.wantReason)
			}
			if tt.wantAction == "update" && got.existingIdx != tt.wantIdx {
				t.Errorf("existingIdx = %d, want %d", got.existingIdx, tt.wantIdx)
			}
		})
	}
}

func TestClassifyImportEmptyIDPanics(t *testing.T) {
	existing := []reference.Reference{
		{ID: "ref1", DOI: "10.1234/abc", Title: "Paper One"},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("classifyImport should panic with empty ID")
		}
	}()

	classifyImport(existing, reference.Reference{ID: "", Title: "No ID"})
}

func TestClassifyImportEmptyExistingList(t *testing.T) {
	var existing []reference.Reference

	got := classifyImport(existing, reference.Reference{ID: "new-ref", DOI: "10.1234/abc", Title: "New Paper"})

	if got.action != "new" {
		t.Errorf("action = %q, want %q", got.action, "new")
	}
}

func TestPersistImports_PreservesPostImportFields(t *testing.T) {
	// Locks in the cross-cutting fix for issue #145: an existing ref's
	// externally-resolved fields (PMCID, PMID, ArXivID, S2ID) and any PDF
	// path additions survive a re-import that doesn't carry those fields.
	// Without reference.MergeUpdate this would regress to the old wholesale
	// replace, silently wiping every `bip ncbi backfill` result.
	dir := t.TempDir()
	path := filepath.Join(dir, "refs.jsonl")

	existing := []reference.Reference{
		{
			ID:      "Smith2024-aa",
			DOI:     "10.1038/foo",
			Title:   "Foo",
			Source:  reference.ImportSource{Type: "paperpile", ID: "uuid-1"},
			PMCID:   "PMC123",
			PMID:    "456",
			S2ID:    "abc",
			PDFPath: "Smith/2024/Smith2024-aa.pdf",
		},
	}

	incoming := reference.Reference{
		ID:     "Smith2024-aa",
		DOI:    "10.1038/foo",
		Title:  "Foo (Paperpile re-export)",
		Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"},
	}

	actions := []storage.RefWithAction{
		{Ref: incoming, Action: "update", ExistingIdx: 0},
	}

	if err := persistImports(path, existing, actions); err != nil {
		t.Fatalf("persistImports: %v", err)
	}

	got, err := storage.ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(got))
	}
	r := got[0]
	if r.PMCID != "PMC123" {
		t.Errorf("PMCID wiped: %q", r.PMCID)
	}
	if r.PMID != "456" {
		t.Errorf("PMID wiped: %q", r.PMID)
	}
	if r.S2ID != "abc" {
		t.Errorf("S2ID wiped: %q", r.S2ID)
	}
	if r.PDFPath != "Smith/2024/Smith2024-aa.pdf" {
		t.Errorf("PDFPath wiped: %q", r.PDFPath)
	}
	if r.Title != "Foo (Paperpile re-export)" {
		t.Errorf("Title should update from incoming: %q", r.Title)
	}
}
