package conflict

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestResolve_OursMoreComplete(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Full abstract text.",
			Authors: []reference.Author{
				{First: "Sarah", Last: "Chen"},
			},
			Venue: "Nature",
		},
		Theirs: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	if plan.Action != ActionKeepOurs {
		t.Errorf("expected ActionKeepOurs, got %s", plan.Action)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(plan.Conflicts))
	}
}

func TestResolve_TheirsMoreComplete(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
		},
		Theirs: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Full abstract text.",
			Authors: []reference.Author{
				{First: "Priya", Last: "Patel"},
			},
			Venue: "Science",
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	if plan.Action != ActionKeepTheirs {
		t.Errorf("expected ActionKeepTheirs, got %s", plan.Action)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(plan.Conflicts))
	}
}

func TestResolve_ComplementaryMerge(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Our abstract text.",
		},
		Theirs: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
			Venue: "Science",
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	if plan.Action != ActionMerge {
		t.Errorf("expected ActionMerge, got %s", plan.Action)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(plan.Conflicts))
	}
}

func TestResolve_TrueConflict(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Version A abstract.",
		},
		Theirs: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Version B abstract.",
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	if plan.Action != ActionConflict {
		t.Errorf("expected ActionConflict, got %s", plan.Action)
	}
	if len(plan.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(plan.Conflicts))
	}
	if plan.Conflicts[0].FieldName != "abstract" {
		t.Errorf("expected conflict on abstract, got %s", plan.Conflicts[0].FieldName)
	}
}

func TestResolve_SameValues(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Same abstract.",
			Venue:    "Nature",
		},
		Theirs: reference.Reference{
			ID:       "paper1",
			DOI:      "10.1234/a",
			Title:    "Paper One",
			Abstract: "Same abstract.",
			Venue:    "Nature",
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	// Both are equal, should keep ours by convention
	if plan.Action != ActionKeepOurs {
		t.Errorf("expected ActionKeepOurs for identical papers, got %s", plan.Action)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(plan.Conflicts))
	}
}

func TestResolve_AuthorList_LongerWins(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
			Authors: []reference.Author{
				{First: "Sarah", Last: "Chen"},
			},
		},
		Theirs: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
			Authors: []reference.Author{
				{First: "Sarah", Last: "Chen"},
				{First: "David", Last: "Kim"},
			},
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	if plan.Action != ActionKeepTheirs {
		t.Errorf("expected ActionKeepTheirs (longer author list), got %s", plan.Action)
	}
}

func TestResolve_AuthorList_SameLengthDifferent(t *testing.T) {
	match := PaperMatch{
		Ours: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
			Authors: []reference.Author{
				{First: "Sarah", Last: "Chen"},
			},
		},
		Theirs: reference.Reference{
			ID:    "paper1",
			DOI:   "10.1234/a",
			Title: "Paper One",
			Authors: []reference.Author{
				{First: "Priya", Last: "Patel"},
			},
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)

	if plan.Action != ActionConflict {
		t.Errorf("expected ActionConflict (same length, different authors), got %s", plan.Action)
	}
}

func TestMergeReferences_Complementary(t *testing.T) {
	ours := reference.Reference{
		ID:       "paper1",
		DOI:      "10.1234/a",
		Title:    "Paper One",
		Abstract: "Our abstract.",
		Authors: []reference.Author{
			{First: "Sarah", Last: "Chen"},
		},
	}
	theirs := reference.Reference{
		ID:    "paper1",
		DOI:   "10.1234/a",
		Title: "Paper One",
		Venue: "Science",
		Published: reference.PublicationDate{
			Year:  2024,
			Month: 6,
		},
	}

	merged, conflicts := MergeReferences(ours, theirs)

	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}

	// Check merged has fields from both
	if merged.Abstract != "Our abstract." {
		t.Errorf("expected abstract from ours, got %q", merged.Abstract)
	}
	if merged.Venue != "Science" {
		t.Errorf("expected venue from theirs, got %q", merged.Venue)
	}
	if merged.Published.Year != 2024 {
		t.Errorf("expected year 2024, got %d", merged.Published.Year)
	}
	if len(merged.Authors) != 1 {
		t.Errorf("expected 1 author, got %d", len(merged.Authors))
	}
}

func TestMergeReferences_TitleConflict(t *testing.T) {
	ours := reference.Reference{
		ID:    "paper1",
		DOI:   "10.1234/a",
		Title: "Original Title",
	}
	theirs := reference.Reference{
		ID:    "paper1",
		DOI:   "10.1234/a",
		Title: "Different Title",
	}

	_, conflicts := MergeReferences(ours, theirs)

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].FieldName != "title" {
		t.Errorf("expected title conflict, got %s", conflicts[0].FieldName)
	}
}

func TestMergeReferences_PublicationDate_MoreSpecific(t *testing.T) {
	ours := reference.Reference{
		ID:  "paper1",
		DOI: "10.1234/a",
		Published: reference.PublicationDate{
			Year: 2024,
		},
	}
	theirs := reference.Reference{
		ID:  "paper1",
		DOI: "10.1234/a",
		Published: reference.PublicationDate{
			Year:  2024,
			Month: 6,
			Day:   15,
		},
	}

	merged, conflicts := MergeReferences(ours, theirs)

	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}
	if merged.Published.Year != 2024 {
		t.Errorf("expected year 2024, got %d", merged.Published.Year)
	}
	if merged.Published.Month != 6 {
		t.Errorf("expected month 6, got %d", merged.Published.Month)
	}
	if merged.Published.Day != 15 {
		t.Errorf("expected day 15, got %d", merged.Published.Day)
	}
}

func TestMergeReferences_SupplementPaths_Union(t *testing.T) {
	ours := reference.Reference{
		ID:              "paper1",
		DOI:             "10.1234/a",
		SupplementPaths: []string{"supp1.pdf", "supp2.pdf"},
	}
	theirs := reference.Reference{
		ID:              "paper1",
		DOI:             "10.1234/a",
		SupplementPaths: []string{"supp2.pdf", "supp3.pdf"},
	}

	merged, conflicts := MergeReferences(ours, theirs)

	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(conflicts))
	}
	if len(merged.SupplementPaths) != 3 {
		t.Errorf("expected 3 supplement paths, got %d", len(merged.SupplementPaths))
	}
}

func TestComputeCompleteness(t *testing.T) {
	tests := []struct {
		name     string
		ref      reference.Reference
		expected int
	}{
		{
			name: "empty",
			ref: reference.Reference{
				ID: "paper1",
			},
			expected: 0,
		},
		{
			name: "with abstract",
			ref: reference.Reference{
				ID:       "paper1",
				Abstract: "Some abstract",
			},
			expected: 5, // Abstract has weight 5
		},
		{
			name: "with authors",
			ref: reference.Reference{
				ID: "paper1",
				Authors: []reference.Author{
					{First: "Sarah", Last: "Chen"},
				},
			},
			expected: 4, // Authors has weight 4
		},
		{
			name: "with multiple fields",
			ref: reference.Reference{
				ID:       "paper1",
				Abstract: "Some abstract",
				Venue:    "Nature",
				Authors: []reference.Author{
					{First: "Sarah", Last: "Chen"},
				},
			},
			expected: 12, // Abstract (5) + Authors (4) + Venue (3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ComputeCompleteness(tt.ref)
			if score != tt.expected {
				t.Errorf("expected completeness %d, got %d", tt.expected, score)
			}
		})
	}
}
