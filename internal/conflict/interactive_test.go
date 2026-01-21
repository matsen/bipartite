package conflict

import (
	"testing"

	"github.com/matsen/bipartite/internal/reference"
)

func TestInteractiveNeeded(t *testing.T) {
	// Parse the interactive_needed fixture
	content := readTestFixture(t, "interactive_needed.jsonl")
	result, err := ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	// Match papers
	matchResult := MatchPapers(result.Conflicts[0])
	if len(matchResult.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matchResult.Matches))
	}

	// Check that both matches need interactive resolution
	unresolvedCount := 0
	for _, match := range matchResult.Matches {
		plan := Resolve(match)
		if plan.Action == ActionConflict {
			unresolvedCount++
		}
	}

	// Paper2 has abstract conflict, paper3 has title and abstract conflict
	if unresolvedCount != 2 {
		t.Errorf("expected 2 unresolved conflicts, got %d", unresolvedCount)
	}
}

func TestInteractiveFieldCounts(t *testing.T) {
	// Paper 2 has venue conflict (Nature vs Science) and abstract conflict
	match := PaperMatch{
		Ours: reference.Reference{
			ID:       "paper2",
			DOI:      "10.1234/b",
			Title:    "Paper Two",
			Abstract: "Version A abstract.",
			Venue:    "Nature",
		},
		Theirs: reference.Reference{
			ID:       "paper2",
			DOI:      "10.1234/b",
			Title:    "Paper Two",
			Abstract: "Version B abstract.",
			Venue:    "Science",
		},
		MatchedBy: "doi",
	}

	plan := Resolve(match)
	if plan.Action != ActionConflict {
		t.Errorf("expected ActionConflict, got %s", plan.Action)
	}

	// Should have 2 conflicts: abstract and venue
	if len(plan.Conflicts) != 2 {
		t.Errorf("expected 2 conflicts, got %d", len(plan.Conflicts))
	}

	// Verify field names
	fieldNames := make(map[string]bool)
	for _, fc := range plan.Conflicts {
		fieldNames[fc.FieldName] = true
	}

	if !fieldNames["abstract"] {
		t.Error("expected conflict on abstract field")
	}
	if !fieldNames["venue"] {
		t.Error("expected conflict on venue field")
	}
}

func TestInteractiveProgressCount(t *testing.T) {
	// Verify that multiple conflicts can be tracked for progress indication
	content := readTestFixture(t, "interactive_needed.jsonl")
	result, err := ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	matchResult := MatchPapers(result.Conflicts[0])

	// Count total conflicts across all matches
	totalConflicts := 0
	for _, match := range matchResult.Matches {
		plan := Resolve(match)
		totalConflicts += len(plan.Conflicts)
	}

	// Paper2 has 2 conflicts (abstract, venue), Paper3 has 2 conflicts (title, abstract)
	if totalConflicts < 2 {
		t.Errorf("expected at least 2 total field conflicts, got %d", totalConflicts)
	}
}

func TestAutoResolveBeforeInteractive(t *testing.T) {
	// When using interactive mode, auto-resolvable conflicts should be handled automatically
	// True conflicts should require prompts

	// Create a mix of auto-resolvable and true conflicts
	autoMatch := PaperMatch{
		Ours: reference.Reference{
			ID:       "auto-paper",
			DOI:      "10.1234/auto",
			Title:    "Auto Paper",
			Abstract: "Full abstract here.",
		},
		Theirs: reference.Reference{
			ID:    "auto-paper",
			DOI:   "10.1234/auto",
			Title: "Auto Paper",
		},
		MatchedBy: "doi",
	}

	trueMatch := PaperMatch{
		Ours: reference.Reference{
			ID:       "true-paper",
			DOI:      "10.1234/true",
			Title:    "True Paper",
			Abstract: "Abstract A",
		},
		Theirs: reference.Reference{
			ID:       "true-paper",
			DOI:      "10.1234/true",
			Title:    "True Paper",
			Abstract: "Abstract B",
		},
		MatchedBy: "doi",
	}

	autoPlan := Resolve(autoMatch)
	truePlan := Resolve(trueMatch)

	if autoPlan.Action == ActionConflict {
		t.Error("auto-resolvable match should not require interactive")
	}

	if truePlan.Action != ActionConflict {
		t.Error("true conflict match should require interactive")
	}
}
