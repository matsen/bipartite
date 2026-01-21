package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/concept"
)

func TestReadAllConcepts(t *testing.T) {
	// Read test fixture
	path := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")
	concepts, err := ReadAllConcepts(path)
	if err != nil {
		t.Fatalf("ReadAllConcepts() error = %v", err)
	}

	if len(concepts) != 4 {
		t.Errorf("ReadAllConcepts() returned %d concepts, want 4", len(concepts))
	}

	// Verify first concept
	if concepts[0].ID != "somatic-hypermutation" {
		t.Errorf("First concept ID = %q, want %q", concepts[0].ID, "somatic-hypermutation")
	}
	if concepts[0].Name != "Somatic Hypermutation" {
		t.Errorf("First concept Name = %q, want %q", concepts[0].Name, "Somatic Hypermutation")
	}
}

func TestReadAllConcepts_NonexistentFile(t *testing.T) {
	concepts, err := ReadAllConcepts("/nonexistent/path/concepts.jsonl")
	if err != nil {
		t.Fatalf("ReadAllConcepts() error = %v, want nil for nonexistent file", err)
	}
	if concepts != nil {
		t.Errorf("ReadAllConcepts() = %v, want nil", concepts)
	}
}

func TestWriteAndReadConcepts(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "concepts.jsonl")

	// Write concepts
	testConcepts := []concept.Concept{
		{ID: "test-concept", Name: "Test Concept", Aliases: []string{"TC"}, Description: "A test"},
		{ID: "another-concept", Name: "Another Concept"},
	}

	err := WriteAllConcepts(path, testConcepts)
	if err != nil {
		t.Fatalf("WriteAllConcepts() error = %v", err)
	}

	// Read back
	readConcepts, err := ReadAllConcepts(path)
	if err != nil {
		t.Fatalf("ReadAllConcepts() error = %v", err)
	}

	if len(readConcepts) != 2 {
		t.Errorf("ReadAllConcepts() returned %d concepts, want 2", len(readConcepts))
	}

	// Verify data integrity
	if readConcepts[0].ID != "test-concept" {
		t.Errorf("First concept ID = %q, want %q", readConcepts[0].ID, "test-concept")
	}
	if len(readConcepts[0].Aliases) != 1 || readConcepts[0].Aliases[0] != "TC" {
		t.Errorf("First concept Aliases = %v, want [TC]", readConcepts[0].Aliases)
	}
}

func TestAppendConcept(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "concepts.jsonl")

	// Append first concept
	c1 := concept.Concept{ID: "first", Name: "First"}
	if err := AppendConcept(path, c1); err != nil {
		t.Fatalf("AppendConcept() error = %v", err)
	}

	// Append second concept
	c2 := concept.Concept{ID: "second", Name: "Second"}
	if err := AppendConcept(path, c2); err != nil {
		t.Fatalf("AppendConcept() error = %v", err)
	}

	// Read and verify
	concepts, err := ReadAllConcepts(path)
	if err != nil {
		t.Fatalf("ReadAllConcepts() error = %v", err)
	}

	if len(concepts) != 2 {
		t.Errorf("ReadAllConcepts() returned %d concepts, want 2", len(concepts))
	}
}

func TestFindConceptByID(t *testing.T) {
	concepts := []concept.Concept{
		{ID: "first", Name: "First"},
		{ID: "second", Name: "Second"},
		{ID: "third", Name: "Third"},
	}

	tests := []struct {
		id        string
		wantIdx   int
		wantFound bool
	}{
		{"first", 0, true},
		{"second", 1, true},
		{"third", 2, true},
		{"nonexistent", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			idx, found := FindConceptByID(concepts, tt.id)
			if idx != tt.wantIdx || found != tt.wantFound {
				t.Errorf("FindConceptByID(%q) = (%d, %v), want (%d, %v)",
					tt.id, idx, found, tt.wantIdx, tt.wantFound)
			}
		})
	}
}

func TestUpsertConceptInSlice(t *testing.T) {
	concepts := []concept.Concept{
		{ID: "existing", Name: "Original Name"},
	}

	// Test update
	updated := concept.Concept{ID: "existing", Name: "Updated Name"}
	result, wasUpdated := UpsertConceptInSlice(concepts, updated)
	if !wasUpdated {
		t.Error("UpsertConceptInSlice() wasUpdated = false, want true")
	}
	if result[0].Name != "Updated Name" {
		t.Errorf("UpsertConceptInSlice() Name = %q, want %q", result[0].Name, "Updated Name")
	}

	// Test insert
	newConcept := concept.Concept{ID: "new", Name: "New Concept"}
	result, wasUpdated = UpsertConceptInSlice(result, newConcept)
	if wasUpdated {
		t.Error("UpsertConceptInSlice() wasUpdated = true, want false")
	}
	if len(result) != 2 {
		t.Errorf("UpsertConceptInSlice() len = %d, want 2", len(result))
	}
}

func TestDeleteConceptFromSlice(t *testing.T) {
	concepts := []concept.Concept{
		{ID: "first", Name: "First"},
		{ID: "second", Name: "Second"},
		{ID: "third", Name: "Third"},
	}

	// Delete middle element
	result, deleted := DeleteConceptFromSlice(concepts, "second")
	if !deleted {
		t.Error("DeleteConceptFromSlice() deleted = false, want true")
	}
	if len(result) != 2 {
		t.Errorf("DeleteConceptFromSlice() len = %d, want 2", len(result))
	}

	// Verify "second" is gone
	_, found := FindConceptByID(result, "second")
	if found {
		t.Error("DeleteConceptFromSlice() 'second' still found in result")
	}

	// Delete nonexistent
	result, deleted = DeleteConceptFromSlice(result, "nonexistent")
	if deleted {
		t.Error("DeleteConceptFromSlice() deleted = true for nonexistent, want false")
	}
}

func TestLoadConceptIDSet(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "concepts", "test-concepts.jsonl")
	idSet, err := LoadConceptIDSet(path)
	if err != nil {
		t.Fatalf("LoadConceptIDSet() error = %v", err)
	}

	expectedIDs := []string{"somatic-hypermutation", "variational-inference", "phylogenetics", "bcr-sequencing"}
	for _, id := range expectedIDs {
		if !idSet[id] {
			t.Errorf("LoadConceptIDSet() missing ID %q", id)
		}
	}

	if idSet["nonexistent"] {
		t.Error("LoadConceptIDSet() found nonexistent ID")
	}
}

func TestReadAllConcepts_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "concepts.jsonl")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte(`{"id": "test", "name": "Test"}\ninvalid json`), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := ReadAllConcepts(path)
	if err == nil {
		t.Error("ReadAllConcepts() error = nil, want error for invalid JSON")
	}
}

func TestReadAllConcepts_InvalidConcept(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "concepts.jsonl")

	// Write concept with invalid ID (uppercase)
	if err := os.WriteFile(path, []byte(`{"id": "INVALID", "name": "Test"}`+"\n"), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := ReadAllConcepts(path)
	if err == nil {
		t.Error("ReadAllConcepts() error = nil, want error for invalid concept")
	}
}
