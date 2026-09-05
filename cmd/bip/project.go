package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/project"
	"github.com/matsen/bipartite/internal/repo"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Exit codes for project commands (per CLI contract)
const (
	ExitProjectNotFound   = 2 // Project not found
	ExitProjectValidation = 3 // Validation error (invalid ID, duplicate, has repos)
)

func init() {
	rootCmd.AddCommand(projectCmd)

	// project add flags
	projectAddCmd.Flags().StringP("name", "n", "", "Display name (required)")
	projectAddCmd.Flags().StringP("description", "d", "", "Description text")
	projectAddCmd.MarkFlagRequired("name")
	projectCmd.AddCommand(projectAddCmd)

	// project get - no extra flags
	projectCmd.AddCommand(projectGetCmd)

	// project list - no extra flags
	projectCmd.AddCommand(projectListCmd)

	// project update flags
	projectUpdateCmd.Flags().StringP("name", "n", "", "New display name")
	projectUpdateCmd.Flags().StringP("description", "d", "", "New description")
	projectCmd.AddCommand(projectUpdateCmd)

	// project delete flags
	projectDeleteCmd.Flags().BoolP("force", "f", false, "Delete even if repos exist")
	projectCmd.AddCommand(projectDeleteCmd)

	// project repos - no extra flags
	projectCmd.AddCommand(projectReposCmd)

	// project papers - no extra flags
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  `Commands for managing projects.`,
}

// ProjectAddResult is the response for the project add command.
type ProjectAddResult struct {
	Status  string          `json:"status"`
	Project project.Project `json:"project"`
}

var projectAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a new project",
	Long:  `Add a new project.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectAdd,
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	now := time.Now().UTC().Format(time.RFC3339)

	// Create project
	p := project.Project{
		ID:          projectID,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Validate
	if err := p.ValidateForCreate(); err != nil {
		exitWithError(ExitProjectValidation, "invalid project: %v", err)
	}

	// Check for global ID collision (papers, projects)
	if err := checkGlobalIDCollision(repoRoot, projectID); err != nil {
		exitWithError(ExitProjectValidation, "%v", err)
	}

	// Load existing projects
	projectsPath := config.ProjectsPath(repoRoot)
	projects, err := storage.ReadAllProjects(projectsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading projects: %v", err)
	}

	// Check for duplicate
	if _, found := storage.FindProjectByID(projects, projectID); found {
		exitWithError(ExitProjectValidation, "project with id %q already exists", projectID)
	}

	// Append to JSONL
	if err := storage.AppendProject(projectsPath, p); err != nil {
		exitWithError(ExitDataError, "writing project: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildProjectsFromJSONL(projectsPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Created project: %s\n", projectID)
		fmt.Printf("  Name: %s\n", name)
		if description != "" {
			fmt.Printf("  Desc: %s\n", description)
		}
	} else {
		outputJSON(ProjectAddResult{
			Status:  "created",
			Project: p,
		})
	}

	return nil
}

// checkGlobalIDCollision checks if the project ID conflicts with existing papers.
func checkGlobalIDCollision(repoRoot, projectID string) error {
	// Check papers (refs)
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return fmt.Errorf("reading refs: %w", err)
	}
	for _, ref := range refs {
		if ref.ID == projectID {
			return fmt.Errorf("id %q already exists as a paper", projectID)
		}
	}

	return nil
}

var projectGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a project by ID",
	Long:  `Retrieve a project node by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectGet,
}

func runProjectGet(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	p, err := db.GetProjectByID(projectID)
	if err != nil {
		exitWithError(ExitDataError, "querying project: %v", err)
	}
	if p == nil {
		exitWithError(ExitProjectNotFound, "project %q not found", projectID)
	}

	if humanOutput {
		fmt.Printf("Project: %s\n", p.ID)
		fmt.Printf("Name:    %s\n", p.Name)
		if p.Description != "" {
			fmt.Printf("Desc:    %s\n", p.Description)
		}
		fmt.Printf("Created: %s\n", p.CreatedAt)
	} else {
		outputJSON(p)
	}

	return nil
}

// ProjectListResult is the response for the project list command.
type ProjectListResult struct {
	Projects []project.Project `json:"projects"`
	Count    int               `json:"count"`
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long:  `List all projects.`,
	RunE:  runProjectList,
}

func runProjectList(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	projects, err := db.GetAllProjects()
	if err != nil {
		exitWithError(ExitDataError, "querying projects: %v", err)
	}

	if humanOutput {
		if len(projects) == 0 {
			fmt.Println("No projects found")
			return nil
		}
		for i, p := range projects {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("Project: %s\n", p.ID)
			fmt.Printf("Name:    %s\n", p.Name)
			if p.Description != "" {
				fmt.Printf("Desc:    %s\n", p.Description)
			}
		}
		fmt.Printf("\nTotal: %d projects\n", len(projects))
	} else {
		if projects == nil {
			projects = []project.Project{}
		}
		outputJSON(ProjectListResult{
			Projects: projects,
			Count:    len(projects),
		})
	}

	return nil
}

// ProjectUpdateResult is the response for the project update command.
type ProjectUpdateResult struct {
	Status  string          `json:"status"`
	Project project.Project `json:"project"`
}

var projectUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a project",
	Long:  `Update an existing project node.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectUpdate,
}

func runProjectUpdate(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]

	nameFlag := cmd.Flags().Changed("name")
	descFlag := cmd.Flags().Changed("description")

	if !nameFlag && !descFlag {
		exitWithError(ExitProjectValidation, "no update flags provided (use --name or --description)")
	}

	// Load existing projects
	projectsPath := config.ProjectsPath(repoRoot)
	projects, err := storage.ReadAllProjects(projectsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading projects: %v", err)
	}

	// Find project
	idx, found := storage.FindProjectByID(projects, projectID)
	if !found {
		exitWithError(ExitProjectNotFound, "project %q not found", projectID)
	}

	// Apply updates
	p := projects[idx]
	if nameFlag {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			exitWithError(ExitProjectValidation, "name cannot be empty")
		}
		p.Name = name
	}
	if descFlag {
		description, _ := cmd.Flags().GetString("description")
		p.Description = description
	}
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	projects[idx] = p

	// Write back
	if err := storage.WriteAllProjects(projectsPath, projects); err != nil {
		exitWithError(ExitDataError, "writing projects: %v", err)
	}

	// Update SQLite index
	db := mustOpenDatabase(repoRoot)
	defer db.Close()
	if _, err := db.RebuildProjectsFromJSONL(projectsPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	// Output
	if humanOutput {
		fmt.Printf("Updated project: %s\n", projectID)
		fmt.Printf("  Name: %s\n", p.Name)
		if p.Description != "" {
			fmt.Printf("  Desc: %s\n", p.Description)
		}
	} else {
		outputJSON(ProjectUpdateResult{
			Status:  "updated",
			Project: p,
		})
	}

	return nil
}

// ProjectDeleteResult is the response for the project delete command.
type ProjectDeleteResult struct {
	Status       string `json:"status"`
	ID           string `json:"id"`
	ReposRemoved int    `json:"repos_removed"`
}

// ProjectDeleteBlockedResult is the response when delete is blocked by repos.
type ProjectDeleteBlockedResult struct {
	Error     string `json:"error"`
	RepoCount int    `json:"repo_count"`
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project",
	Long:  `Delete a project.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectDelete,
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Validate project exists and check dependencies
	projects, repos, repoCount := validateProjectForDelete(repoRoot, projectID)

	// Block if not force and has dependencies
	if !force && repoCount > 0 {
		outputDeleteBlocked(projectID, repoCount)
		os.Exit(ExitProjectValidation)
	}

	// Perform cascade delete
	reposRemoved := cascadeDeleteProject(repoRoot, projectID, projects, repos, repoCount)

	// Output result
	outputDeleteResult(projectID, reposRemoved)
	return nil
}

// validateProjectForDelete checks project exists and counts dependencies.
func validateProjectForDelete(repoRoot, projectID string) ([]project.Project, []repo.Repo, int) {
	projectsPath := config.ProjectsPath(repoRoot)
	projects, err := storage.ReadAllProjects(projectsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading projects: %v", err)
	}
	if _, found := storage.FindProjectByID(projects, projectID); !found {
		exitWithError(ExitProjectNotFound, "project %q not found", projectID)
	}

	reposPath := config.ReposPath(repoRoot)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil {
		exitWithError(ExitDataError, "reading repos: %v", err)
	}

	repoCount := countReposByProject(repos, projectID)

	return projects, repos, repoCount
}

// outputDeleteBlocked outputs error when delete is blocked by dependencies.
func outputDeleteBlocked(projectID string, repoCount int) {
	if humanOutput {
		fmt.Fprintf(os.Stderr, "error: project %q has %d repos; use --force to delete anyway\n", projectID, repoCount)
	} else {
		outputJSON(ProjectDeleteBlockedResult{
			Error:     fmt.Sprintf("project %q has %d repos; use --force to delete anyway", projectID, repoCount),
			RepoCount: repoCount,
		})
	}
}

// cascadeDeleteProject deletes a project and its dependent repos.
func cascadeDeleteProject(repoRoot, projectID string, projects []project.Project, repos []repo.Repo, repoCount int) (reposRemoved int) {
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	reposPath := config.ReposPath(repoRoot)
	projectsPath := config.ProjectsPath(repoRoot)

	// Delete repos belonging to this project
	if repoCount > 0 {
		repos, reposRemoved = deleteReposByProject(repos, projectID)
		if err := storage.WriteAllRepos(reposPath, repos); err != nil {
			exitWithError(ExitDataError, "writing repos: %v", err)
		}
		if _, err := db.RebuildReposFromJSONL(reposPath); err != nil {
			exitWithError(ExitDataError, "rebuilding repos index: %v", err)
		}
	}

	// Delete project from JSONL
	projects, _ = storage.DeleteProjectFromSlice(projects, projectID)
	if err := storage.WriteAllProjects(projectsPath, projects); err != nil {
		exitWithError(ExitDataError, "writing projects: %v", err)
	}

	// Rebuild projects index
	if _, err := db.RebuildProjectsFromJSONL(projectsPath); err != nil {
		exitWithError(ExitDataError, "updating index: %v", err)
	}

	return reposRemoved
}

// outputDeleteResult outputs the delete result.
func outputDeleteResult(projectID string, reposRemoved int) {
	if humanOutput {
		if reposRemoved > 0 {
			fmt.Printf("Deleted project %q with %d repos\n", projectID, reposRemoved)
		} else {
			fmt.Printf("Deleted project %q\n", projectID)
		}
	} else {
		outputJSON(ProjectDeleteResult{
			Status:       "deleted",
			ID:           projectID,
			ReposRemoved: reposRemoved,
		})
	}
}

// countReposByProject counts repos belonging to a project.
func countReposByProject(repos []repo.Repo, projectID string) int {
	count := 0
	for _, r := range repos {
		if r.Project == projectID {
			count++
		}
	}
	return count
}

// deleteReposByProject removes all repos belonging to a project.
func deleteReposByProject(repos []repo.Repo, projectID string) ([]repo.Repo, int) {
	var remaining []repo.Repo
	removed := 0
	for _, r := range repos {
		if r.Project != projectID {
			remaining = append(remaining, r)
		} else {
			removed++
		}
	}
	return remaining, removed
}

// ProjectReposResult is the response for the project repos command.
type ProjectReposResult struct {
	ProjectID string      `json:"project_id"`
	Repos     []repo.Repo `json:"repos"`
	Count     int         `json:"count"`
}

var projectReposCmd = &cobra.Command{
	Use:   "repos <id>",
	Short: "List repos belonging to a project",
	Long:  `Query all repos linked to a specific project.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectRepos,
}

func runProjectRepos(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Verify project exists
	p, err := db.GetProjectByID(projectID)
	if err != nil {
		exitWithError(ExitDataError, "querying project: %v", err)
	}
	if p == nil {
		exitWithError(ExitProjectNotFound, "project %q not found", projectID)
	}

	// Get repos
	repos, err := db.GetReposByProject(projectID)
	if err != nil {
		exitWithError(ExitDataError, "querying repos: %v", err)
	}

	if humanOutput {
		fmt.Printf("Repos for project: %s\n", projectID)
		if len(repos) == 0 {
			fmt.Println("\n(no repos)")
		} else {
			fmt.Println()
			for _, r := range repos {
				fmt.Printf("  %s (%s)\n", r.ID, r.Type)
				if r.GitHubURL != "" {
					fmt.Printf("    %s\n", r.GitHubURL)
				}
				if r.Language != "" || len(r.Topics) > 0 {
					parts := []string{}
					if r.Language != "" {
						parts = append(parts, r.Language)
					}
					if len(r.Topics) > 0 {
						parts = append(parts, strings.Join(r.Topics, ", "))
					}
					fmt.Printf("    %s\n", strings.Join(parts, " · "))
				}
			}
		}
		fmt.Printf("\nTotal: %d repos\n", len(repos))
	} else {
		if repos == nil {
			repos = []repo.Repo{}
		}
		outputJSON(ProjectReposResult{
			ProjectID: projectID,
			Repos:     repos,
			Count:     len(repos),
		})
	}

	return nil
}
