package main

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestFindDuplicateDOIs(t *testing.T) {
	tests := []struct {
		name       string
		refs       []reference.Reference
		wantGroups int
		wantIDs    []string // IDs of the first (only) group, if wantGroups == 1
	}{
		{
			name: "identical DOIs differing only in case are one group",
			refs: []reference.Reference{
				{ID: "A", DOI: "10.1093/molbev/msl010"},
				{ID: "B", DOI: "10.1093/MOLBEV/MSL010"},
			},
			wantGroups: 1,
			wantIDs:    []string{"A", "B"},
		},
		{
			name: "genuinely different DOIs are not grouped",
			refs: []reference.Reference{
				{ID: "A", DOI: "10.1/aaa"},
				{ID: "B", DOI: "10.1/bbb"},
			},
			wantGroups: 0,
		},
		{
			name: "blank and prefix-only DOIs normalize to empty and are skipped",
			refs: []reference.Reference{
				{ID: "A", DOI: "  "},
				{ID: "B", DOI: "DOI:"},
				{ID: "C", DOI: ""},
			},
			wantGroups: 0,
		},
		{
			name: "URL prefix and case variant normalize to the same DOI",
			refs: []reference.Reference{
				{ID: "A", DOI: "https://doi.org/10.1234/x"},
				{ID: "B", DOI: "10.1234/X"},
			},
			wantGroups: 1,
			wantIDs:    []string{"A", "B"},
		},
		{
			name: "same DOI groups even when year and authors disagree",
			refs: []reference.Reference{
				{ID: "Unknown2004-fc", DOI: "10.1023/a:1017067816551", Published: reference.PublicationDate{Year: 2004}},
				{ID: "Gerrish1998-bv", DOI: "10.1023/A:1017067816551", Published: reference.PublicationDate{Year: 1998}},
			},
			wantGroups: 1,
			wantIDs:    []string{"Unknown2004-fc", "Gerrish1998-bv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := findDuplicateDOIs(tt.refs)
			if len(issues) != tt.wantGroups {
				t.Fatalf("got %d groups, want %d: %+v", len(issues), tt.wantGroups, issues)
			}
			if tt.wantGroups == 1 {
				if len(issues[0].IDs) != len(tt.wantIDs) {
					t.Fatalf("got IDs %v, want %v", issues[0].IDs, tt.wantIDs)
				}
				for i, id := range tt.wantIDs {
					if issues[0].IDs[i] != id {
						t.Errorf("IDs[%d] = %q, want %q", i, issues[0].IDs[i], id)
					}
				}
			}
		})
	}
}
