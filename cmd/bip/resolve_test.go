package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/conflict"
)

func TestResolveCmd_NoConflicts(t *testing.T) {
	// Create a temp directory with a valid bip repo structure
	tmpDir := t.TempDir()
	bipDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(bipDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	// Create config.json
	configContent := `{"pdf_root": ""}`
	if err := os.WriteFile(filepath.Join(bipDir, "config.json"), []byte(configContent), 0644); err != nil {
		t.Fatalf("writing config.json: %v", err)
	}

	// Create refs.jsonl without conflicts
	refsContent := `{"id":"paper1","doi":"10.1234/a","title":"Paper One"}
{"id":"paper2","doi":"10.1234/b","title":"Paper Two"}
`
	if err := os.WriteFile(filepath.Join(bipDir, "refs.jsonl"), []byte(refsContent), 0644); err != nil {
		t.Fatalf("writing refs.jsonl: %v", err)
	}

	// Parse the file
	result, err := conflict.ParseString(refsContent)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if result.HasConflicts() {
		t.Error("expected no conflicts")
	}
}

func TestResolveCmd_SimpleOursBetter(t *testing.T) {
	content := readTestFixture(t, "simple_ours_better.jsonl")
	result, err := conflict.ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	// Match papers in the conflict region
	matchResult := conflict.MatchPapers(result.Conflicts[0])
	if len(matchResult.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchResult.Matches))
	}

	// Resolve the match
	plan := conflict.Resolve(matchResult.Matches[0])
	if plan.Action != conflict.ActionKeepOurs {
		t.Errorf("expected ActionKeepOurs, got %s", plan.Action)
	}
}

func TestResolveCmd_SimpleTheirsBetter(t *testing.T) {
	content := readTestFixture(t, "simple_theirs_better.jsonl")
	result, err := conflict.ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	matchResult := conflict.MatchPapers(result.Conflicts[0])
	if len(matchResult.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchResult.Matches))
	}

	plan := conflict.Resolve(matchResult.Matches[0])
	if plan.Action != conflict.ActionKeepTheirs {
		t.Errorf("expected ActionKeepTheirs, got %s", plan.Action)
	}
}

func TestResolveCmd_ComplementaryMerge(t *testing.T) {
	content := readTestFixture(t, "complementary_merge.jsonl")
	result, err := conflict.ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	matchResult := conflict.MatchPapers(result.Conflicts[0])
	if len(matchResult.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchResult.Matches))
	}

	plan := conflict.Resolve(matchResult.Matches[0])
	if plan.Action != conflict.ActionMerge {
		t.Errorf("expected ActionMerge, got %s", plan.Action)
	}

	// Verify merge produces combined result
	merged := conflict.ApplyResolution(matchResult.Matches[0], plan)
	if merged.Abstract == "" {
		t.Error("expected merged abstract from ours")
	}
	if merged.Venue == "" {
		t.Error("expected merged venue from theirs")
	}
}

func TestResolveCmd_TrueConflict(t *testing.T) {
	content := readTestFixture(t, "true_conflict.jsonl")
	result, err := conflict.ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	matchResult := conflict.MatchPapers(result.Conflicts[0])
	if len(matchResult.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchResult.Matches))
	}

	plan := conflict.Resolve(matchResult.Matches[0])
	if plan.Action != conflict.ActionConflict {
		t.Errorf("expected ActionConflict, got %s", plan.Action)
	}
	if len(plan.Conflicts) == 0 {
		t.Error("expected at least one field conflict")
	}
}

func TestResolveCmd_MultiplePapers(t *testing.T) {
	content := readTestFixture(t, "multiple_papers.jsonl")
	result, err := conflict.ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	matchResult := conflict.MatchPapers(result.Conflicts[0])

	// Should have 1 match (paper2), 1 ours only (paper4), 1 theirs only (paper5)
	if len(matchResult.Matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matchResult.Matches))
	}
	if len(matchResult.OursOnly) != 1 {
		t.Errorf("expected 1 ours only, got %d", len(matchResult.OursOnly))
	}
	if len(matchResult.TheirsOnly) != 1 {
		t.Errorf("expected 1 theirs only, got %d", len(matchResult.TheirsOnly))
	}

	// Verify the ours only paper
	if matchResult.OursOnly[0].ID != "paper4" {
		t.Errorf("expected ours only paper4, got %s", matchResult.OursOnly[0].ID)
	}

	// Verify the theirs only paper
	if matchResult.TheirsOnly[0].ID != "paper5" {
		t.Errorf("expected theirs only paper5, got %s", matchResult.TheirsOnly[0].ID)
	}
}

func TestResolveCmd_DryRunOutputFields(t *testing.T) {
	content := readTestFixture(t, "simple_ours_better.jsonl")
	result, err := conflict.ParseString(content)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	// Simulate dry-run processing
	var operations []ResolveOp
	for _, region := range result.Conflicts {
		matchResult := conflict.MatchPapers(region)
		for _, match := range matchResult.Matches {
			plan := conflict.Resolve(match)
			operations = append(operations, ResolveOp{
				PaperID: plan.PaperID,
				DOI:     plan.DOI,
				Action:  string(plan.Action),
				Reason:  plan.Reason,
			})
		}
	}

	if len(operations) == 0 {
		t.Error("expected at least one operation")
	}

	op := operations[0]
	if op.PaperID == "" {
		t.Error("expected operation to have paper ID")
	}
	if op.Action == "" {
		t.Error("expected operation to have action")
	}
	if op.Reason == "" {
		t.Error("expected operation to have reason")
	}
}

func readTestFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "conflict", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return string(data)
}

func TestResolveCmd_MalformedMarkers(t *testing.T) {
	content := readTestFixture(t, "malformed_markers.jsonl")
	_, err := conflict.ParseString(content)
	if err == nil {
		t.Fatal("expected error for malformed markers")
	}

	parseErr, ok := err.(conflict.ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T", err)
	}

	// Should be an unterminated conflict
	if !strings.Contains(parseErr.Message, "unterminated") {
		t.Errorf("expected unterminated error, got: %s", parseErr.Message)
	}
}
