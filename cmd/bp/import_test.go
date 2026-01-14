package main

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestClassifyImport(t *testing.T) {
	existing := []reference.Reference{
		{ID: "ref1", DOI: "10.1234/abc", Title: "Paper One"},
		{ID: "ref2", DOI: "", Title: "Paper Two (no DOI)"},
		{ID: "ref3", DOI: "10.5678/xyz", Title: "Paper Three"},
	}

	tests := []struct {
		name       string
		newRef     reference.Reference
		wantAction string
		wantReason string
		wantIdx    int
	}{
		{
			name:       "DOI match returns update with doi_match reason",
			newRef:     reference.Reference{ID: "new-id", DOI: "10.1234/abc", Title: "Updated Paper One"},
			wantAction: "update",
			wantReason: "doi_match",
			wantIdx:    0,
		},
		{
			name:       "ID match without DOI returns update with id_match reason",
			newRef:     reference.Reference{ID: "ref2", DOI: "", Title: "Updated Paper Two"},
			wantAction: "update",
			wantReason: "id_match",
			wantIdx:    1,
		},
		{
			name:       "ID match with different DOI still matches by DOI first",
			newRef:     reference.Reference{ID: "ref1", DOI: "10.5678/xyz", Title: "Matches ref3 by DOI"},
			wantAction: "update",
			wantReason: "doi_match",
			wantIdx:    2, // Matches ref3 by DOI, not ref1 by ID
		},
		{
			name:       "ID match when new ref has DOI but no DOI match",
			newRef:     reference.Reference{ID: "ref2", DOI: "10.9999/new", Title: "New DOI for existing ID"},
			wantAction: "update",
			wantReason: "id_match",
			wantIdx:    1,
		},
		{
			name:       "no match returns new",
			newRef:     reference.Reference{ID: "brand-new", DOI: "10.9999/brand-new", Title: "Brand New Paper"},
			wantAction: "new",
			wantReason: "",
			wantIdx:    0,
		},
		{
			name:       "no match without DOI returns new",
			newRef:     reference.Reference{ID: "another-new", DOI: "", Title: "Another New Paper"},
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
