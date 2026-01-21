package conflict

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_NoConflicts(t *testing.T) {
	content := readTestFixture(t, "no_conflicts.jsonl")
	result, err := ParseString(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasConflicts() {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}

	if len(result.CleanLines) != 3 {
		t.Errorf("expected 3 clean lines, got %d", len(result.CleanLines))
	}
}

func TestParse_SimpleConflict(t *testing.T) {
	content := readTestFixture(t, "simple_ours_better.jsonl")
	result, err := ParseString(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasConflicts() {
		t.Fatal("expected conflicts")
	}

	if len(result.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(result.Conflicts))
	}

	conflict := result.Conflicts[0]
	if len(conflict.OursRefs) != 1 {
		t.Errorf("expected 1 ours ref, got %d", len(conflict.OursRefs))
	}
	if len(conflict.TheirsRefs) != 1 {
		t.Errorf("expected 1 theirs ref, got %d", len(conflict.TheirsRefs))
	}

	// Verify ours has abstract
	if conflict.OursRefs[0].Abstract == "" {
		t.Error("expected ours to have abstract")
	}
	// Verify theirs does NOT have abstract
	if conflict.TheirsRefs[0].Abstract != "" {
		t.Errorf("expected theirs to NOT have abstract, got %q", conflict.TheirsRefs[0].Abstract)
	}

	// Verify clean lines (2 papers outside conflict)
	if len(result.CleanLines) != 2 {
		t.Errorf("expected 2 clean lines, got %d", len(result.CleanLines))
	}
}

func TestParse_MultiplePapers(t *testing.T) {
	content := readTestFixture(t, "multiple_papers.jsonl")
	result, err := ParseString(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(result.Conflicts))
	}

	conflict := result.Conflicts[0]
	// Ours has paper2 and paper4
	if len(conflict.OursRefs) != 2 {
		t.Errorf("expected 2 ours refs, got %d", len(conflict.OursRefs))
	}
	// Theirs has paper2 and paper5
	if len(conflict.TheirsRefs) != 2 {
		t.Errorf("expected 2 theirs refs, got %d", len(conflict.TheirsRefs))
	}
}

func TestParse_MalformedMarkers(t *testing.T) {
	content := readTestFixture(t, "malformed_markers.jsonl")
	_, err := ParseString(content)
	if err == nil {
		t.Fatal("expected error for malformed markers")
	}

	parseErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}

	if parseErr.Message != "unterminated conflict region at end of file" {
		t.Errorf("unexpected error message: %s", parseErr.Message)
	}
}

func TestParse_UnexpectedSeparator(t *testing.T) {
	content := `{"id":"paper1","doi":"10.1234/a","title":"Paper One"}
=======
{"id":"paper2","doi":"10.1234/b","title":"Paper Two"}`

	_, err := ParseString(content)
	if err == nil {
		t.Fatal("expected error for unexpected separator")
	}

	parseErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}

	if parseErr.Line != 2 {
		t.Errorf("expected error on line 2, got line %d", parseErr.Line)
	}
}

func TestParse_UnexpectedEndMarker(t *testing.T) {
	content := `{"id":"paper1","doi":"10.1234/a","title":"Paper One"}
>>>>>>> feature
{"id":"paper2","doi":"10.1234/b","title":"Paper Two"}`

	_, err := ParseString(content)
	if err == nil {
		t.Fatal("expected error for unexpected end marker")
	}

	parseErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}

	if parseErr.Line != 2 {
		t.Errorf("expected error on line 2, got line %d", parseErr.Line)
	}
}

func TestParse_NestedConflictMarkers(t *testing.T) {
	content := `{"id":"paper1","doi":"10.1234/a","title":"Paper One"}
<<<<<<< HEAD
{"id":"paper2","doi":"10.1234/b","title":"Paper Two"}
<<<<<<< feature
{"id":"paper3","doi":"10.1234/c","title":"Paper Three"}
=======
>>>>>>> another`

	_, err := ParseString(content)
	if err == nil {
		t.Fatal("expected error for nested conflict markers")
	}

	parseErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}

	if parseErr.Line != 4 {
		t.Errorf("expected error on line 4, got line %d", parseErr.Line)
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	content := `{"id":"paper1","doi":"10.1234/a","title":"Paper One"}
<<<<<<< HEAD
{invalid json}
=======
{"id":"paper2","doi":"10.1234/b","title":"Paper Two"}
>>>>>>> feature`

	_, err := ParseString(content)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	parseErr, ok := err.(ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T: %v", err, err)
	}

	if parseErr.Line != 3 {
		t.Errorf("expected error on line 3, got line %d", parseErr.Line)
	}
}

func TestParse_LineNumbers(t *testing.T) {
	content := readTestFixture(t, "simple_ours_better.jsonl")
	result, err := ParseString(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	conflict := result.Conflicts[0]
	// <<<<<<< is on line 2
	if conflict.StartLine != 2 {
		t.Errorf("expected conflict start line 2, got %d", conflict.StartLine)
	}
	// >>>>>>> is on line 6
	if conflict.EndLine != 6 {
		t.Errorf("expected conflict end line 6, got %d", conflict.EndLine)
	}
}

func readTestFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "conflict", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return string(data)
}
