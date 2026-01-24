package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/github"
	"github.com/matsen/bipartite/internal/repo"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Exit codes for repo commands (per CLI contract)
const (
	ExitRepoNotFound    = 2 // Repo not found
	ExitRepoValidation  = 3 // Validation error
	ExitRepoDataError   = 4 // Data error
	ExitRepoGitHubError = 5 // GitHub API error
)

func init() {
	rootCmd.AddCommand(repoCmd)

	// repo add flags
	repoAddCmd.Flags().StringP("project", "p", "", "Project ID (required)")
	repoAddCmd.Flags().String("id", "", "Override derived repo ID")
	repoAddCmd.Flags().Bool("manual", false, "Create manual repo (no GitHub fetch)")
	repoAddCmd.Flags().StringP("name", "n", "", "Display name (required for manual)")
	repoAddCmd.Flags().StringP("description", "d", "", "Description (manual only)")
	repoAddCmd.Flags().String("topics", "", "Comma-separated topics (manual only)")
	repoAddCmd.MarkFlagRequired("project")
	repoCmd.AddCommand(repoAddCmd)

	// repo get - no extra flags
	repoCmd.AddCommand(repoGetCmd)

	// repo list flags
	repoListCmd.Flags().StringP("project", "p", "", "Filter by project")
	repoCmd.AddCommand(repoListCmd)

	// repo update flags
	repoUpdateCmd.Flags().StringP("name", "n", "", "New display name")
	repoUpdateCmd.Flags().StringP("description", "d", "", "New description")
	repoUpdateCmd.Flags().String("topics", "", "New comma-separated topics")
	repoCmd.AddCommand(repoUpdateCmd)

	// repo delete - no extra flags
	repoCmd.AddCommand(repoDeleteCmd)

	// repo refresh - no extra flags
	repoCmd.AddCommand(repoRefreshCmd)
}

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repository nodes",
	Long:  `Commands for managing repository nodes in the knowledge graph.`,
}

// RepoAddResult is the response for the repo add command.
type RepoAddResult struct {
	Status string    `json:"status"`
	Repo   repo.Repo `json:"repo"`
}

var repoAddCmd = &cobra.Command{
	Use:   "add <github-url-or-org/repo>",
	Short: "Add a repo to a project",
	Long: `Add a GitHub repository to a project.

Examples:
  bip repo add https://github.com/matsen/bipartite --project bipartite
  bip repo add matsen/bipartite --project bipartite
  bip repo add --manual --project dasm2 --id internal-tools --name "Internal Tools"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRepoAdd,
}

func runRepoAdd(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	projectID, _ := cmd.Flags().GetString("project")
	repoID, _ := cmd.Flags().GetString("id")
	isManual, _ := cmd.Flags().GetBool("manual")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	topicsStr, _ := cmd.Flags().GetString("topics")

	// Verify project exists
	projectsPath := config.ProjectsPath(repoRoot)
	projectIDs, err := storage.LoadProjectIDSet(projectsPath)
	if err != nil {
		exitWithError(ExitRepoDataError, "reading projects: %v", err)
	}
	if !projectIDs[projectID] {
		exitWithError(ExitRepoValidation, "project %q not found", projectID)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var r repo.Repo

	if isManual {
		// Manual repo creation
		if repoID == "" {
			exitWithError(ExitRepoValidation, "--id is required for manual repos")
		}
		if name == "" {
			exitWithError(ExitRepoValidation, "--name is required for manual repos")
		}

		var topics []string
		if topicsStr != "" {
			topics = parseTopics(topicsStr)
		}

		r = repo.Repo{
			ID:          repoID,
			Project:     projectID,
			Type:        repo.TypeManual,
			Name:        name,
			Description: description,
			Topics:      topics,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	} else {
		// GitHub repo creation
		if len(args) == 0 {
			exitWithError(ExitRepoValidation, "GitHub URL or org/repo required (or use --manual)")
		}
		githubInput := args[0]

		// Parse and normalize GitHub URL
		normalizedURL, err := github.NormalizeGitHubURL(githubInput)
		if err != nil {
			exitWithError(ExitRepoValidation, "invalid GitHub URL: %v", err)
		}

		// Derive ID if not provided
		if repoID == "" {
			repoID, err = github.DeriveRepoID(githubInput)
			if err != nil {
				exitWithError(ExitRepoValidation, "cannot derive repo ID: %v", err)
			}
		}

		// Check for duplicate GitHub URL
		reposPath := config.ReposPath(repoRoot)
		repos, err := storage.ReadAllRepos(reposPath)
		if err != nil {
			exitWithError(ExitRepoDataError, "reading repos: %v", err)
		}
		if idx, found := storage.FindRepoByGitHubURL(repos, normalizedURL); found {
			existingRepo := repos[idx]
			exitWithError(ExitRepoValidation, "repo with GitHub URL %q already exists (id: %q, project: %q)", normalizedURL, existingRepo.ID, existingRepo.Project)
		}

		// Fetch metadata from GitHub
		client := github.NewClient()
		meta, err := client.FetchRepoMetadata(githubInput)
		if err != nil {
			switch err {
			case github.ErrRepoNotFound:
				exitWithError(ExitRepoGitHubError, "GitHub repository not found: %s", githubInput)
			case github.ErrRateLimited:
				exitWithError(ExitRepoGitHubError, "GitHub API rate limit exceeded; try again later or set GITHUB_TOKEN")
			case github.ErrUnauthorized:
				exitWithError(ExitRepoGitHubError, "GitHub API authentication failed; check GITHUB_TOKEN")
			default:
				exitWithError(ExitRepoGitHubError, "GitHub API error: %v", err)
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
		exitWithError(ExitRepoValidation, "invalid repo: %v", err)
	}

	// Check for duplicate ID
	reposPath := config.ReposPath(repoRoot)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		exitWithError(ExitRepoDataError, "reading repos: %v", err)
	}
	if _, found := storage.FindRepoByID(repos, r.ID); found {
		exitWithError(ExitRepoValidation, "repo with id %q already exists", r.ID)
	}

	// Append to JSONL
	if err := storage.AppendRepo(reposPath, r); err != nil {
		exitWithError(ExitRepoDataError, "writing repo: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildReposFromJSONL(reposPath); err != nil {
		exitWithError(ExitRepoDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Created repo: %s\n", r.ID)
		fmt.Printf("  Project:  %s\n", r.Project)
		fmt.Printf("  Type:     %s\n", r.Type)
		fmt.Printf("  Name:     %s\n", r.Name)
		if r.GitHubURL != "" {
			fmt.Printf("  URL:      %s\n", r.GitHubURL)
		}
		if r.Description != "" {
			fmt.Printf("  Desc:     %s\n", r.Description)
		}
		if r.Language != "" {
			fmt.Printf("  Language: %s\n", r.Language)
		}
		if len(r.Topics) > 0 {
			fmt.Printf("  Topics:   %s\n", strings.Join(r.Topics, ", "))
		}
	} else {
		outputJSON(RepoAddResult{
			Status: "created",
			Repo:   r,
		})
	}

	return nil
}

// parseTopics parses a comma-separated string into a slice of trimmed strings.
func parseTopics(s string) []string {
	parts := strings.Split(s, ",")
	var topics []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			topics = append(topics, p)
		}
	}
	return topics
}

var repoGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a repo by ID",
	Long:  `Retrieve a repository node by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoGet,
}

func runRepoGet(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	repoID := args[0]

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	r, err := db.GetRepoByID(repoID)
	if err != nil {
		exitWithError(ExitRepoDataError, "querying repo: %v", err)
	}
	if r == nil {
		exitWithError(ExitRepoNotFound, "repo %q not found", repoID)
	}

	if humanOutput {
		fmt.Printf("Repo:     %s\n", r.ID)
		fmt.Printf("Project:  %s\n", r.Project)
		fmt.Printf("Type:     %s\n", r.Type)
		fmt.Printf("Name:     %s\n", r.Name)
		if r.GitHubURL != "" {
			fmt.Printf("URL:      %s\n", r.GitHubURL)
		}
		if r.Description != "" {
			fmt.Printf("Desc:     %s\n", r.Description)
		}
		if r.Language != "" {
			fmt.Printf("Language: %s\n", r.Language)
		}
		if len(r.Topics) > 0 {
			fmt.Printf("Topics:   %s\n", strings.Join(r.Topics, ", "))
		}
		fmt.Printf("Created:  %s\n", r.CreatedAt)
	} else {
		outputJSON(r)
	}

	return nil
}

// RepoListResult is the response for the repo list command.
type RepoListResult struct {
	Repos []repo.Repo `json:"repos"`
	Count int         `json:"count"`
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repos",
	Long:  `List all repository nodes in the knowledge graph.`,
	RunE:  runRepoList,
}

func runRepoList(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectFilter, _ := cmd.Flags().GetString("project")

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	var repos []repo.Repo
	var err error

	if projectFilter != "" {
		repos, err = db.GetReposByProject(projectFilter)
	} else {
		repos, err = db.GetAllRepos()
	}
	if err != nil {
		exitWithError(ExitRepoDataError, "querying repos: %v", err)
	}

	if humanOutput {
		if len(repos) == 0 {
			if projectFilter != "" {
				fmt.Printf("No repos found for project %q\n", projectFilter)
			} else {
				fmt.Println("No repos found")
			}
			return nil
		}
		for i, r := range repos {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("Repo:    %s\n", r.ID)
			fmt.Printf("Project: %s\n", r.Project)
			fmt.Printf("Type:    %s\n", r.Type)
			fmt.Printf("Name:    %s\n", r.Name)
			if r.GitHubURL != "" {
				fmt.Printf("URL:     %s\n", r.GitHubURL)
			}
		}
		fmt.Printf("\nTotal: %d repos\n", len(repos))
	} else {
		if repos == nil {
			repos = []repo.Repo{}
		}
		outputJSON(RepoListResult{
			Repos: repos,
			Count: len(repos),
		})
	}

	return nil
}

// RepoUpdateResult is the response for the repo update command.
type RepoUpdateResult struct {
	Status string    `json:"status"`
	Repo   repo.Repo `json:"repo"`
}

var repoUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a repo",
	Long:  `Update an existing repository node.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoUpdate,
}

func runRepoUpdate(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	repoID := args[0]

	nameFlag := cmd.Flags().Changed("name")
	descFlag := cmd.Flags().Changed("description")
	topicsFlag := cmd.Flags().Changed("topics")

	if !nameFlag && !descFlag && !topicsFlag {
		exitWithError(ExitRepoValidation, "no update flags provided (use --name, --description, or --topics)")
	}

	// Load existing repos
	reposPath := config.ReposPath(repoRoot)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		exitWithError(ExitRepoDataError, "reading repos: %v", err)
	}

	// Find repo
	idx, found := storage.FindRepoByID(repos, repoID)
	if !found {
		exitWithError(ExitRepoNotFound, "repo %q not found", repoID)
	}

	// Apply updates
	r := repos[idx]
	if nameFlag {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			exitWithError(ExitRepoValidation, "name cannot be empty")
		}
		r.Name = name
	}
	if descFlag {
		description, _ := cmd.Flags().GetString("description")
		r.Description = description
	}
	if topicsFlag {
		topicsStr, _ := cmd.Flags().GetString("topics")
		if topicsStr == "" {
			r.Topics = nil
		} else {
			r.Topics = parseTopics(topicsStr)
		}
	}
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	repos[idx] = r

	// Write back
	if err := storage.WriteAllRepos(reposPath, repos); err != nil {
		exitWithError(ExitRepoDataError, "writing repos: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildReposFromJSONL(reposPath); err != nil {
		exitWithError(ExitRepoDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Updated repo: %s\n", repoID)
		fmt.Printf("  Name:     %s\n", r.Name)
		if r.Description != "" {
			fmt.Printf("  Desc:     %s\n", r.Description)
		}
		if len(r.Topics) > 0 {
			fmt.Printf("  Topics:   %s\n", strings.Join(r.Topics, ", "))
		}
	} else {
		outputJSON(RepoUpdateResult{
			Status: "updated",
			Repo:   r,
		})
	}

	return nil
}

// RepoDeleteResult is the response for the repo delete command.
type RepoDeleteResult struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a repo",
	Long:  `Delete a repository node from the knowledge graph.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoDelete,
}

func runRepoDelete(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	repoID := args[0]

	// Load existing repos
	reposPath := config.ReposPath(repoRoot)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		exitWithError(ExitRepoDataError, "reading repos: %v", err)
	}

	// Find and delete repo
	repos, found := storage.DeleteRepoFromSlice(repos, repoID)
	if !found {
		exitWithError(ExitRepoNotFound, "repo %q not found", repoID)
	}

	// Write back
	if err := storage.WriteAllRepos(reposPath, repos); err != nil {
		exitWithError(ExitRepoDataError, "writing repos: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildReposFromJSONL(reposPath); err != nil {
		exitWithError(ExitRepoDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Deleted repo %q\n", repoID)
	} else {
		outputJSON(RepoDeleteResult{
			Status: "deleted",
			ID:     repoID,
		})
	}

	return nil
}

// RepoRefreshResult is the response for the repo refresh command.
type RepoRefreshResult struct {
	Status string    `json:"status"`
	Repo   repo.Repo `json:"repo"`
}

var repoRefreshCmd = &cobra.Command{
	Use:   "refresh <id>",
	Short: "Refresh GitHub metadata for a repo",
	Long:  `Re-fetch metadata from GitHub for a repository.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoRefresh,
}

func runRepoRefresh(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	repoID := args[0]

	// Load existing repos
	reposPath := config.ReposPath(repoRoot)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		exitWithError(ExitRepoDataError, "reading repos: %v", err)
	}

	// Find repo
	idx, found := storage.FindRepoByID(repos, repoID)
	if !found {
		exitWithError(ExitRepoNotFound, "repo %q not found", repoID)
	}

	r := repos[idx]

	// Check if repo is GitHub type
	if r.Type != repo.TypeGitHub {
		exitWithError(ExitRepoValidation, "repo %q is manual type (no GitHub URL to refresh)", repoID)
	}

	// Fetch updated metadata from GitHub
	client := github.NewClient()
	meta, err := client.FetchRepoMetadata(r.GitHubURL)
	if err != nil {
		switch err {
		case github.ErrRepoNotFound:
			exitWithError(ExitRepoGitHubError, "GitHub repository not found (may have been deleted or made private)")
		case github.ErrRateLimited:
			exitWithError(ExitRepoGitHubError, "GitHub API rate limit exceeded; try again later or set GITHUB_TOKEN")
		case github.ErrUnauthorized:
			exitWithError(ExitRepoGitHubError, "GitHub API authentication failed; check GITHUB_TOKEN")
		default:
			exitWithError(ExitRepoGitHubError, "GitHub API error: %v", err)
		}
	}

	// Update metadata
	r.Name = meta.Name
	r.Description = meta.Description
	r.Topics = meta.Topics
	r.Language = meta.Language
	r.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	repos[idx] = r

	// Write back
	if err := storage.WriteAllRepos(reposPath, repos); err != nil {
		exitWithError(ExitRepoDataError, "writing repos: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildReposFromJSONL(reposPath); err != nil {
		exitWithError(ExitRepoDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Refreshed repo: %s\n", repoID)
		fmt.Printf("  Name:     %s\n", r.Name)
		if r.Description != "" {
			fmt.Printf("  Desc:     %s\n", r.Description)
		}
		if r.Language != "" {
			fmt.Printf("  Language: %s\n", r.Language)
		}
		if len(r.Topics) > 0 {
			fmt.Printf("  Topics:   %s\n", strings.Join(r.Topics, ", "))
		}
	} else {
		outputJSON(RepoRefreshResult{
			Status: "refreshed",
			Repo:   r,
		})
	}

	return nil
}
