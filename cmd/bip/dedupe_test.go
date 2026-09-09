package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
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

func TestFindTitleGroups(t *testing.T) {
	tests := []struct {
		name       string
		refs       []reference.Reference
		wantGroups int
	}{
		{
			name: "same normalized title, different DOIs and source IDs",
			refs: []reference.Reference{
				{ID: "A", Title: "A Great Paper", DOI: "10.1/aaa", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
				{ID: "B", Title: "a great paper", DOI: "10.2/bbb", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-2"}},
			},
			wantGroups: 1,
		},
		{
			name: "titles differing only in punctuation and case",
			refs: []reference.Reference{
				{ID: "A", Title: "Some: Paper-Title"},
				{ID: "B", Title: "some paper title"},
			},
			wantGroups: 1,
		},
		{
			name: "three-member title group",
			refs: []reference.Reference{
				{ID: "X", Title: "Restricted after Ligation"},
				{ID: "Y", Title: "restricted after ligation"},
				{ID: "Z", Title: "Restricted After Ligation!"},
			},
			wantGroups: 1,
		},
		{
			name: "unknown-title sentinel is never grouped",
			refs: []reference.Reference{
				{ID: "A", Title: "[no title]"},
				{ID: "B", Title: "[no title]"},
			},
			wantGroups: 0,
		},
		{
			name: "HTML markup breaks title match (documented limitation)",
			refs: []reference.Reference{
				{ID: "A", Title: "Restricted after <i>IGHG2</i>"},
				{ID: "B", Title: "Restricted after IGHG2"},
			},
			wantGroups: 0,
		},
		{
			name: "unicode-aware normalization treats umlaut and ASCII as distinct",
			refs: []reference.Reference{
				{ID: "A", Title: "Müller cells"},
				{ID: "B", Title: "Muller cells"},
			},
			wantGroups: 0,
		},
		{
			name: "empty title is never grouped",
			refs: []reference.Reference{
				{ID: "A", Title: ""},
				{ID: "B", Title: ""},
			},
			wantGroups: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := findTitleGroups(tt.refs)
			if len(groups) != tt.wantGroups {
				t.Fatalf("got %d groups, want %d: %+v", len(groups), tt.wantGroups, groups)
			}
			for _, g := range groups {
				if g.MatchBasis != "title" {
					t.Errorf("MatchBasis = %q, want %q", g.MatchBasis, "title")
				}
				if g.Primary != "" {
					t.Errorf("Primary = %q, want empty for title group", g.Primary)
				}
				if len(g.Members) < 2 {
					t.Errorf("Members = %v, want 2+", g.Members)
				}
			}
		})
	}
}

func TestFindTitleGroups_SupersedesSuppression(t *testing.T) {
	tests := []struct {
		name       string
		refs       []reference.Reference
		wantGroups int
	}{
		{
			name: "pair fully linked by supersedes is suppressed",
			refs: []reference.Reference{
				{ID: "Preprint", Title: "A Shared Title", DOI: "10.1/preprint"},
				{ID: "Published", Title: "a shared title", DOI: "10.1/published", Supersedes: "10.1/PREPRINT"},
			},
			wantGroups: 0,
		},
		{
			name: "pair with no supersedes link is still reported",
			refs: []reference.Reference{
				{ID: "Preprint", Title: "A Shared Title", DOI: "10.1/preprint"},
				{ID: "Published", Title: "a shared title", DOI: "10.1/published"},
			},
			wantGroups: 1,
		},
		{
			name: "three-member group fully chained by supersedes is suppressed",
			refs: []reference.Reference{
				{ID: "A", Title: "A Shared Title", DOI: "10.1/a"},
				{ID: "B", Title: "a shared title", DOI: "10.1/b", Supersedes: "10.1/a"},
				{ID: "C", Title: "A SHARED TITLE", DOI: "10.1/c", Supersedes: "10.1/b"},
			},
			wantGroups: 0,
		},
		{
			name: "three-member group only partially linked is still reported",
			refs: []reference.Reference{
				{ID: "A", Title: "A Shared Title", DOI: "10.1/a"},
				{ID: "B", Title: "a shared title", DOI: "10.1/b", Supersedes: "10.1/a"},
				{ID: "C", Title: "A SHARED TITLE", DOI: "10.1/c"},
			},
			wantGroups: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := findTitleGroups(tt.refs)
			if len(groups) != tt.wantGroups {
				t.Fatalf("got %d groups, want %d: %+v", len(groups), tt.wantGroups, groups)
			}
		})
	}
}

func TestFindTitleGroups_ThreeMemberGroupHasAllMembers(t *testing.T) {
	refs := []reference.Reference{
		{ID: "Barbulescu2025-xt", Title: "A Shared Title"},
		{ID: "Mesin2026-ra", Title: "a shared title"},
		{ID: "Barbulescu2026-ub", Title: "A SHARED TITLE"},
	}

	groups := findTitleGroups(refs)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Members) != 3 {
		t.Fatalf("expected 3 members, got %d: %v", len(groups[0].Members), groups[0].Members)
	}
}

func TestFindDuplicateGroups_SourceIDAndTitleDoNotDoubleCount(t *testing.T) {
	// A pair that matches on both source ID and title is reported once per
	// basis — one source_id group and one title group — and contributes
	// exactly one entry to the duplicate count, not two.
	refs := []reference.Reference{
		{ID: "A", Title: "Same Title", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
		{ID: "B", Title: "same title", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
	}

	groups := findDuplicateGroups(refs)

	sourceIDGroups := 0
	titleGroups := 0
	totalDupes := 0
	for _, g := range groups {
		switch g.MatchBasis {
		case "source_id":
			sourceIDGroups++
			totalDupes += len(g.Duplicates)
		case "title":
			titleGroups++
		}
	}

	if sourceIDGroups != 1 {
		t.Errorf("sourceIDGroups = %d, want 1", sourceIDGroups)
	}
	if titleGroups != 1 {
		t.Errorf("titleGroups = %d, want 1", titleGroups)
	}
	if totalDupes != 1 {
		t.Errorf("TotalDupes-equivalent = %d, want 1", totalDupes)
	}
}

// mixedGroupFixture is one source-ID pair (A / A-2) plus one title-only pair
// (P / Q) — the shape that makes the report-only contract observable.
func mixedGroupFixture() []reference.Reference {
	return []reference.Reference{
		{ID: "A", Title: "Source Dupe", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
		{ID: "A-2", Title: "Source Dupe (copy)", Source: reference.ImportSource{Type: "paperpile", ID: "uuid-1"}},
		{ID: "P", Title: "Title Only Match", DOI: "10.1/preprint"},
		{ID: "Q", Title: "title only match", DOI: "10.2/published"},
	}
}

func TestSourceIDGroupsSelectsOnlyMergeableGroups(t *testing.T) {
	groups := findDuplicateGroups(mixedGroupFixture())

	mergeGroups := sourceIDGroups(groups)
	if len(mergeGroups) != 1 {
		t.Fatalf("expected 1 mergeable group, got %d", len(mergeGroups))
	}
	if mergeGroups[0].Primary != "A" || len(mergeGroups[0].Duplicates) != 1 || mergeGroups[0].Duplicates[0] != "A-2" {
		t.Errorf("unexpected merge group: %+v", mergeGroups[0])
	}
}

// TestFindTitleGroupsLeavesDuplicatesEmpty pins the invariant that keeps
// title groups harmless: they carry Members for review and never a
// Duplicates list that a merge could act on.
func TestFindTitleGroupsLeavesDuplicatesEmpty(t *testing.T) {
	groups := findTitleGroups(mixedGroupFixture())
	if len(groups) == 0 {
		t.Fatal("fixture should produce a title group")
	}
	for _, g := range groups {
		if len(g.Duplicates) != 0 || g.Primary != "" {
			t.Errorf("title group %q has merge fields set: Primary=%q Duplicates=%v", g.Title, g.Primary, g.Duplicates)
		}
	}
}

// TestPerformMergeNeverDropsTitleGroupMembers exercises the destructive path
// itself. The title group here is handed a populated Duplicates list — the
// shape a future caller could construct by mistake — and performMerge must
// still refuse to act on it, since a shared title is not paper identity.
func TestPerformMergeNeverDropsTitleGroupMembers(t *testing.T) {
	repoRoot := t.TempDir()
	refsPath := config.RefsPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(refsPath), 0o755); err != nil {
		t.Fatalf("creating refs dir: %v", err)
	}

	refs := mixedGroupFixture()
	if err := storage.WriteAll(refsPath, refs); err != nil {
		t.Fatalf("writing refs: %v", err)
	}

	groups := []DuplicateGroup{
		{
			MatchBasis: MatchBasisSourceID,
			SourceType: "paperpile",
			SourceID:   "uuid-1",
			Primary:    "A",
			Duplicates: []string{"A-2"},
		},
		{
			MatchBasis: MatchBasisTitle,
			Title:      "titleonlymatch",
			Members:    []string{"P", "Q"},
			Primary:    "P",
			Duplicates: []string{"Q"}, // must be ignored
		},
	}

	if err := performMerge(repoRoot, refs, groups); err != nil {
		t.Fatalf("performMerge: %v", err)
	}

	got, err := storage.ReadAll(refsPath)
	if err != nil {
		t.Fatalf("reading refs back: %v", err)
	}
	var gotIDs []string
	for _, ref := range got {
		gotIDs = append(gotIDs, ref.ID)
	}
	want := []string{"A", "P", "Q"} // only A-2, the source-ID duplicate, is dropped
	if !slices.Equal(gotIDs, want) {
		t.Errorf("refs after merge = %v, want %v", gotIDs, want)
	}
}
