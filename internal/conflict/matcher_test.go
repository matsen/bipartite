package conflict

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestMatchPapers_ByDOI(t *testing.T) {
	region := ConflictRegion{
		OursRefs: []reference.Reference{
			{ID: "paper1", DOI: "10.1234/a", Title: "Paper One"},
		},
		TheirsRefs: []reference.Reference{
			{ID: "paper1-different-id", DOI: "10.1234/a", Title: "Paper One"},
		},
	}

	result := MatchPapers(region)

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}

	match := result.Matches[0]
	if match.MatchedBy != "doi" {
		t.Errorf("expected match by doi, got %s", match.MatchedBy)
	}
	if match.Ours.ID != "paper1" {
		t.Errorf("expected ours ID paper1, got %s", match.Ours.ID)
	}
	if match.Theirs.ID != "paper1-different-id" {
		t.Errorf("expected theirs ID paper1-different-id, got %s", match.Theirs.ID)
	}

	if len(result.OursOnly) != 0 {
		t.Errorf("expected 0 ours only, got %d", len(result.OursOnly))
	}
	if len(result.TheirsOnly) != 0 {
		t.Errorf("expected 0 theirs only, got %d", len(result.TheirsOnly))
	}
}

func TestMatchPapers_ByID(t *testing.T) {
	region := ConflictRegion{
		OursRefs: []reference.Reference{
			{ID: "paper1", Title: "Paper One"},
		},
		TheirsRefs: []reference.Reference{
			{ID: "paper1", Title: "Paper One Updated"},
		},
	}

	result := MatchPapers(region)

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}

	match := result.Matches[0]
	if match.MatchedBy != "id" {
		t.Errorf("expected match by id, got %s", match.MatchedBy)
	}
}

func TestMatchPapers_NoMatch(t *testing.T) {
	region := ConflictRegion{
		OursRefs: []reference.Reference{
			{ID: "paper1", DOI: "10.1234/a", Title: "Paper One"},
		},
		TheirsRefs: []reference.Reference{
			{ID: "paper2", DOI: "10.1234/b", Title: "Paper Two"},
		},
	}

	result := MatchPapers(region)

	if len(result.Matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result.Matches))
	}
	if len(result.OursOnly) != 1 {
		t.Errorf("expected 1 ours only, got %d", len(result.OursOnly))
	}
	if len(result.TheirsOnly) != 1 {
		t.Errorf("expected 1 theirs only, got %d", len(result.TheirsOnly))
	}

	if result.OursOnly[0].ID != "paper1" {
		t.Errorf("expected ours only paper1, got %s", result.OursOnly[0].ID)
	}
	if result.TheirsOnly[0].ID != "paper2" {
		t.Errorf("expected theirs only paper2, got %s", result.TheirsOnly[0].ID)
	}
}

func TestMatchPapers_MixedMatching(t *testing.T) {
	region := ConflictRegion{
		OursRefs: []reference.Reference{
			{ID: "paper1", DOI: "10.1234/a", Title: "Paper One"},
			{ID: "paper2", Title: "Paper Two"}, // No DOI, matches by ID
			{ID: "paper3", DOI: "10.1234/c", Title: "Paper Three - Only Ours"},
		},
		TheirsRefs: []reference.Reference{
			{ID: "paper1-new", DOI: "10.1234/a", Title: "Paper One"}, // Matches by DOI
			{ID: "paper2", Title: "Paper Two"},                       // Matches by ID
			{ID: "paper4", DOI: "10.1234/d", Title: "Paper Four - Only Theirs"},
		},
	}

	result := MatchPapers(region)

	if len(result.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(result.Matches))
	}

	// Check first match is by DOI
	if result.Matches[0].MatchedBy != "doi" {
		t.Errorf("expected first match by doi, got %s", result.Matches[0].MatchedBy)
	}
	// Check second match is by ID
	if result.Matches[1].MatchedBy != "id" {
		t.Errorf("expected second match by id, got %s", result.Matches[1].MatchedBy)
	}

	if len(result.OursOnly) != 1 {
		t.Errorf("expected 1 ours only, got %d", len(result.OursOnly))
	}
	if len(result.TheirsOnly) != 1 {
		t.Errorf("expected 1 theirs only, got %d", len(result.TheirsOnly))
	}

	if result.OursOnly[0].ID != "paper3" {
		t.Errorf("expected ours only paper3, got %s", result.OursOnly[0].ID)
	}
	if result.TheirsOnly[0].ID != "paper4" {
		t.Errorf("expected theirs only paper4, got %s", result.TheirsOnly[0].ID)
	}
}

func TestMatchPapers_DOIPrioritizedOverID(t *testing.T) {
	// Paper with same DOI but different IDs should match by DOI
	region := ConflictRegion{
		OursRefs: []reference.Reference{
			{ID: "smith2024-a", DOI: "10.1234/a", Title: "Smith et al."},
		},
		TheirsRefs: []reference.Reference{
			{ID: "Smith2024-xyz", DOI: "10.1234/a", Title: "Smith et al."},
		},
	}

	result := MatchPapers(region)

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}
	if result.Matches[0].MatchedBy != "doi" {
		t.Errorf("expected match by doi, got %s", result.Matches[0].MatchedBy)
	}
}

func TestMatchPapers_EmptyRegion(t *testing.T) {
	region := ConflictRegion{
		OursRefs:   []reference.Reference{},
		TheirsRefs: []reference.Reference{},
	}

	result := MatchPapers(region)

	if len(result.Matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result.Matches))
	}
	if len(result.OursOnly) != 0 {
		t.Errorf("expected 0 ours only, got %d", len(result.OursOnly))
	}
	if len(result.TheirsOnly) != 0 {
		t.Errorf("expected 0 theirs only, got %d", len(result.TheirsOnly))
	}
}

func TestMatchPapers_FromTestFixture(t *testing.T) {
	// Parse a test fixture and match papers
	content := readTestFixture(t, "multiple_papers.jsonl")
	parseResult, err := ParseString(content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	conflict := parseResult.Conflicts[0]
	matchResult := MatchPapers(conflict)

	// paper2 matches by DOI, paper4 only in ours, paper5 only in theirs
	if len(matchResult.Matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matchResult.Matches))
	}
	if len(matchResult.OursOnly) != 1 {
		t.Errorf("expected 1 ours only, got %d", len(matchResult.OursOnly))
	}
	if len(matchResult.TheirsOnly) != 1 {
		t.Errorf("expected 1 theirs only, got %d", len(matchResult.TheirsOnly))
	}

	// Verify the matched paper
	if matchResult.Matches[0].Ours.DOI != "10.1234/b" {
		t.Errorf("expected matched paper DOI 10.1234/b, got %s", matchResult.Matches[0].Ours.DOI)
	}

	// Verify ours only paper
	if matchResult.OursOnly[0].ID != "paper4" {
		t.Errorf("expected ours only paper4, got %s", matchResult.OursOnly[0].ID)
	}

	// Verify theirs only paper
	if matchResult.TheirsOnly[0].ID != "paper5" {
		t.Errorf("expected theirs only paper5, got %s", matchResult.TheirsOnly[0].ID)
	}
}
