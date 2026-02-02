package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/project"
	"github.com/matsen/bipartite/internal/repo"
	"github.com/matsen/bipartite/internal/storage"
	"gopkg.in/yaml.v3"
)

// setupTestEnvironment creates a test bipartite repository and sets up global config.
// Returns cleanup function that must be deferred.
func setupTestEnvironment(t *testing.T, tmpDir string) func() {
	t.Helper()

	// Create global config directory
	configDir := filepath.Join(tmpDir, "config", "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create global config with nexus_path (YAML format)
	globalConfig := "nexus_path: " + tmpDir + "\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(globalConfig), 0644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}

	// Set XDG_CONFIG_HOME and save original
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))

	// Reset global config cache to pick up new config
	config.ResetGlobalConfigCache()

	return func() {
		// Restore original XDG_CONFIG_HOME
		if origXDG == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		}
		// Reset cache again
		config.ResetGlobalConfigCache()
	}
}

func TestProjectImportCommand(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Set up test environment with global config
	cleanup := setupTestEnvironment(t, tmpDir)
	defer cleanup()

	// Initialize bipartite structure
	bipDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(bipDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite directory: %v", err)
	}
	cacheDir := filepath.Join(bipDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create empty JSONL files
	files := []string{"refs.jsonl", "concepts.jsonl", "projects.jsonl", "repos.jsonl", "edges.jsonl"}
	for _, f := range files {
		path := filepath.Join(bipDir, f)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", f, err)
		}
	}

	// Create test config file
	configData := map[string]ProjectConfig{
		"test-project": {
			Name:  "Test Project",
			Repos: []string{}, // Empty repos to avoid GitHub API calls
		},
		"another-project": {
			// No name provided - should default to ID
			Repos: []string{},
		},
	}

	configBytes, err := yaml.Marshal(configData)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tmpDir, "test-projects.yml")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Execute import command with dry-run first
	cmd := projectImportCmd
	cmd.Flags().Set("dry-run", "true")
	if err := cmd.RunE(cmd, []string{configPath}); err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Verify no changes were made during dry run
	projectsPath := config.ProjectsPath(tmpDir)
	projects, err := storage.ReadAllProjects(projectsPath)
	if err != nil {
		t.Fatalf("Failed to read projects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Dry run should not create projects, got %d", len(projects))
	}

	// Now run actual import
	cmd.Flags().Set("dry-run", "false")
	if err := cmd.RunE(cmd, []string{configPath}); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify projects were created
	projects, err = storage.ReadAllProjects(projectsPath)
	if err != nil {
		t.Fatalf("Failed to read projects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Verify project details
	projectMap := make(map[string]project.Project)
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	if p, ok := projectMap["test-project"]; !ok {
		t.Error("test-project not found")
	} else if p.Name != "Test Project" {
		t.Errorf("Expected name 'Test Project', got %q", p.Name)
	}

	if p, ok := projectMap["another-project"]; !ok {
		t.Error("another-project not found")
	} else if p.Name != "another-project" {
		t.Errorf("Expected name to default to ID 'another-project', got %q", p.Name)
	}

	// Test idempotence - run import again
	if err := cmd.RunE(cmd, []string{configPath}); err != nil {
		t.Fatalf("Second import failed: %v", err)
	}

	// Verify no duplicates were created
	projects, err = storage.ReadAllProjects(projectsPath)
	if err != nil {
		t.Fatalf("Failed to read projects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects after second import, got %d", len(projects))
	}
}

func TestProjectImportWithRepos(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Set up test environment with global config
	cleanup := setupTestEnvironment(t, tmpDir)
	defer cleanup()

	// Initialize bipartite structure
	bipDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(bipDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite directory: %v", err)
	}
	cacheDir := filepath.Join(bipDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create empty JSONL files
	files := []string{"refs.jsonl", "concepts.jsonl", "projects.jsonl", "repos.jsonl", "edges.jsonl"}
	for _, f := range files {
		path := filepath.Join(bipDir, f)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", f, err)
		}
	}

	// Create test config file with repos
	configData := map[string]ProjectConfig{
		"test-project": {
			Name:  "Test Project",
			Repos: []string{"matsen/bipartite", "matsen/netam"},
		},
	}

	configBytes, err := yaml.Marshal(configData)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tmpDir, "test-projects.yml")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Execute import with --no-fetch to avoid GitHub API calls
	cmd := projectImportCmd
	cmd.Flags().Set("no-fetch", "true")
	cmd.Flags().Set("dry-run", "false")
	if err := cmd.RunE(cmd, []string{configPath}); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify repos were created
	reposPath := config.ReposPath(tmpDir)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		t.Fatalf("Failed to read repos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(repos))
	}

	// Verify repo details
	repoMap := make(map[string]repo.Repo)
	for _, r := range repos {
		repoMap[r.ID] = r
	}

	if r, ok := repoMap["bipartite"]; !ok {
		t.Error("bipartite repo not found")
	} else {
		if r.Project != "test-project" {
			t.Errorf("Expected project 'test-project', got %q", r.Project)
		}
		if r.GitHubURL != "https://github.com/matsen/bipartite" {
			t.Errorf("Expected normalized URL, got %q", r.GitHubURL)
		}
	}
}

func TestProjectImportWithConcepts(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Set up test environment with global config
	cleanup := setupTestEnvironment(t, tmpDir)
	defer cleanup()

	// Initialize bipartite structure
	bipDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(bipDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite directory: %v", err)
	}
	cacheDir := filepath.Join(bipDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create empty JSONL files
	files := []string{"refs.jsonl", "projects.jsonl", "repos.jsonl", "edges.jsonl"}
	for _, f := range files {
		path := filepath.Join(bipDir, f)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", f, err)
		}
	}

	// Create concepts file with some concepts
	conceptsPath := filepath.Join(bipDir, "concepts.jsonl")
	conceptsData := `{"id":"bcr-phylogenetics","name":"BCR Phylogenetics"}
{"id":"somatic-hypermutation","name":"Somatic Hypermutation"}`
	if err := os.WriteFile(conceptsPath, []byte(conceptsData), 0644); err != nil {
		t.Fatalf("Failed to create concepts file: %v", err)
	}

	// Create test config with concepts
	configData := map[string]ProjectConfig{
		"test-project": {
			Name:     "Test Project",
			Concepts: []string{"bcr-phylogenetics", "somatic-hypermutation", "nonexistent-concept"},
		},
	}

	configBytes, err := yaml.Marshal(configData)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tmpDir, "test-projects.yml")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Execute import with --link-concepts
	cmd := projectImportCmd
	cmd.Flags().Set("link-concepts", "true")
	cmd.Flags().Set("dry-run", "false")
	if err := cmd.RunE(cmd, []string{configPath}); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify edges were created
	edgesPath := filepath.Join(bipDir, "edges.jsonl")
	edgesData, err := os.ReadFile(edgesPath)
	if err != nil {
		t.Fatalf("Failed to read edges: %v", err)
	}

	// Should have 2 edges (2 valid concepts, 1 skipped)
	lines := 0
	for _, b := range edgesData {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("Expected 2 edges (newlines), got %d, content: %q", lines, string(edgesData))
	}
}

func TestProjectImportRepoIDCollision(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Set up test environment with global config
	cleanup := setupTestEnvironment(t, tmpDir)
	defer cleanup()

	// Initialize bipartite structure
	bipDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(bipDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite directory: %v", err)
	}
	cacheDir := filepath.Join(bipDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create empty JSONL files
	files := []string{"refs.jsonl", "concepts.jsonl", "projects.jsonl", "repos.jsonl", "edges.jsonl"}
	for _, f := range files {
		path := filepath.Join(bipDir, f)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", f, err)
		}
	}

	// Create test config with same repo name from different orgs
	configData := map[string]ProjectConfig{
		"project-a": {
			Repos: []string{"org1/utils"},
		},
		"project-b": {
			Repos: []string{"org2/utils"},
		},
	}

	configBytes, err := yaml.Marshal(configData)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configPath := filepath.Join(tmpDir, "test-projects.yml")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Execute import with --no-fetch
	cmd := projectImportCmd
	cmd.Flags().Set("no-fetch", "true")
	cmd.Flags().Set("dry-run", "false")
	if err := cmd.RunE(cmd, []string{configPath}); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify both repos were created with different IDs
	reposPath := config.ReposPath(tmpDir)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		t.Fatalf("Failed to read repos: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(repos))
	}

	// Verify unique IDs
	ids := make(map[string]bool)
	for _, r := range repos {
		if ids[r.ID] {
			t.Errorf("Duplicate repo ID: %s", r.ID)
		}
		ids[r.ID] = true
	}

	// Verify unique URLs
	urls := make(map[string]bool)
	for _, r := range repos {
		if urls[r.GitHubURL] {
			t.Errorf("Duplicate GitHub URL: %s", r.GitHubURL)
		}
		urls[r.GitHubURL] = true
	}
}
