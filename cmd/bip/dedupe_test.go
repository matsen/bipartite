package main

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestFindDuplicateGroups(t *testing.T) {
	tests := []struct {
		name       string
		refs       []reference.Reference
		wantGroups int
		wantDupes  int // total duplicates across all groups
	}{
		{
			name: "no duplicates",
			refs: []reference.Reference{
				{ID: "A", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "B", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-2"}},
				{ID: "C", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-3"}},
			},
			wantGroups: 0,
			wantDupes:  0,
		},
		{
			name: "one duplicate pair",
			refs: []reference.Reference{
				{ID: "A", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "A-2", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "B", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-2"}},
			},
			wantGroups: 1,
			wantDupes:  1,
		},
		{
			name: "multiple duplicates same source",
			refs: []reference.Reference{
				{ID: "A", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "A-2", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "A-3", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "A-4", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
			},
			wantGroups: 1,
			wantDupes:  3, // 3 duplicates (A-2, A-3, A-4)
		},
		{
			name: "multiple duplicate groups",
			refs: []reference.Reference{
				{ID: "A", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "A-2", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "B", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-2"}},
				{ID: "B-2", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-2"}},
				{ID: "C", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-3"}},
			},
			wantGroups: 2,
			wantDupes:  2,
		},
		{
			name: "different source types not duplicates",
			refs: []reference.Reference{
				{ID: "A", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "B", Source: reference.ImportSource{Type: "s2", ID: "uuid-1"}}, // Same ID, different type
			},
			wantGroups: 0,
			wantDupes:  0,
		},
		{
			name: "empty source ID skipped",
			refs: []reference.Reference{
				{ID: "A", Source: reference.ImportSource{Type: "manual", ID: ""}},
				{ID: "B", Source: reference.ImportSource{Type: "manual", ID: ""}},
			},
			wantGroups: 0,
			wantDupes:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := findDuplicateGroups(tt.refs)

			if len(groups) != tt.wantGroups {
				t.Errorf("got %d groups, want %d", len(groups), tt.wantGroups)
			}

			totalDupes := 0
			for _, g := range groups {
				totalDupes += len(g.Duplicates)
			}
			if totalDupes != tt.wantDupes {
				t.Errorf("got %d total duplicates, want %d", totalDupes, tt.wantDupes)
			}
		})
	}
}

func TestFindDuplicateGroups_PrimaryIsFirst(t *testing.T) {
	refs := []reference.Reference{
		{ID: "First", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
		{ID: "Second", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
		{ID: "Third", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
	}

	groups := findDuplicateGroups(refs)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	if groups[0].Primary != "First" {
		t.Errorf("Primary = %q, want %q", groups[0].Primary, "First")
	}

	expectedDupes := []string{"Second", "Third"}
	if len(groups[0].Duplicates) != len(expectedDupes) {
		t.Fatalf("Duplicates len = %d, want %d", len(groups[0].Duplicates), len(expectedDupes))
	}
	for i, d := range expectedDupes {
		if groups[0].Duplicates[i] != d {
			t.Errorf("Duplicates[%d] = %q, want %q", i, groups[0].Duplicates[i], d)
		}
	}
}
