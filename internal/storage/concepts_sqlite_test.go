package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/concept"
)

func TestRebuildConceptsFromJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Use test fixture
	conceptsPath := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")

	count, err := db.RebuildConceptsFromJSONL(conceptsPath)
	if err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v", err)
	}

	if count != 4 {
		t.Errorf("RebuildConceptsFromJSONL() count = %d, want 4", count)
	}
}

func TestRebuildConceptsFromJSONL_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Should return 0 count for nonexistent file (not error)
	count, err := db.RebuildConceptsFromJSONL("/nonexistent/path/concepts.jsonl")
	if err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v, want nil for nonexistent file", err)
	}
	if count != 0 {
		t.Errorf("RebuildConceptsFromJSONL() count = %d, want 0", count)
	}
}

func TestGetConceptByID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Populate with test data
	conceptsPath := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v", err)
	}

	// Test found
	c, err := db.GetConceptByID("somatic-hypermutation")
	if err != nil {
		t.Fatalf("GetConceptByID() error = %v", err)
	}
	if c == nil {
		t.Fatal("GetConceptByID() returned nil, want concept")
	}
	if c.Name != "Somatic Hypermutation" {
		t.Errorf("GetConceptByID() Name = %q, want %q", c.Name, "Somatic Hypermutation")
	}
	if len(c.Aliases) != 1 || c.Aliases[0] != "SHM" {
		t.Errorf("GetConceptByID() Aliases = %v, want [SHM]", c.Aliases)
	}

	// Test not found
	c, err = db.GetConceptByID("nonexistent")
	if err != nil {
		t.Fatalf("GetConceptByID() error = %v", err)
	}
	if c != nil {
		t.Errorf("GetConceptByID() = %v, want nil for nonexistent", c)
	}
}

func TestGetAllConcepts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Populate with test data
	conceptsPath := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v", err)
	}

	concepts, err := db.GetAllConcepts()
	if err != nil {
		t.Fatalf("GetAllConcepts() error = %v", err)
	}

	if len(concepts) != 4 {
		t.Errorf("GetAllConcepts() returned %d concepts, want 4", len(concepts))
	}

	// Verify sorted by ID
	if concepts[0].ID != "bcr-sequencing" {
		t.Errorf("GetAllConcepts() first ID = %q, want %q (sorted)", concepts[0].ID, "bcr-sequencing")
	}
}

func TestSearchConcepts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Populate with test data
	conceptsPath := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v", err)
	}

	// Search by name
	results, err := db.SearchConcepts("Somatic", 10)
	if err != nil {
		t.Fatalf("SearchConcepts() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchConcepts('Somatic') returned %d results, want 1", len(results))
	}

	// Search by alias
	results, err = db.SearchConcepts("SHM", 10)
	if err != nil {
		t.Fatalf("SearchConcepts() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchConcepts('SHM') returned %d results, want 1", len(results))
	}

	// Search by description
	results, err = db.SearchConcepts("Bayesian", 10)
	if err != nil {
		t.Fatalf("SearchConcepts() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchConcepts('Bayesian') returned %d results, want 1", len(results))
	}
}

func TestCountConcepts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Empty initially
	count, err := db.CountConcepts()
	if err != nil {
		t.Fatalf("CountConcepts() error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountConcepts() = %d, want 0 initially", count)
	}

	// Populate with test data
	conceptsPath := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v", err)
	}

	count, err = db.CountConcepts()
	if err != nil {
		t.Fatalf("CountConcepts() error = %v", err)
	}
	if count != 4 {
		t.Errorf("CountConcepts() = %d, want 4", count)
	}
}

func TestGetPapersByConcept(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Create test edges file and rebuild
	edgesPath := filepath.Join(tmpDir, "edges.jsonl")
	testEdges := `{"source_id": "Paper1", "target_id": "test-concept", "relationship_type": "introduces", "summary": "Test 1"}
{"source_id": "Paper2", "target_id": "test-concept", "relationship_type": "applies", "summary": "Test 2"}
{"source_id": "Paper3", "target_id": "other-concept", "relationship_type": "applies", "summary": "Test 3"}
`
	if err := os.WriteFile(edgesPath, []byte(testEdges), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
		t.Fatalf("RebuildEdgesFromJSONL() error = %v", err)
	}

	// Test all papers for concept
	papers, err := db.GetPapersByConcept("test-concept", "")
	if err != nil {
		t.Fatalf("GetPapersByConcept() error = %v", err)
	}
	if len(papers) != 2 {
		t.Errorf("GetPapersByConcept() returned %d papers, want 2", len(papers))
	}

	// Test filter by relationship type
	papers, err = db.GetPapersByConcept("test-concept", "introduces")
	if err != nil {
		t.Fatalf("GetPapersByConcept() error = %v", err)
	}
	if len(papers) != 1 {
		t.Errorf("GetPapersByConcept() with type filter returned %d papers, want 1", len(papers))
	}
}

func TestGetConceptsByPaper(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Create test concepts and edges
	conceptsPath := filepath.Join(tmpDir, "concepts.jsonl")
	testConcepts := []concept.Concept{
		{ID: "concept-a", Name: "Concept A"},
		{ID: "concept-b", Name: "Concept B"},
	}
	if err := WriteAllConcepts(conceptsPath, testConcepts); err != nil {
		t.Fatalf("WriteAllConcepts error = %v", err)
	}
	if _, err := db.RebuildConceptsFromJSONL(conceptsPath); err != nil {
		t.Fatalf("RebuildConceptsFromJSONL() error = %v", err)
	}

	edgesPath := filepath.Join(tmpDir, "edges.jsonl")
	testEdges := `{"source_id": "Paper1", "target_id": "concept-a", "relationship_type": "introduces", "summary": "Test 1"}
{"source_id": "Paper1", "target_id": "concept-b", "relationship_type": "applies", "summary": "Test 2"}
{"source_id": "Paper1", "target_id": "Paper2", "relationship_type": "cites", "summary": "Not a concept edge"}
`
	if err := os.WriteFile(edgesPath, []byte(testEdges), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
		t.Fatalf("RebuildEdgesFromJSONL() error = %v", err)
	}

	// Test all concepts for paper
	concepts, err := db.GetConceptsByPaper("Paper1", "")
	if err != nil {
		t.Fatalf("GetConceptsByPaper() error = %v", err)
	}
	// Should only return 2 concepts (not the paper-paper edge)
	if len(concepts) != 2 {
		t.Errorf("GetConceptsByPaper() returned %d concepts, want 2", len(concepts))
	}

	// Test filter by relationship type
	concepts, err = db.GetConceptsByPaper("Paper1", "introduces")
	if err != nil {
		t.Fatalf("GetConceptsByPaper() error = %v", err)
	}
	if len(concepts) != 1 {
		t.Errorf("GetConceptsByPaper() with type filter returned %d concepts, want 1", len(concepts))
	}
}

func TestCountEdgesByTarget(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	// Create test edges
	edgesPath := filepath.Join(tmpDir, "edges.jsonl")
	testEdges := `{"source_id": "Paper1", "target_id": "test-concept", "relationship_type": "introduces", "summary": "Test 1"}
{"source_id": "Paper2", "target_id": "test-concept", "relationship_type": "applies", "summary": "Test 2"}
{"source_id": "Paper3", "target_id": "other-concept", "relationship_type": "applies", "summary": "Test 3"}
`
	if err := os.WriteFile(edgesPath, []byte(testEdges), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
		t.Fatalf("RebuildEdgesFromJSONL() error = %v", err)
	}

	count, err := db.CountEdgesByTarget("test-concept")
	if err != nil {
		t.Fatalf("CountEdgesByTarget() error = %v", err)
	}
	if count != 2 {
		t.Errorf("CountEdgesByTarget() = %d, want 2", count)
	}

	count, err = db.CountEdgesByTarget("nonexistent")
	if err != nil {
		t.Fatalf("CountEdgesByTarget() error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountEdgesByTarget() for nonexistent = %d, want 0", count)
	}
}
