package storage

import (
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/repo"
)

func TestReadAllRepos(t *testing.T) {
	// Test reading valid repos
	repos, err := ReadAllRepos("../../testdata/repos/valid.jsonl")
	if err != nil {
		t.Fatalf("ReadAllRepos() error = %v", err)
	}
	if len(repos) != 4 {
		t.Errorf("ReadAllRepos() got %d repos, want 4", len(repos))
	}

	// Verify first repo
	if repos[0].ID != "dasm2-code" {
		t.Errorf("repos[0].ID = %q, want %q", repos[0].ID, "dasm2-code")
	}
	if repos[0].Project != "dasm2" {
		t.Errorf("repos[0].Project = %q, want %q", repos[0].Project, "dasm2")
	}
	if repos[0].Type != "github" {
		t.Errorf("repos[0].Type = %q, want %q", repos[0].Type, "github")
	}
}

func TestReadAllRepos_Empty(t *testing.T) {
	repos, err := ReadAllRepos("../../testdata/repos/empty.jsonl")
	if err != nil {
		t.Fatalf("ReadAllRepos() error = %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("ReadAllRepos() got %d repos, want 0", len(repos))
	}
}

func TestReadAllRepos_NotFound(t *testing.T) {
	repos, err := ReadAllRepos("../../testdata/repos/nonexistent.jsonl")
	if err != nil {
		t.Fatalf("ReadAllRepos() error = %v", err)
	}
	if repos != nil {
		t.Errorf("ReadAllRepos() got %v, want nil", repos)
	}
}

func TestWriteAllRepos(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "repos.jsonl")

	repos := []repo.Repo{
		{ID: "test1", Project: "proj1", Type: "github", Name: "Test 1", GitHubURL: "https://github.com/test/test1"},
		{ID: "test2", Project: "proj1", Type: "manual", Name: "Test 2"},
	}

	err := WriteAllRepos(path, repos)
	if err != nil {
		t.Fatalf("WriteAllRepos() error = %v", err)
	}

	// Read back and verify
	read, err := ReadAllRepos(path)
	if err != nil {
		t.Fatalf("ReadAllRepos() error = %v", err)
	}
	if len(read) != 2 {
		t.Errorf("ReadAllRepos() got %d repos, want 2", len(read))
	}
	if read[0].ID != "test1" {
		t.Errorf("read[0].ID = %q, want %q", read[0].ID, "test1")
	}
}

func TestAppendRepo(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "repos.jsonl")

	// Append first repo
	r1 := repo.Repo{ID: "test1", Project: "proj1", Type: "manual", Name: "Test 1"}
	err := AppendRepo(path, r1)
	if err != nil {
		t.Fatalf("AppendRepo() error = %v", err)
	}

	// Append second repo
	r2 := repo.Repo{ID: "test2", Project: "proj1", Type: "manual", Name: "Test 2"}
	err = AppendRepo(path, r2)
	if err != nil {
		t.Fatalf("AppendRepo() error = %v", err)
	}

	// Read and verify
	repos, err := ReadAllRepos(path)
	if err != nil {
		t.Fatalf("ReadAllRepos() error = %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2", len(repos))
	}
}

func TestFindRepoByID(t *testing.T) {
	repos := []repo.Repo{
		{ID: "first", Project: "proj1", Type: "manual", Name: "First"},
		{ID: "second", Project: "proj1", Type: "manual", Name: "Second"},
		{ID: "third", Project: "proj2", Type: "manual", Name: "Third"},
	}

	// Find existing
	idx, found := FindRepoByID(repos, "second")
	if !found {
		t.Error("FindRepoByID() found = false, want true")
	}
	if idx != 1 {
		t.Errorf("FindRepoByID() idx = %d, want 1", idx)
	}

	// Find non-existing
	idx, found = FindRepoByID(repos, "fourth")
	if found {
		t.Error("FindRepoByID() found = true, want false")
	}
	if idx != -1 {
		t.Errorf("FindRepoByID() idx = %d, want -1", idx)
	}
}

func TestFindRepoByGitHubURL(t *testing.T) {
	repos := []repo.Repo{
		{ID: "first", Project: "proj1", Type: "github", Name: "First", GitHubURL: "https://github.com/test/first"},
		{ID: "second", Project: "proj1", Type: "manual", Name: "Second"},
		{ID: "third", Project: "proj2", Type: "github", Name: "Third", GitHubURL: "https://github.com/test/third"},
	}

	// Find existing
	idx, found := FindRepoByGitHubURL(repos, "https://github.com/test/third")
	if !found {
		t.Error("FindRepoByGitHubURL() found = false, want true")
	}
	if idx != 2 {
		t.Errorf("FindRepoByGitHubURL() idx = %d, want 2", idx)
	}

	// Find non-existing
	idx, found = FindRepoByGitHubURL(repos, "https://github.com/test/fourth")
	if found {
		t.Error("FindRepoByGitHubURL() found = true, want false")
	}

	// Empty URL
	idx, found = FindRepoByGitHubURL(repos, "")
	if found {
		t.Error("FindRepoByGitHubURL() found = true for empty URL, want false")
	}
}

func TestDeleteRepoFromSlice(t *testing.T) {
	repos := []repo.Repo{
		{ID: "first", Project: "proj1", Type: "manual", Name: "First"},
		{ID: "second", Project: "proj1", Type: "manual", Name: "Second"},
		{ID: "third", Project: "proj2", Type: "manual", Name: "Third"},
	}

	// Delete existing
	result, deleted := DeleteRepoFromSlice(repos, "second")
	if !deleted {
		t.Error("DeleteRepoFromSlice() deleted = false, want true")
	}
	if len(result) != 2 {
		t.Errorf("DeleteRepoFromSlice() len = %d, want 2", len(result))
	}

	// Verify "second" is gone
	_, found := FindRepoByID(result, "second")
	if found {
		t.Error("second should be deleted")
	}

	// Delete non-existing
	result, deleted = DeleteRepoFromSlice(result, "nonexistent")
	if deleted {
		t.Error("DeleteRepoFromSlice() deleted = true, want false")
	}
}

func TestGetReposByProject(t *testing.T) {
	repos := []repo.Repo{
		{ID: "first", Project: "proj1", Type: "manual", Name: "First"},
		{ID: "second", Project: "proj1", Type: "manual", Name: "Second"},
		{ID: "third", Project: "proj2", Type: "manual", Name: "Third"},
	}

	// Get repos for proj1
	proj1Repos := GetReposByProject(repos, "proj1")
	if len(proj1Repos) != 2 {
		t.Errorf("GetReposByProject() len = %d, want 2", len(proj1Repos))
	}

	// Get repos for proj2
	proj2Repos := GetReposByProject(repos, "proj2")
	if len(proj2Repos) != 1 {
		t.Errorf("GetReposByProject() len = %d, want 1", len(proj2Repos))
	}

	// Get repos for non-existing project
	noRepos := GetReposByProject(repos, "proj3")
	if len(noRepos) != 0 {
		t.Errorf("GetReposByProject() len = %d, want 0", len(noRepos))
	}
}

func TestLoadRepoIDSet(t *testing.T) {
	idSet, err := LoadRepoIDSet("../../testdata/repos/valid.jsonl")
	if err != nil {
		t.Fatalf("LoadRepoIDSet() error = %v", err)
	}

	if len(idSet) != 4 {
		t.Errorf("LoadRepoIDSet() len = %d, want 4", len(idSet))
	}

	if !idSet["dasm2-code"] {
		t.Error("idSet should contain 'dasm2-code'")
	}
	if !idSet["bipartite-code"] {
		t.Error("idSet should contain 'bipartite-code'")
	}
}
