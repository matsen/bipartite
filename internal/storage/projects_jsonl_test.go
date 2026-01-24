package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/project"
)

func TestReadAllProjects(t *testing.T) {
	// Test reading valid projects
	projects, err := ReadAllProjects("../../testdata/projects/valid.jsonl")
	if err != nil {
		t.Fatalf("ReadAllProjects() error = %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("ReadAllProjects() got %d projects, want 3", len(projects))
	}

	// Verify first project
	if projects[0].ID != "dasm2" {
		t.Errorf("projects[0].ID = %q, want %q", projects[0].ID, "dasm2")
	}
	if projects[0].Name != "DASM2" {
		t.Errorf("projects[0].Name = %q, want %q", projects[0].Name, "DASM2")
	}
}

func TestReadAllProjects_Empty(t *testing.T) {
	projects, err := ReadAllProjects("../../testdata/projects/empty.jsonl")
	if err != nil {
		t.Fatalf("ReadAllProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("ReadAllProjects() got %d projects, want 0", len(projects))
	}
}

func TestReadAllProjects_NotFound(t *testing.T) {
	projects, err := ReadAllProjects("../../testdata/projects/nonexistent.jsonl")
	if err != nil {
		t.Fatalf("ReadAllProjects() error = %v", err)
	}
	if projects != nil {
		t.Errorf("ReadAllProjects() got %v, want nil", projects)
	}
}

func TestReadAllProjects_InvalidID(t *testing.T) {
	_, err := ReadAllProjects("../../testdata/projects/invalid_id.jsonl")
	if err == nil {
		t.Error("ReadAllProjects() expected error for invalid ID, got nil")
	}
}

func TestWriteAllProjects(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "projects.jsonl")

	projects := []project.Project{
		{ID: "test1", Name: "Test 1", Description: "First test"},
		{ID: "test2", Name: "Test 2"},
	}

	err := WriteAllProjects(path, projects)
	if err != nil {
		t.Fatalf("WriteAllProjects() error = %v", err)
	}

	// Read back and verify
	read, err := ReadAllProjects(path)
	if err != nil {
		t.Fatalf("ReadAllProjects() error = %v", err)
	}
	if len(read) != 2 {
		t.Errorf("ReadAllProjects() got %d projects, want 2", len(read))
	}
	if read[0].ID != "test1" {
		t.Errorf("read[0].ID = %q, want %q", read[0].ID, "test1")
	}
}

func TestAppendProject(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "projects.jsonl")

	// Append first project
	p1 := project.Project{ID: "test1", Name: "Test 1"}
	err := AppendProject(path, p1)
	if err != nil {
		t.Fatalf("AppendProject() error = %v", err)
	}

	// Append second project
	p2 := project.Project{ID: "test2", Name: "Test 2"}
	err = AppendProject(path, p2)
	if err != nil {
		t.Fatalf("AppendProject() error = %v", err)
	}

	// Read and verify
	projects, err := ReadAllProjects(path)
	if err != nil {
		t.Fatalf("ReadAllProjects() error = %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("got %d projects, want 2", len(projects))
	}
}

func TestFindProjectByID(t *testing.T) {
	projects := []project.Project{
		{ID: "first", Name: "First"},
		{ID: "second", Name: "Second"},
		{ID: "third", Name: "Third"},
	}

	// Find existing
	idx, found := FindProjectByID(projects, "second")
	if !found {
		t.Error("FindProjectByID() found = false, want true")
	}
	if idx != 1 {
		t.Errorf("FindProjectByID() idx = %d, want 1", idx)
	}

	// Find non-existing
	idx, found = FindProjectByID(projects, "fourth")
	if found {
		t.Error("FindProjectByID() found = true, want false")
	}
	if idx != -1 {
		t.Errorf("FindProjectByID() idx = %d, want -1", idx)
	}
}

func TestDeleteProjectFromSlice(t *testing.T) {
	projects := []project.Project{
		{ID: "first", Name: "First"},
		{ID: "second", Name: "Second"},
		{ID: "third", Name: "Third"},
	}

	// Delete existing
	result, deleted := DeleteProjectFromSlice(projects, "second")
	if !deleted {
		t.Error("DeleteProjectFromSlice() deleted = false, want true")
	}
	if len(result) != 2 {
		t.Errorf("DeleteProjectFromSlice() len = %d, want 2", len(result))
	}

	// Verify "second" is gone
	_, found := FindProjectByID(result, "second")
	if found {
		t.Error("second should be deleted")
	}

	// Delete non-existing
	result, deleted = DeleteProjectFromSlice(result, "nonexistent")
	if deleted {
		t.Error("DeleteProjectFromSlice() deleted = true, want false")
	}
}

func TestLoadProjectIDSet(t *testing.T) {
	idSet, err := LoadProjectIDSet("../../testdata/projects/valid.jsonl")
	if err != nil {
		t.Fatalf("LoadProjectIDSet() error = %v", err)
	}

	if len(idSet) != 3 {
		t.Errorf("LoadProjectIDSet() len = %d, want 3", len(idSet))
	}

	if !idSet["dasm2"] {
		t.Error("idSet should contain 'dasm2'")
	}
	if !idSet["phylo-review"] {
		t.Error("idSet should contain 'phylo-review'")
	}
	if !idSet["bipartite"] {
		t.Error("idSet should contain 'bipartite'")
	}
}

func TestLoadProjectIDSet_NotFound(t *testing.T) {
	idSet, err := LoadProjectIDSet("nonexistent.jsonl")
	if err != nil {
		t.Fatalf("LoadProjectIDSet() error = %v", err)
	}
	if len(idSet) != 0 {
		t.Errorf("LoadProjectIDSet() len = %d, want 0", len(idSet))
	}
}

func init() {
	// Ensure we're running from the correct directory
	if _, err := os.Stat("../../testdata/projects/valid.jsonl"); os.IsNotExist(err) {
		// Try to find the testdata directory
		wd, _ := os.Getwd()
		t := &testing.T{}
		t.Logf("Working directory: %s", wd)
	}
}
