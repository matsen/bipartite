package storage

import (
	"path/filepath"
	"testing"
)

func TestRebuildProjectsFromJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildProjectsFromJSONL("../../testdata/projects/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildProjectsFromJSONL() error = %v", err)
	}
	if count != 3 {
		t.Errorf("RebuildProjectsFromJSONL() count = %d, want 3", count)
	}

	// Verify count
	dbCount, err := db.CountProjects()
	if err != nil {
		t.Fatalf("CountProjects() error = %v", err)
	}
	if dbCount != 3 {
		t.Errorf("CountProjects() = %d, want 3", dbCount)
	}
}

func TestGetProjectByID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	_, err = db.RebuildProjectsFromJSONL("../../testdata/projects/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildProjectsFromJSONL() error = %v", err)
	}

	// Get existing project
	p, err := db.GetProjectByID("dasm2")
	if err != nil {
		t.Fatalf("GetProjectByID() error = %v", err)
	}
	if p == nil {
		t.Fatal("GetProjectByID() returned nil")
	}
	if p.ID != "dasm2" {
		t.Errorf("p.ID = %q, want %q", p.ID, "dasm2")
	}
	if p.Name != "DASM2" {
		t.Errorf("p.Name = %q, want %q", p.Name, "DASM2")
	}

	// Get non-existing project
	p, err = db.GetProjectByID("nonexistent")
	if err != nil {
		t.Fatalf("GetProjectByID() error = %v", err)
	}
	if p != nil {
		t.Errorf("GetProjectByID() = %v, want nil", p)
	}
}

func TestGetAllProjects(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	_, err = db.RebuildProjectsFromJSONL("../../testdata/projects/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildProjectsFromJSONL() error = %v", err)
	}

	projects, err := db.GetAllProjects()
	if err != nil {
		t.Fatalf("GetAllProjects() error = %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("GetAllProjects() len = %d, want 3", len(projects))
	}

	// Verify ordering (should be alphabetical by ID)
	if projects[0].ID != "bipartite" {
		t.Errorf("projects[0].ID = %q, want %q", projects[0].ID, "bipartite")
	}
}

func TestRebuildProjectsFromJSONL_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildProjectsFromJSONL("../../testdata/projects/empty.jsonl")
	if err != nil {
		t.Fatalf("RebuildProjectsFromJSONL() error = %v", err)
	}
	if count != 0 {
		t.Errorf("RebuildProjectsFromJSONL() count = %d, want 0", count)
	}
}

func TestRebuildProjectsFromJSONL_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildProjectsFromJSONL("../../testdata/projects/nonexistent.jsonl")
	if err != nil {
		t.Fatalf("RebuildProjectsFromJSONL() error = %v", err)
	}
	if count != 0 {
		t.Errorf("RebuildProjectsFromJSONL() count = %d, want 0", count)
	}
}
