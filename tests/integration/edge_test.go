// Package integration provides integration tests for bipartite commands.
package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var (
	bpBinary     string
	bpBinaryOnce sync.Once
	bpBinaryErr  error
)

// getBPBinary builds the bp binary once and returns its path.
func getBPBinary(t *testing.T) string {
	t.Helper()
	bpBinaryOnce.Do(func() {
		// Get module root directory
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			bpBinaryErr = os.ErrInvalid
			return
		}
		moduleRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))

		// Build bp to a temp location
		tmpDir, err := os.MkdirTemp("", "bp-test-*")
		if err != nil {
			bpBinaryErr = err
			return
		}
		bpBinary = filepath.Join(tmpDir, "bp")

		cmd := exec.Command("go", "build", "-o", bpBinary, "./cmd/bip")
		cmd.Dir = moduleRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			bpBinaryErr = &buildError{output: string(output), err: err}
			return
		}
	})
	if bpBinaryErr != nil {
		t.Fatalf("failed to build bp: %v", bpBinaryErr)
	}
	return bpBinary
}

type buildError struct {
	output string
	err    error
}

func (e *buildError) Error() string {
	return e.err.Error() + ": " + e.output
}

// setupTestRepo creates a minimal bipartite repo with test refs for edge testing.
// Returns the repo directory and a config directory for XDG_CONFIG_HOME.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create .bipartite directory
	bpDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(filepath.Join(bpDir, "cache"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create minimal config
	configContent := "pdf_root: \"\"\npdf_reader: system\n"
	if err := os.WriteFile(filepath.Join(bpDir, "config.yml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create refs.jsonl with test papers
	refsContent := `{"id":"PaperA","title":"Paper A","authors":[{"last":"A"}],"published":{"year":2024},"source":{"type":"manual"}}
{"id":"PaperB","title":"Paper B","authors":[{"last":"B"}],"published":{"year":2024},"source":{"type":"manual"}}
{"id":"PaperC","title":"Paper C","authors":[{"last":"C"}],"published":{"year":2024},"source":{"type":"manual"}}
`
	if err := os.WriteFile(filepath.Join(bpDir, "refs.jsonl"), []byte(refsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create global config directory with nexus_path pointing to test repo
	configDir := filepath.Join(tmpDir, "config", "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := "nexus_path: " + tmpDir + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

// runBP executes the bp command with given args and returns output.
// Uses XDG_CONFIG_HOME to point to test-specific global config with nexus_path.
func runBP(t *testing.T, repoDir string, args ...string) (string, error) {
	t.Helper()
	bp := getBPBinary(t)
	cmd := exec.Command(bp, args...)
	cmd.Dir = repoDir
	// Set XDG_CONFIG_HOME to the test config directory (parent of bip/)
	configHome := filepath.Join(repoDir, "config")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configHome)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestEdgeAdd(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Test adding an edge
	output, err := runBP(t, repoDir, "edge", "add",
		"--source", "PaperA",
		"--target", "PaperB",
		"--type", "cites",
		"--summary", "Paper A cites Paper B")
	if err != nil {
		t.Fatalf("edge add failed: %v\nOutput: %s", err, output)
	}

	// Verify JSON output
	output, err = runBP(t, repoDir, "edge", "add",
		"--source", "PaperA",
		"--target", "PaperC",
		"--type", "extends",
		"--summary", "Paper A extends Paper C")
	if err != nil {
		t.Fatalf("edge add (second) failed: %v\nOutput: %s", err, output)
	}

	// Check JSON output format
	var result struct {
		Action string `json:"action"`
		Edge   struct {
			SourceID         string `json:"source_id"`
			TargetID         string `json:"target_id"`
			RelationshipType string `json:"relationship_type"`
		} `json:"edge"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}
	if result.Action != "added" {
		t.Errorf("expected action 'added', got %q", result.Action)
	}
	if result.Edge.SourceID != "PaperA" {
		t.Errorf("expected source_id 'PaperA', got %q", result.Edge.SourceID)
	}
}

func TestEdgeAddMissingPaper(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Test adding edge with non-existent source
	_, err := runBP(t, repoDir, "edge", "add",
		"--source", "NonExistent",
		"--target", "PaperB",
		"--type", "cites",
		"--summary", "Test")
	if err == nil {
		t.Fatal("expected error for non-existent source paper")
	}

	// Test adding edge with non-existent target
	_, err = runBP(t, repoDir, "edge", "add",
		"--source", "PaperA",
		"--target", "NonExistent",
		"--type", "cites",
		"--summary", "Test")
	if err == nil {
		t.Fatal("expected error for non-existent target paper")
	}
}

func TestEdgeAddUpsert(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add initial edge
	_, err := runBP(t, repoDir, "edge", "add",
		"--source", "PaperA",
		"--target", "PaperB",
		"--type", "cites",
		"--summary", "Original summary")
	if err != nil {
		t.Fatalf("initial edge add failed: %v", err)
	}

	// Update same edge
	output, err := runBP(t, repoDir, "edge", "add",
		"--source", "PaperA",
		"--target", "PaperB",
		"--type", "cites",
		"--summary", "Updated summary")
	if err != nil {
		t.Fatalf("upsert edge add failed: %v\nOutput: %s", err, output)
	}

	var result struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Action != "updated" {
		t.Errorf("expected action 'updated', got %q", result.Action)
	}
}

func TestEdgeList(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add some edges
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperC", "-r", "extends", "-m", "A extends C")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperA", "-r", "builds-on", "-m", "B builds on A")

	// Test list outgoing (default)
	output, err := runBP(t, repoDir, "edge", "list", "PaperA")
	if err != nil {
		t.Fatalf("edge list failed: %v\nOutput: %s", err, output)
	}

	var listResult struct {
		PaperID  string `json:"paper_id"`
		Outgoing []struct {
			TargetID string `json:"target_id"`
		} `json:"outgoing"`
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v\nOutput: %s", err, output)
	}
	if len(listResult.Outgoing) != 2 {
		t.Errorf("expected 2 outgoing edges, got %d", len(listResult.Outgoing))
	}

	// Test list incoming
	output, err = runBP(t, repoDir, "edge", "list", "PaperA", "--incoming")
	if err != nil {
		t.Fatalf("edge list --incoming failed: %v", err)
	}

	var incomingResult struct {
		Incoming []struct {
			SourceID string `json:"source_id"`
		} `json:"incoming"`
	}
	if err := json.Unmarshal([]byte(output), &incomingResult); err != nil {
		t.Fatalf("failed to parse incoming output: %v", err)
	}
	if len(incomingResult.Incoming) != 1 {
		t.Errorf("expected 1 incoming edge, got %d", len(incomingResult.Incoming))
	}

	// Test list --all
	output, err = runBP(t, repoDir, "edge", "list", "PaperA", "--all")
	if err != nil {
		t.Fatalf("edge list --all failed: %v", err)
	}

	var allResult struct {
		Outgoing []interface{} `json:"outgoing"`
		Incoming []interface{} `json:"incoming"`
	}
	if err := json.Unmarshal([]byte(output), &allResult); err != nil {
		t.Fatalf("failed to parse --all output: %v", err)
	}
	if len(allResult.Outgoing) != 2 || len(allResult.Incoming) != 1 {
		t.Errorf("expected 2 outgoing and 1 incoming, got %d and %d",
			len(allResult.Outgoing), len(allResult.Incoming))
	}
}

func TestEdgeSearch(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add edges with different types
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperC", "-r", "extends", "-m", "A extends C")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperC", "-r", "cites", "-m", "B cites C")

	// Search by type
	output, err := runBP(t, repoDir, "edge", "search", "--type", "cites")
	if err != nil {
		t.Fatalf("edge search failed: %v\nOutput: %s", err, output)
	}

	var searchResult struct {
		RelationshipType string `json:"relationship_type"`
		Edges            []struct {
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(output), &searchResult); err != nil {
		t.Fatalf("failed to parse search output: %v", err)
	}
	if len(searchResult.Edges) != 2 {
		t.Errorf("expected 2 'cites' edges, got %d", len(searchResult.Edges))
	}
	if searchResult.RelationshipType != "cites" {
		t.Errorf("expected relationship_type 'cites', got %q", searchResult.RelationshipType)
	}

	// Search for non-existent type
	output, err = runBP(t, repoDir, "edge", "search", "--type", "nonexistent")
	if err != nil {
		t.Fatalf("edge search (empty) failed: %v", err)
	}
	if err := json.Unmarshal([]byte(output), &searchResult); err != nil {
		t.Fatalf("failed to parse empty search output: %v", err)
	}
	if len(searchResult.Edges) != 0 {
		t.Errorf("expected 0 edges for non-existent type, got %d", len(searchResult.Edges))
	}
}

func TestEdgeExport(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add edges
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperC", "-r", "extends", "-m", "A extends C")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperC", "-r", "cites", "-m", "B cites C")

	// Export all edges
	output, err := runBP(t, repoDir, "edge", "export")
	if err != nil {
		t.Fatalf("edge export failed: %v\nOutput: %s", err, output)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 exported edges, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var edge struct {
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
		}
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i+1, err)
		}
	}

	// Export edges for specific paper
	output, err = runBP(t, repoDir, "edge", "export", "--paper", "PaperA")
	if err != nil {
		t.Fatalf("edge export --paper failed: %v", err)
	}

	lines = strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 edges involving PaperA, got %d", len(lines))
	}
}

func TestEdgeImport(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create import file
	importContent := `{"source_id":"PaperA","target_id":"PaperB","relationship_type":"cites","summary":"Imported: A cites B"}
{"source_id":"PaperB","target_id":"PaperC","relationship_type":"extends","summary":"Imported: B extends C"}
{"source_id":"PaperA","target_id":"NonExistent","relationship_type":"cites","summary":"Should be skipped"}
`
	importPath := filepath.Join(repoDir, "import.jsonl")
	if err := os.WriteFile(importPath, []byte(importContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Import edges
	output, err := runBP(t, repoDir, "edge", "import", importPath)
	if err != nil {
		t.Fatalf("edge import failed: %v\nOutput: %s", err, output)
	}

	var importResult struct {
		Added   int `json:"added"`
		Updated int `json:"updated"`
		Skipped int `json:"skipped"`
		Errors  []struct {
			Line  int    `json:"line"`
			Error string `json:"error"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(output), &importResult); err != nil {
		t.Fatalf("failed to parse import output: %v\nOutput: %s", err, output)
	}

	if importResult.Added != 2 {
		t.Errorf("expected 2 added, got %d", importResult.Added)
	}
	if importResult.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", importResult.Skipped)
	}
	if len(importResult.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(importResult.Errors))
	}
}

func TestEdgeExportImportRoundTrip(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add edges
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperC", "-r", "extends", "-m", "B extends C")

	// Export to file
	exportOutput, err := runBP(t, repoDir, "edge", "export")
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	exportPath := filepath.Join(repoDir, "exported.jsonl")
	if err := os.WriteFile(exportPath, []byte(exportOutput), 0644); err != nil {
		t.Fatal(err)
	}

	// Clear edges by removing edges.jsonl
	edgesPath := filepath.Join(repoDir, ".bipartite", "edges.jsonl")
	if err := os.Remove(edgesPath); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	// Re-import
	output, err := runBP(t, repoDir, "edge", "import", exportPath)
	if err != nil {
		t.Fatalf("re-import failed: %v\nOutput: %s", err, output)
	}

	var importResult struct {
		Added int `json:"added"`
	}
	if err := json.Unmarshal([]byte(output), &importResult); err != nil {
		t.Fatalf("failed to parse import output: %v", err)
	}
	if importResult.Added != 2 {
		t.Errorf("expected 2 edges re-imported, got %d", importResult.Added)
	}

	// Verify edges are back
	listOutput, err := runBP(t, repoDir, "edge", "export")
	if err != nil {
		t.Fatalf("final export failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(listOutput), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 edges after round-trip, got %d", len(lines))
	}
}

func TestRebuildWithEdges(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add some edges first
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperC", "-r", "extends", "-m", "B extends C")

	// Delete the database
	dbPath := filepath.Join(repoDir, ".bipartite", "cache", "refs.db")
	os.Remove(dbPath)

	// Rebuild
	output, err := runBP(t, repoDir, "rebuild")
	if err != nil {
		t.Fatalf("rebuild failed: %v\nOutput: %s", err, output)
	}

	var result struct {
		Status     string `json:"status"`
		References int    `json:"references"`
		Edges      int    `json:"edges"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse rebuild output: %v", err)
	}

	if result.Status != "rebuilt" {
		t.Errorf("expected status 'rebuilt', got %q", result.Status)
	}
	if result.References != 3 {
		t.Errorf("expected 3 references, got %d", result.References)
	}
	if result.Edges != 2 {
		t.Errorf("expected 2 edges, got %d", result.Edges)
	}

	// Verify edges are still queryable
	listOutput, err := runBP(t, repoDir, "edge", "list", "PaperA")
	if err != nil {
		t.Fatalf("edge list after rebuild failed: %v", err)
	}

	var listResult struct {
		Outgoing []interface{} `json:"outgoing"`
	}
	if err := json.Unmarshal([]byte(listOutput), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v", err)
	}
	if len(listResult.Outgoing) != 1 {
		t.Errorf("expected 1 outgoing edge after rebuild, got %d", len(listResult.Outgoing))
	}
}

func TestGroomWithOrphanedEdges(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add edges
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")

	// Manually add an orphaned edge to edges.jsonl
	edgesPath := filepath.Join(repoDir, ".bipartite", "edges.jsonl")
	f, err := os.OpenFile(edgesPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"source_id":"PaperA","target_id":"NonExistent","relationship_type":"cites","summary":"Orphaned"}` + "\n")
	f.Close()

	// Run groom
	output, err := runBP(t, repoDir, "groom")
	if err != nil {
		t.Fatalf("groom failed: %v\nOutput: %s", err, output)
	}

	var groomResult struct {
		Status        string `json:"status"`
		OrphanedEdges []struct {
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
			Reason   string `json:"reason"`
		} `json:"orphaned_edges"`
		Fixed bool `json:"fixed"`
	}
	if err := json.Unmarshal([]byte(output), &groomResult); err != nil {
		t.Fatalf("failed to parse groom output: %v\nOutput: %s", err, output)
	}

	if groomResult.Status != "orphaned" {
		t.Errorf("expected status 'orphaned', got %q", groomResult.Status)
	}
	if len(groomResult.OrphanedEdges) != 1 {
		t.Errorf("expected 1 orphaned edge, got %d", len(groomResult.OrphanedEdges))
	}
	if groomResult.Fixed {
		t.Error("expected fixed=false without --fix flag")
	}

	// Run groom with --fix
	output, err = runBP(t, repoDir, "groom", "--fix")
	if err != nil {
		t.Fatalf("groom --fix failed: %v\nOutput: %s", err, output)
	}

	if err := json.Unmarshal([]byte(output), &groomResult); err != nil {
		t.Fatalf("failed to parse groom --fix output: %v", err)
	}

	if groomResult.Status != "fixed" {
		t.Errorf("expected status 'fixed', got %q", groomResult.Status)
	}
	if !groomResult.Fixed {
		t.Error("expected fixed=true with --fix flag")
	}

	// Verify orphaned edge is removed
	output, err = runBP(t, repoDir, "groom")
	if err != nil {
		t.Fatalf("groom (after fix) failed: %v", err)
	}
	if err := json.Unmarshal([]byte(output), &groomResult); err != nil {
		t.Fatalf("failed to parse final groom output: %v", err)
	}
	if groomResult.Status != "clean" {
		t.Errorf("expected status 'clean' after fix, got %q", groomResult.Status)
	}
}

func TestCheckWithEdges(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add valid edges
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperC", "-r", "extends", "-m", "B extends C")

	// Run check
	output, err := runBP(t, repoDir, "check")
	if err != nil {
		t.Fatalf("check failed: %v\nOutput: %s", err, output)
	}

	var checkResult struct {
		Status     string `json:"status"`
		References int    `json:"references"`
		Edges      int    `json:"edges"`
		Issues     []struct {
			Type string `json:"type"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(output), &checkResult); err != nil {
		t.Fatalf("failed to parse check output: %v\nOutput: %s", err, output)
	}

	if checkResult.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", checkResult.Status)
	}
	if checkResult.References != 3 {
		t.Errorf("expected 3 references, got %d", checkResult.References)
	}
	if checkResult.Edges != 2 {
		t.Errorf("expected 2 edges, got %d", checkResult.Edges)
	}
	if len(checkResult.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(checkResult.Issues))
	}
}

func TestFullEdgeWorkflow(t *testing.T) {
	repoDir := setupTestRepo(t)

	// 1. Add edges
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperB", "-r", "cites", "-m", "A cites B for foundational work")
	runBP(t, repoDir, "edge", "add", "-s", "PaperA", "-t", "PaperC", "-r", "extends", "-m", "A extends C's methodology")
	runBP(t, repoDir, "edge", "add", "-s", "PaperB", "-t", "PaperC", "-r", "contradicts", "-m", "B contradicts C's findings")

	// 2. List edges from PaperA
	output, _ := runBP(t, repoDir, "edge", "list", "PaperA", "--all")
	var listResult struct {
		Outgoing []interface{} `json:"outgoing"`
		Incoming []interface{} `json:"incoming"`
	}
	json.Unmarshal([]byte(output), &listResult)
	if len(listResult.Outgoing) != 2 {
		t.Errorf("expected 2 outgoing from PaperA, got %d", len(listResult.Outgoing))
	}

	// 3. Search by type
	output, _ = runBP(t, repoDir, "edge", "search", "--type", "cites")
	var searchResult struct {
		Edges []interface{} `json:"edges"`
	}
	json.Unmarshal([]byte(output), &searchResult)
	if len(searchResult.Edges) != 1 {
		t.Errorf("expected 1 'cites' edge, got %d", len(searchResult.Edges))
	}

	// 4. Export all edges
	output, _ = runBP(t, repoDir, "edge", "export")
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 exported edges, got %d", len(lines))
	}

	// 5. Check repository
	output, _ = runBP(t, repoDir, "check")
	var checkResult struct {
		Status string `json:"status"`
		Edges  int    `json:"edges"`
	}
	json.Unmarshal([]byte(output), &checkResult)
	if checkResult.Status != "ok" {
		t.Errorf("expected check status 'ok', got %q", checkResult.Status)
	}
	if checkResult.Edges != 3 {
		t.Errorf("expected 3 edges in check, got %d", checkResult.Edges)
	}

	// 6. Rebuild and verify
	dbPath := filepath.Join(repoDir, ".bipartite", "cache", "refs.db")
	os.Remove(dbPath)
	runBP(t, repoDir, "rebuild")

	output, _ = runBP(t, repoDir, "edge", "list", "PaperA")
	json.Unmarshal([]byte(output), &listResult)
	if len(listResult.Outgoing) != 2 {
		t.Errorf("expected 2 outgoing after rebuild, got %d", len(listResult.Outgoing))
	}
}
