package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/github"
	"github.com/matsen/bipartite/internal/project"
	"github.com/matsen/bipartite/internal/repo"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

func init() {
	// project import flags
	projectImportCmd.Flags().Bool("link-concepts", false, "Create concept↔project edges for listed concepts")
	projectImportCmd.Flags().Bool("dry-run", false, "Show what would be created without making changes")
	projectImportCmd.Flags().Bool("no-fetch", false, "Skip GitHub metadata fetch (create repos with minimal data)")
	projectCmd.AddCommand(projectImportCmd)
}

// ProjectConfig represents a project entry in the config file.
type ProjectConfig struct {
	Name     string   `json:"name,omitempty"`     // Optional: display name (defaults to key)
	Repos    []string `json:"repos,omitempty"`    // GitHub org/repo entries
	Concepts []string `json:"concepts,omitempty"` // Concept IDs for edge creation
	Context  string   `json:"context,omitempty"`  // Path to context markdown file
}

// ProjectImportResult is the response for the project import command.
type ProjectImportResult struct {
	Status          string                `json:"status"`
	ProjectsCreated int                   `json:"projects_created"`
	ProjectsSkipped int                   `json:"projects_skipped"`
	ReposCreated    int                   `json:"repos_created"`
	ReposSkipped    int                   `json:"repos_skipped"`
	ReposFailed     int                   `json:"repos_failed"`
	EdgesCreated    int                   `json:"edges_created"`
	EdgesSkipped    int                   `json:"edges_skipped"`
	Warnings        []string              `json:"warnings,omitempty"`
	Details         *ProjectImportDetails `json:"details,omitempty"`
}

// ProjectImportDetails contains detailed breakdown of import actions.
type ProjectImportDetails struct {
	Projects []ProjectImportAction `json:"projects"`
	Repos    []RepoImportAction    `json:"repos"`
	Edges    []EdgeImportAction    `json:"edges,omitempty"`
}

// ProjectImportAction describes what happened to a project during import.
type ProjectImportAction struct {
	ID     string `json:"id"`
	Action string `json:"action"` // "created" or "skipped"
	Reason string `json:"reason,omitempty"`
}

// RepoImportAction describes what happened to a repo during import.
type RepoImportAction struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	GitHubURL string `json:"github_url,omitempty"`
	Action    string `json:"action"` // "created", "skipped", or "failed"
	Reason    string `json:"reason,omitempty"`
}

// EdgeImportAction describes what happened to an edge during import.
type EdgeImportAction struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Action   string `json:"action"` // "created" or "skipped"
	Reason   string `json:"reason,omitempty"`
}

var projectImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import projects from a config file",
	Long: `Import projects and repos from a JSON config file.

The config file should have project IDs as keys with optional name, repos, and concepts:

{
  "dasm": {
    "name": "DASM",
    "repos": ["matsengrp/netam", "matsengrp/dasm2-experiments"],
    "concepts": ["somatic-hypermutation", "antibody-fitness-prediction"],
    "context": "context/dasm.md"
  }
}

Examples:
  bip project import projects.json
  bip project import projects.json --link-concepts
  bip project import projects.json --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectImport,
}

func runProjectImport(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	configPath := args[0]

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	linkConcepts, _ := cmd.Flags().GetBool("link-concepts")
	noFetch, _ := cmd.Flags().GetBool("no-fetch")

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		exitWithError(ExitDataError, "reading config file: %v", err)
	}

	var projectConfigs map[string]ProjectConfig
	if err := json.Unmarshal(data, &projectConfigs); err != nil {
		exitWithError(ExitDataError, "parsing config file: %v", err)
	}

	if len(projectConfigs) == 0 {
		exitWithError(ExitProjectValidation, "config file contains no projects")
	}

	// Load existing data
	projectsPath := config.ProjectsPath(repoRoot)
	existingProjects, err := storage.ReadAllProjects(projectsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading projects: %v", err)
	}

	reposPath := config.ReposPath(repoRoot)
	existingRepos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		exitWithError(ExitDataError, "reading repos: %v", err)
	}

	edgesPath := config.EdgesPath(repoRoot)
	existingEdges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	// Load concept IDs for validation
	conceptsPath := config.ConceptsPath(repoRoot)
	conceptIDs, err := storage.LoadConceptIDSet(conceptsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading concepts: %v", err)
	}

	// Build lookup maps
	existingProjectIDs := make(map[string]bool)
	for _, p := range existingProjects {
		existingProjectIDs[p.ID] = true
	}

	existingRepoIDs := make(map[string]bool)
	existingRepoURLs := make(map[string]bool)
	for _, r := range existingRepos {
		existingRepoIDs[r.ID] = true
		if r.GitHubURL != "" {
			existingRepoURLs[r.GitHubURL] = true
		}
	}

	existingEdgeKeys := make(map[edge.EdgeKey]bool)
	for _, e := range existingEdges {
		existingEdgeKeys[e.Key()] = true
	}

	// Process imports
	result := &ProjectImportResult{
		Warnings: []string{},
		Details: &ProjectImportDetails{
			Projects: []ProjectImportAction{},
			Repos:    []RepoImportAction{},
			Edges:    []EdgeImportAction{},
		},
	}

	var newProjects []project.Project
	var newRepos []repo.Repo
	var newEdges []edge.Edge

	now := time.Now().UTC().Format(time.RFC3339)
	ghClient := github.NewClient()

	// Sort project IDs for deterministic output
	projectIDs := make([]string, 0, len(projectConfigs))
	for id := range projectConfigs {
		projectIDs = append(projectIDs, id)
	}
	sort.Strings(projectIDs)

	// Process each project
	for _, projectID := range projectIDs {
		cfg := projectConfigs[projectID]

		// Create project if it doesn't exist
		if existingProjectIDs[projectID] {
			result.ProjectsSkipped++
			result.Details.Projects = append(result.Details.Projects, ProjectImportAction{
				ID:     projectID,
				Action: "skipped",
				Reason: "already exists",
			})
		} else {
			name := cfg.Name
			if name == "" {
				name = projectID // Default name to ID
			}

			p := project.Project{
				ID:          projectID,
				Name:        name,
				Description: "",
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			if err := p.ValidateForCreate(); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("project %q: invalid: %v", projectID, err))
				continue
			}

			newProjects = append(newProjects, p)
			existingProjectIDs[projectID] = true // Mark as added for repo validation
			result.ProjectsCreated++
			result.Details.Projects = append(result.Details.Projects, ProjectImportAction{
				ID:     projectID,
				Action: "created",
			})
		}

		// Process repos
		for _, repoSpec := range cfg.Repos {
			repoAction := processRepoImport(repoSpec, projectID, now, ghClient, noFetch,
				existingRepoIDs, existingRepoURLs, &newRepos, result)
			result.Details.Repos = append(result.Details.Repos, repoAction)
		}

		// Process concept edges if requested
		if linkConcepts {
			for _, conceptID := range cfg.Concepts {
				edgeAction := processEdgeImport(conceptID, projectID, now,
					conceptIDs, existingEdgeKeys, &newEdges, result)
				result.Details.Edges = append(result.Details.Edges, edgeAction)
			}
		}
	}

	// Handle dry run
	if dryRun {
		result.Status = "dry_run"
		outputImportResult(result)
		return nil
	}

	// Write new data
	if len(newProjects) > 0 {
		for _, p := range newProjects {
			if err := storage.AppendProject(projectsPath, p); err != nil {
				exitWithError(ExitDataError, "writing project: %v", err)
			}
		}
	}

	if len(newRepos) > 0 {
		for _, r := range newRepos {
			if err := storage.AppendRepo(reposPath, r); err != nil {
				exitWithError(ExitDataError, "writing repo: %v", err)
			}
		}
	}

	if len(newEdges) > 0 {
		existingEdges = append(existingEdges, newEdges...)
		if err := storage.WriteAllEdges(edgesPath, existingEdges); err != nil {
			exitWithError(ExitDataError, "writing edges: %v", err)
		}
	}

	// Rebuild SQLite indexes
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	if len(newProjects) > 0 {
		if _, err := db.RebuildProjectsFromJSONL(projectsPath); err != nil {
			exitWithError(ExitDataError, "updating projects index: %v", err)
		}
	}

	if len(newRepos) > 0 {
		if _, err := db.RebuildReposFromJSONL(reposPath); err != nil {
			exitWithError(ExitDataError, "updating repos index: %v", err)
		}
	}

	if len(newEdges) > 0 {
		if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
			exitWithError(ExitDataError, "updating edges index: %v", err)
		}
	}

	result.Status = "completed"
	outputImportResult(result)

	return nil
}

// processRepoImport handles importing a single repo.
func processRepoImport(repoSpec, projectID, now string, ghClient *github.Client, noFetch bool,
	existingRepoIDs, existingRepoURLs map[string]bool, newRepos *[]repo.Repo, result *ProjectImportResult) RepoImportAction {

	// Normalize GitHub URL
	normalizedURL, err := github.NormalizeGitHubURL(repoSpec)
	if err != nil {
		result.ReposFailed++
		return RepoImportAction{
			ID:        repoSpec,
			ProjectID: projectID,
			Action:    "failed",
			Reason:    fmt.Sprintf("invalid GitHub URL: %v", err),
		}
	}

	// Derive repo ID
	repoID, err := github.DeriveRepoID(repoSpec)
	if err != nil {
		result.ReposFailed++
		return RepoImportAction{
			ID:        repoSpec,
			ProjectID: projectID,
			GitHubURL: normalizedURL,
			Action:    "failed",
			Reason:    fmt.Sprintf("cannot derive repo ID: %v", err),
		}
	}

	// Check if repo URL already exists
	if existingRepoURLs[normalizedURL] {
		result.ReposSkipped++
		return RepoImportAction{
			ID:        repoID,
			ProjectID: projectID,
			GitHubURL: normalizedURL,
			Action:    "skipped",
			Reason:    "GitHub URL already exists",
		}
	}

	// Handle ID collision - append project ID to make unique
	originalID := repoID
	if existingRepoIDs[repoID] {
		repoID = fmt.Sprintf("%s-%s", projectID, repoID)
		// If still collision, use full org-repo format
		if existingRepoIDs[repoID] {
			// ParseGitHubURL won't fail here since NormalizeGitHubURL already succeeded
			owner, repoName, _ := github.ParseGitHubURL(repoSpec)
			if owner != "" && repoName != "" {
				repoID = strings.ToLower(fmt.Sprintf("%s-%s", owner, repoName))
			}
		}
	}

	var r repo.Repo

	if noFetch {
		// Create minimal repo without GitHub metadata
		r = repo.Repo{
			ID:        repoID,
			Project:   projectID,
			Type:      repo.TypeGitHub,
			Name:      originalID, // Use repo name as display name
			GitHubURL: normalizedURL,
			CreatedAt: now,
			UpdatedAt: now,
		}
	} else {
		// Fetch metadata from GitHub
		meta, err := ghClient.FetchRepoMetadata(repoSpec)
		if err != nil {
			switch err {
			case github.ErrRepoNotFound:
				result.ReposFailed++
				return RepoImportAction{
					ID:        repoID,
					ProjectID: projectID,
					GitHubURL: normalizedURL,
					Action:    "failed",
					Reason:    "GitHub repository not found",
				}
			case github.ErrRateLimited:
				result.ReposFailed++
				result.Warnings = append(result.Warnings, "GitHub rate limit exceeded; try --no-fetch or set GITHUB_TOKEN")
				return RepoImportAction{
					ID:        repoID,
					ProjectID: projectID,
					GitHubURL: normalizedURL,
					Action:    "failed",
					Reason:    "GitHub rate limit exceeded",
				}
			case github.ErrUnauthorized:
				result.ReposFailed++
				return RepoImportAction{
					ID:        repoID,
					ProjectID: projectID,
					GitHubURL: normalizedURL,
					Action:    "failed",
					Reason:    "GitHub authentication failed",
				}
			default:
				result.ReposFailed++
				return RepoImportAction{
					ID:        repoID,
					ProjectID: projectID,
					GitHubURL: normalizedURL,
					Action:    "failed",
					Reason:    fmt.Sprintf("GitHub API error: %v", err),
				}
			}
		}

		r = repo.Repo{
			ID:          repoID,
			Project:     projectID,
			Type:        repo.TypeGitHub,
			Name:        meta.Name,
			GitHubURL:   normalizedURL,
			Description: meta.Description,
			Topics:      meta.Topics,
			Language:    meta.Language,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}

	// Validate repo
	if err := r.ValidateForCreate(); err != nil {
		result.ReposFailed++
		return RepoImportAction{
			ID:        repoID,
			ProjectID: projectID,
			GitHubURL: normalizedURL,
			Action:    "failed",
			Reason:    fmt.Sprintf("invalid repo: %v", err),
		}
	}

	*newRepos = append(*newRepos, r)
	existingRepoIDs[repoID] = true
	existingRepoURLs[normalizedURL] = true
	result.ReposCreated++

	return RepoImportAction{
		ID:        repoID,
		ProjectID: projectID,
		GitHubURL: normalizedURL,
		Action:    "created",
	}
}

// processEdgeImport handles creating a concept↔project edge.
func processEdgeImport(conceptID, projectID, now string,
	conceptIDs map[string]bool, existingEdgeKeys map[edge.EdgeKey]bool, newEdges *[]edge.Edge, result *ProjectImportResult) EdgeImportAction {

	// Validate concept exists
	if !conceptIDs[conceptID] {
		result.EdgesSkipped++
		return EdgeImportAction{
			SourceID: "concept:" + conceptID,
			TargetID: "project:" + projectID,
			Action:   "skipped",
			Reason:   fmt.Sprintf("concept %q not found", conceptID),
		}
	}

	// Create edge (concept → project)
	e := edge.Edge{
		SourceID:         "concept:" + conceptID,
		TargetID:         "project:" + projectID,
		RelationshipType: "applied-in",
		Summary:          fmt.Sprintf("Concept %s is applied in project %s", conceptID, projectID),
		CreatedAt:        now,
	}

	// Check for existing edge
	if existingEdgeKeys[e.Key()] {
		result.EdgesSkipped++
		return EdgeImportAction{
			SourceID: e.SourceID,
			TargetID: e.TargetID,
			Action:   "skipped",
			Reason:   "edge already exists",
		}
	}

	*newEdges = append(*newEdges, e)
	existingEdgeKeys[e.Key()] = true
	result.EdgesCreated++

	return EdgeImportAction{
		SourceID: e.SourceID,
		TargetID: e.TargetID,
		Action:   "created",
	}
}

// outputImportResult outputs the import result in the appropriate format.
func outputImportResult(result *ProjectImportResult) {
	if humanOutput {
		if result.Status == "dry_run" {
			fmt.Println("Dry run - no changes made")
			fmt.Println()
		}

		fmt.Printf("Projects: %d created, %d skipped\n", result.ProjectsCreated, result.ProjectsSkipped)
		fmt.Printf("Repos: %d created, %d skipped, %d failed\n", result.ReposCreated, result.ReposSkipped, result.ReposFailed)
		if result.EdgesCreated > 0 || result.EdgesSkipped > 0 {
			fmt.Printf("Edges: %d created, %d skipped\n", result.EdgesCreated, result.EdgesSkipped)
		}

		if len(result.Warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range result.Warnings {
				fmt.Printf("  %s\n", w)
			}
		}

		if result.Status == "completed" && (result.ProjectsCreated > 0 || result.ReposCreated > 0) {
			fmt.Println("\nRun `bip viz --open` to see the updated knowledge graph")
		}
	} else {
		outputJSON(result)
	}
}
