// Package integration provides integration tests for bipartite commands.
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// T030: Integration test for project CRUD
func TestProjectCRUD(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Test project add
	output, err := runBP(t, repoDir, "project", "add", "dasm2", "--name", "DASM2", "--description", "Distance-based antibody modeling")
	if err != nil {
		t.Fatalf("project add failed: %v\nOutput: %s", err, output)
	}

	var addResult struct {
		Status  string `json:"status"`
		Project struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"project"`
	}
	if err := json.Unmarshal([]byte(output), &addResult); err != nil {
		t.Fatalf("failed to parse add output: %v\nOutput: %s", err, output)
	}
	if addResult.Status != "created" {
		t.Errorf("expected status 'created', got %q", addResult.Status)
	}
	if addResult.Project.ID != "dasm2" {
		t.Errorf("expected id 'dasm2', got %q", addResult.Project.ID)
	}

	// Test project get
	output, err = runBP(t, repoDir, "project", "get", "dasm2")
	if err != nil {
		t.Fatalf("project get failed: %v\nOutput: %s", err, output)
	}

	var getResult struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(output), &getResult); err != nil {
		t.Fatalf("failed to parse get output: %v", err)
	}
	if getResult.Name != "DASM2" {
		t.Errorf("expected name 'DASM2', got %q", getResult.Name)
	}

	// Test project list
	// Add another project first
	runBP(t, repoDir, "project", "add", "netam", "--name", "NetAM")

	output, err = runBP(t, repoDir, "project", "list")
	if err != nil {
		t.Fatalf("project list failed: %v\nOutput: %s", err, output)
	}

	var listResult struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v", err)
	}
	if listResult.Count != 2 {
		t.Errorf("expected 2 projects, got %d", listResult.Count)
	}

	// Test project update
	output, err = runBP(t, repoDir, "project", "update", "dasm2", "--name", "DASM2 v2", "--description", "Updated desc")
	if err != nil {
		t.Fatalf("project update failed: %v\nOutput: %s", err, output)
	}

	var updateResult struct {
		Status  string `json:"status"`
		Project struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"project"`
	}
	if err := json.Unmarshal([]byte(output), &updateResult); err != nil {
		t.Fatalf("failed to parse update output: %v", err)
	}
	if updateResult.Status != "updated" {
		t.Errorf("expected status 'updated', got %q", updateResult.Status)
	}
	if updateResult.Project.Name != "DASM2 v2" {
		t.Errorf("expected name 'DASM2 v2', got %q", updateResult.Project.Name)
	}

	// Test project delete
	output, err = runBP(t, repoDir, "project", "delete", "netam")
	if err != nil {
		t.Fatalf("project delete failed: %v\nOutput: %s", err, output)
	}

	var deleteResult struct {
		Status string `json:"status"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal([]byte(output), &deleteResult); err != nil {
		t.Fatalf("failed to parse delete output: %v", err)
	}
	if deleteResult.Status != "deleted" {
		t.Errorf("expected status 'deleted', got %q", deleteResult.Status)
	}

	// Verify deletion
	output, err = runBP(t, repoDir, "project", "list")
	if err != nil {
		t.Fatalf("project list (after delete) failed: %v", err)
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v", err)
	}
	if listResult.Count != 1 {
		t.Errorf("expected 1 project after delete, got %d", listResult.Count)
	}
}

// Test project ID collision with papers
func TestProjectIDCollision(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Try to create project with ID that matches a paper
	_, err := runBP(t, repoDir, "project", "add", "PaperA", "--name", "Collision")
	if err == nil {
		t.Fatal("expected error for project ID collision with paper")
	}
}

// T038: Integration test for repo CRUD
func TestRepoCRUD(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create a project first
	runBP(t, repoDir, "project", "add", "dasm2", "--name", "DASM2")

	// Test repo add (manual mode to avoid GitHub API calls in tests)
	output, err := runBP(t, repoDir, "repo", "add", "--manual",
		"--project", "dasm2",
		"--id", "dasm2-code",
		"--name", "DASM2 Code",
		"--description", "Main code repository",
		"--topics", "ml,python")
	if err != nil {
		t.Fatalf("repo add failed: %v\nOutput: %s", err, output)
	}

	var addResult struct {
		Status string `json:"status"`
		Repo   struct {
			ID      string   `json:"id"`
			Project string   `json:"project"`
			Type    string   `json:"type"`
			Name    string   `json:"name"`
			Topics  []string `json:"topics"`
		} `json:"repo"`
	}
	if err := json.Unmarshal([]byte(output), &addResult); err != nil {
		t.Fatalf("failed to parse add output: %v\nOutput: %s", err, output)
	}
	if addResult.Status != "created" {
		t.Errorf("expected status 'created', got %q", addResult.Status)
	}
	if addResult.Repo.Type != "manual" {
		t.Errorf("expected type 'manual', got %q", addResult.Repo.Type)
	}
	if len(addResult.Repo.Topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(addResult.Repo.Topics))
	}

	// Test repo get
	output, err = runBP(t, repoDir, "repo", "get", "dasm2-code")
	if err != nil {
		t.Fatalf("repo get failed: %v\nOutput: %s", err, output)
	}

	var getResult struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(output), &getResult); err != nil {
		t.Fatalf("failed to parse get output: %v", err)
	}
	if getResult.Name != "DASM2 Code" {
		t.Errorf("expected name 'DASM2 Code', got %q", getResult.Name)
	}

	// Test repo list
	// Add another repo
	runBP(t, repoDir, "repo", "add", "--manual",
		"--project", "dasm2",
		"--id", "dasm2-paper",
		"--name", "DASM2 Paper")

	output, err = runBP(t, repoDir, "repo", "list")
	if err != nil {
		t.Fatalf("repo list failed: %v\nOutput: %s", err, output)
	}

	var listResult struct {
		Repos []struct {
			ID string `json:"id"`
		} `json:"repos"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v", err)
	}
	if listResult.Count != 2 {
		t.Errorf("expected 2 repos, got %d", listResult.Count)
	}

	// Test repo list with project filter
	output, err = runBP(t, repoDir, "repo", "list", "--project", "dasm2")
	if err != nil {
		t.Fatalf("repo list --project failed: %v", err)
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse filtered list output: %v", err)
	}
	if listResult.Count != 2 {
		t.Errorf("expected 2 repos for project, got %d", listResult.Count)
	}

	// Test project repos command
	output, err = runBP(t, repoDir, "project", "repos", "dasm2")
	if err != nil {
		t.Fatalf("project repos failed: %v\nOutput: %s", err, output)
	}

	var projectReposResult struct {
		ProjectID string `json:"project_id"`
		Repos     []struct {
			ID string `json:"id"`
		} `json:"repos"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(output), &projectReposResult); err != nil {
		t.Fatalf("failed to parse project repos output: %v", err)
	}
	if projectReposResult.Count != 2 {
		t.Errorf("expected 2 repos, got %d", projectReposResult.Count)
	}

	// Test repo update
	output, err = runBP(t, repoDir, "repo", "update", "dasm2-code",
		"--name", "DASM2 Main Code",
		"--topics", "ml,python,antibodies")
	if err != nil {
		t.Fatalf("repo update failed: %v\nOutput: %s", err, output)
	}

	var updateResult struct {
		Status string `json:"status"`
		Repo   struct {
			Name   string   `json:"name"`
			Topics []string `json:"topics"`
		} `json:"repo"`
	}
	if err := json.Unmarshal([]byte(output), &updateResult); err != nil {
		t.Fatalf("failed to parse update output: %v", err)
	}
	if updateResult.Repo.Name != "DASM2 Main Code" {
		t.Errorf("expected name 'DASM2 Main Code', got %q", updateResult.Repo.Name)
	}
	if len(updateResult.Repo.Topics) != 3 {
		t.Errorf("expected 3 topics after update, got %d", len(updateResult.Repo.Topics))
	}

	// Test repo delete
	output, err = runBP(t, repoDir, "repo", "delete", "dasm2-paper")
	if err != nil {
		t.Fatalf("repo delete failed: %v\nOutput: %s", err, output)
	}

	var deleteResult struct {
		Status string `json:"status"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal([]byte(output), &deleteResult); err != nil {
		t.Fatalf("failed to parse delete output: %v", err)
	}
	if deleteResult.Status != "deleted" {
		t.Errorf("expected status 'deleted', got %q", deleteResult.Status)
	}

	// Verify deletion
	output, err = runBP(t, repoDir, "repo", "list")
	if err != nil {
		t.Fatalf("repo list (after delete) failed: %v", err)
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v", err)
	}
	if listResult.Count != 1 {
		t.Errorf("expected 1 repo after delete, got %d", listResult.Count)
	}
}

// T069: Integration test for rebuild with projects/repos
func TestRebuildWithProjectsRepos(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add projects and repos
	runBP(t, repoDir, "project", "add", "dasm2", "--name", "DASM2")
	runBP(t, repoDir, "repo", "add", "--manual",
		"--project", "dasm2",
		"--id", "dasm2-code",
		"--name", "DASM2 Code")

	// Delete the database
	dbPath := filepath.Join(repoDir, ".bipartite", "cache", "refs.db")
	os.Remove(dbPath)

	// Rebuild
	output, err := runBP(t, repoDir, "rebuild")
	if err != nil {
		t.Fatalf("rebuild failed: %v\nOutput: %s", err, output)
	}

	var result struct {
		Status   string `json:"status"`
		Projects int    `json:"projects"`
		Repos    int    `json:"repos"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse rebuild output: %v\nOutput: %s", err, output)
	}

	if result.Status != "rebuilt" {
		t.Errorf("expected status 'rebuilt', got %q", result.Status)
	}
	if result.Projects != 1 {
		t.Errorf("expected 1 project, got %d", result.Projects)
	}
	if result.Repos != 1 {
		t.Errorf("expected 1 repo, got %d", result.Repos)
	}

	// Verify data is still queryable
	output, err = runBP(t, repoDir, "project", "repos", "dasm2")
	if err != nil {
		t.Fatalf("project repos after rebuild failed: %v", err)
	}

	var reposResult struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(output), &reposResult); err != nil {
		t.Fatalf("failed to parse repos output: %v", err)
	}
	if reposResult.Count != 1 {
		t.Errorf("expected 1 repo after rebuild, got %d", reposResult.Count)
	}
}

// T070: Integration test for check with project/repo constraints
func TestCheckWithProjectsRepos(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Add valid project and repo
	runBP(t, repoDir, "project", "add", "dasm2", "--name", "DASM2")
	runBP(t, repoDir, "repo", "add", "--manual",
		"--project", "dasm2",
		"--id", "dasm2-code",
		"--name", "DASM2 Code")

	// Run check
	output, err := runBP(t, repoDir, "check")
	if err != nil {
		t.Fatalf("check failed: %v\nOutput: %s", err, output)
	}

	var checkResult struct {
		Status   string `json:"status"`
		Projects int    `json:"projects"`
		Repos    int    `json:"repos"`
		Issues   []struct {
			Type string `json:"type"`
		} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(output), &checkResult); err != nil {
		t.Fatalf("failed to parse check output: %v\nOutput: %s", err, output)
	}

	if checkResult.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", checkResult.Status)
	}
	if checkResult.Projects != 1 {
		t.Errorf("expected 1 project, got %d", checkResult.Projects)
	}
	if checkResult.Repos != 1 {
		t.Errorf("expected 1 repo, got %d", checkResult.Repos)
	}
	if len(checkResult.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(checkResult.Issues))
	}
}

// Test project delete with cascade
func TestProjectDeleteCascade(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create project and repo
	runBP(t, repoDir, "project", "add", "dasm2", "--name", "DASM2")
	runBP(t, repoDir, "repo", "add", "--manual",
		"--project", "dasm2",
		"--id", "dasm2-code",
		"--name", "DASM2 Code")

	// Try to delete without force (should fail)
	_, err := runBP(t, repoDir, "project", "delete", "dasm2")
	if err == nil {
		t.Fatal("expected error when deleting project with repos without --force")
	}

	// Delete with force
	output, err := runBP(t, repoDir, "project", "delete", "dasm2", "--force")
	if err != nil {
		t.Fatalf("project delete --force failed: %v\nOutput: %s", err, output)
	}

	var deleteResult struct {
		Status       string `json:"status"`
		ReposRemoved int    `json:"repos_removed"`
	}
	if err := json.Unmarshal([]byte(output), &deleteResult); err != nil {
		t.Fatalf("failed to parse delete output: %v", err)
	}

	if deleteResult.Status != "deleted" {
		t.Errorf("expected status 'deleted', got %q", deleteResult.Status)
	}
	if deleteResult.ReposRemoved != 1 {
		t.Errorf("expected 1 repo removed, got %d", deleteResult.ReposRemoved)
	}

	// Verify repos are gone
	output, err = runBP(t, repoDir, "repo", "list")
	if err != nil {
		t.Fatalf("repo list failed: %v", err)
	}
	var listResult struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(output), &listResult); err != nil {
		t.Fatalf("failed to parse list output: %v", err)
	}
	if listResult.Count != 0 {
		t.Errorf("expected 0 repos after cascade delete, got %d", listResult.Count)
	}
}

// T065/T066: Test repo refresh (manual repo should fail)
func TestRepoRefreshManual(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create a project and manual repo
	runBP(t, repoDir, "project", "add", "dasm2", "--name", "DASM2")
	runBP(t, repoDir, "repo", "add", "--manual",
		"--project", "dasm2",
		"--id", "dasm2-code",
		"--name", "DASM2 Code")

	// T066: Try to refresh manual repo (should fail)
	_, err := runBP(t, repoDir, "repo", "refresh", "dasm2-code")
	if err == nil {
		t.Fatal("expected error when refreshing manual repo")
	}
}
