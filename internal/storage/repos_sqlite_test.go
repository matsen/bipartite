package storage

import (
	"path/filepath"
	"testing"
)

func TestRebuildReposFromJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildReposFromJSONL("../../testdata/repos/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildReposFromJSONL() error = %v", err)
	}
	if count != 4 {
		t.Errorf("RebuildReposFromJSONL() count = %d, want 4", count)
	}

	// Verify count
	dbCount, err := db.CountRepos()
	if err != nil {
		t.Fatalf("CountRepos() error = %v", err)
	}
	if dbCount != 4 {
		t.Errorf("CountRepos() = %d, want 4", dbCount)
	}
}

func TestGetRepoByID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	_, err = db.RebuildReposFromJSONL("../../testdata/repos/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildReposFromJSONL() error = %v", err)
	}

	// Get existing repo
	r, err := db.GetRepoByID("dasm2-code")
	if err != nil {
		t.Fatalf("GetRepoByID() error = %v", err)
	}
	if r == nil {
		t.Fatal("GetRepoByID() returned nil")
	}
	if r.ID != "dasm2-code" {
		t.Errorf("r.ID = %q, want %q", r.ID, "dasm2-code")
	}
	if r.Project != "dasm2" {
		t.Errorf("r.Project = %q, want %q", r.Project, "dasm2")
	}
	if r.Type != "github" {
		t.Errorf("r.Type = %q, want %q", r.Type, "github")
	}
	if len(r.Topics) != 2 {
		t.Errorf("len(r.Topics) = %d, want 2", len(r.Topics))
	}

	// Get non-existing repo
	r, err = db.GetRepoByID("nonexistent")
	if err != nil {
		t.Fatalf("GetRepoByID() error = %v", err)
	}
	if r != nil {
		t.Errorf("GetRepoByID() = %v, want nil", r)
	}
}

func TestGetAllRepos(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	_, err = db.RebuildReposFromJSONL("../../testdata/repos/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildReposFromJSONL() error = %v", err)
	}

	repos, err := db.GetAllRepos()
	if err != nil {
		t.Fatalf("GetAllRepos() error = %v", err)
	}
	if len(repos) != 4 {
		t.Errorf("GetAllRepos() len = %d, want 4", len(repos))
	}
}

func TestDBGetReposByProject(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	_, err = db.RebuildReposFromJSONL("../../testdata/repos/valid.jsonl")
	if err != nil {
		t.Fatalf("RebuildReposFromJSONL() error = %v", err)
	}

	// Get repos for dasm2 (should have 3: dasm2-code, dasm2-paper, internal-tools)
	repos, err := db.GetReposByProject("dasm2")
	if err != nil {
		t.Fatalf("GetReposByProject() error = %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("GetReposByProject('dasm2') len = %d, want 3", len(repos))
	}

	// Get repos for bipartite (should have 1)
	repos, err = db.GetReposByProject("bipartite")
	if err != nil {
		t.Fatalf("GetReposByProject() error = %v", err)
	}
	if len(repos) != 1 {
		t.Errorf("GetReposByProject('bipartite') len = %d, want 1", len(repos))
	}

	// Get repos for non-existing project
	repos, err = db.GetReposByProject("nonexistent")
	if err != nil {
		t.Fatalf("GetReposByProject() error = %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("GetReposByProject('nonexistent') len = %d, want 0", len(repos))
	}
}

func TestRebuildReposFromJSONL_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildReposFromJSONL("../../testdata/repos/empty.jsonl")
	if err != nil {
		t.Fatalf("RebuildReposFromJSONL() error = %v", err)
	}
	if count != 0 {
		t.Errorf("RebuildReposFromJSONL() count = %d, want 0", count)
	}
}

func TestRebuildReposFromJSONL_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	count, err := db.RebuildReposFromJSONL("../../testdata/repos/nonexistent.jsonl")
	if err != nil {
		t.Fatalf("RebuildReposFromJSONL() error = %v", err)
	}
	if count != 0 {
		t.Errorf("RebuildReposFromJSONL() count = %d, want 0", count)
	}
}
