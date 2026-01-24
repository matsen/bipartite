package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/project"
	"github.com/matsen/bipartite/internal/repo"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Exit codes for project commands (per CLI contract)
const (
	ExitProjectNotFound   = 2 // Project not found
	ExitProjectValidation = 3 // Validation error (invalid ID, duplicate, has edges)
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
	projectDeleteCmd.Flags().BoolP("force", "f", false, "Delete even if edges or repos exist")
	projectCmd.AddCommand(projectDeleteCmd)

	// project repos - no extra flags
	projectCmd.AddCommand(projectReposCmd)

	// project concepts flags
	projectConceptsCmd.Flags().StringP("type", "t", "", "Filter by relationship type")
	projectCmd.AddCommand(projectConceptsCmd)

	// project papers - no extra flags
	projectCmd.AddCommand(projectPapersCmd)
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project nodes",
	Long:  `Commands for managing project nodes in the knowledge graph.`,
}

// ProjectAddResult is the response for the project add command.
type ProjectAddResult struct {
	Status  string          `json:"status"`
	Project project.Project `json:"project"`
}

var projectAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a new project",
	Long:  `Add a new project node to the knowledge graph.`,
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

	// Check for global ID collision (papers, concepts, projects)
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

// checkGlobalIDCollision checks if the project ID conflicts with existing papers or concepts.
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

	// Check concepts
	conceptsPath := config.ConceptsPath(repoRoot)
	conceptIDs, err := storage.LoadConceptIDSet(conceptsPath)
	if err != nil {
		return fmt.Errorf("reading concepts: %w", err)
	}
	if conceptIDs[projectID] {
		return fmt.Errorf("id %q already exists as a concept", projectID)
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
	Long:  `List all project nodes in the knowledge graph.`,
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
	EdgesRemoved int    `json:"edges_removed"`
}

// ProjectDeleteBlockedResult is the response when delete is blocked by edges or repos.
type ProjectDeleteBlockedResult struct {
	Error     string `json:"error"`
	EdgeCount int    `json:"edge_count"`
	RepoCount int    `json:"repo_count"`
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project",
	Long:  `Delete a project node from the knowledge graph.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectDelete,
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	// Validate project exists and check dependencies
	projects, repos, repoCount, edgeCount := validateProjectForDelete(repoRoot, projectID)

	// Block if not force and has dependencies
	if !force && (repoCount > 0 || edgeCount > 0) {
		outputDeleteBlocked(projectID, repoCount, edgeCount)
		os.Exit(ExitProjectValidation)
	}

	// Perform cascade delete
	reposRemoved, edgesRemoved := cascadeDeleteProject(repoRoot, projectID, projects, repos, repoCount, edgeCount)

	// Output result
	outputDeleteResult(projectID, reposRemoved, edgesRemoved)
	return nil
}

// validateProjectForDelete checks project exists and counts dependencies.
func validateProjectForDelete(repoRoot, projectID string) ([]project.Project, []repo.Repo, int, int) {
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
	edgeCount := countEdgesForProject(repoRoot, projectID)

	return projects, repos, repoCount, edgeCount
}

// outputDeleteBlocked outputs error when delete is blocked by dependencies.
func outputDeleteBlocked(projectID string, repoCount, edgeCount int) {
	if humanOutput {
		fmt.Fprintf(os.Stderr, "error: project %q has %d repos and %d linked edges; use --force to delete anyway\n", projectID, repoCount, edgeCount)
	} else {
		outputJSON(ProjectDeleteBlockedResult{
			Error:     fmt.Sprintf("project %q has %d repos and %d linked edges; use --force to delete anyway", projectID, repoCount, edgeCount),
			EdgeCount: edgeCount,
			RepoCount: repoCount,
		})
	}
}

// cascadeDeleteProject deletes a project and its dependent repos/edges.
func cascadeDeleteProject(repoRoot, projectID string, projects []project.Project, repos []repo.Repo, repoCount, edgeCount int) (reposRemoved, edgesRemoved int) {
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

	// Delete edges involving this project
	if edgeCount > 0 {
		edgesRemoved = deleteEdgesForProject(repoRoot, projectID, db)
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

	return reposRemoved, edgesRemoved
}

// outputDeleteResult outputs the delete result.
func outputDeleteResult(projectID string, reposRemoved, edgesRemoved int) {
	if humanOutput {
		if reposRemoved > 0 || edgesRemoved > 0 {
			fmt.Printf("Deleted project %q with %d repos and %d edges\n", projectID, reposRemoved, edgesRemoved)
		} else {
			fmt.Printf("Deleted project %q\n", projectID)
		}
	} else {
		outputJSON(ProjectDeleteResult{
			Status:       "deleted",
			ID:           projectID,
			ReposRemoved: reposRemoved,
			EdgesRemoved: edgesRemoved,
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

// countEdgesForProject counts edges involving a project (either source or target).
func countEdgesForProject(repoRoot, projectID string) int {
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		return 0
	}

	prefixedID := "project:" + projectID
	count := 0
	for _, e := range edges {
		if e.SourceID == prefixedID || e.TargetID == prefixedID {
			count++
		}
	}
	return count
}

// deleteEdgesForProject removes all edges involving a project.
func deleteEdgesForProject(repoRoot, projectID string, db *storage.DB) int {
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	prefixedID := "project:" + projectID
	var remaining []edge.Edge
	removed := 0
	for _, e := range edges {
		if e.SourceID != prefixedID && e.TargetID != prefixedID {
			remaining = append(remaining, e)
		} else {
			removed++
		}
	}

	if err := storage.WriteAllEdges(edgesPath, remaining); err != nil {
		exitWithError(ExitDataError, "writing edges: %v", err)
	}

	if _, err := db.RebuildEdgesFromJSONL(edgesPath); err != nil {
		exitWithError(ExitDataError, "rebuilding edges index: %v", err)
	}

	return removed
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

// ProjectConceptsResult is the response for the project concepts command.
type ProjectConceptsResult struct {
	ProjectID string               `json:"project_id"`
	Concepts  []ProjectConceptEdge `json:"concepts"`
	Count     int                  `json:"count"`
}

// ProjectConceptEdge represents a concept linked to a project.
type ProjectConceptEdge struct {
	ConceptID        string `json:"concept_id"`
	RelationshipType string `json:"relationship_type"`
	Summary          string `json:"summary"`
}

var projectConceptsCmd = &cobra.Command{
	Use:   "concepts <id>",
	Short: "List concepts linked to a project",
	Long:  `Query all concepts linked to a specific project.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectConcepts,
}

func runProjectConcepts(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	projectID := args[0]
	relType, _ := cmd.Flags().GetString("type")

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

	// Get edges involving this project
	concepts, err := getConceptsForProject(repoRoot, projectID, relType)
	if err != nil {
		exitWithError(ExitDataError, "querying concepts: %v", err)
	}

	if humanOutput {
		fmt.Printf("Concepts linked to project: %s\n", projectID)
		if len(concepts) == 0 {
			fmt.Println("\n(no concepts)")
		} else {
			fmt.Println()
			for _, c := range concepts {
				fmt.Printf("  %s --[%s]--> project:%s\n", c.ConceptID, c.RelationshipType, projectID)
				fmt.Printf("    %q\n", c.Summary)
			}
		}
		fmt.Printf("\nTotal: %d concepts\n", len(concepts))
	} else {
		if concepts == nil {
			concepts = []ProjectConceptEdge{}
		}
		outputJSON(ProjectConceptsResult{
			ProjectID: projectID,
			Concepts:  concepts,
			Count:     len(concepts),
		})
	}

	return nil
}

// getConceptsForProject queries edges where the project is target and source is a concept.
func getConceptsForProject(repoRoot, projectID, relType string) ([]ProjectConceptEdge, error) {
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		return nil, err
	}

	prefixedID := "project:" + projectID
	var results []ProjectConceptEdge
	for _, e := range edges {
		// Check if this edge targets our project
		if e.TargetID == prefixedID {
			// Check if source is a concept (has concept: prefix)
			if strings.HasPrefix(e.SourceID, "concept:") {
				if relType == "" || e.RelationshipType == relType {
					results = append(results, ProjectConceptEdge{
						ConceptID:        e.SourceID,
						RelationshipType: e.RelationshipType,
						Summary:          e.Summary,
					})
				}
			}
		}
		// Also check if project is source (project→concept edges)
		if e.SourceID == prefixedID {
			if strings.HasPrefix(e.TargetID, "concept:") {
				if relType == "" || e.RelationshipType == relType {
					results = append(results, ProjectConceptEdge{
						ConceptID:        e.TargetID,
						RelationshipType: e.RelationshipType,
						Summary:          e.Summary,
					})
				}
			}
		}
	}
	return results, nil
}

// ProjectPapersResult is the response for the project papers command.
type ProjectPapersResult struct {
	ProjectID string             `json:"project_id"`
	Papers    []ProjectPaperEdge `json:"papers"`
	Count     int                `json:"count"`
}

// ProjectPaperEdge represents a paper linked to a project via a concept.
type ProjectPaperEdge struct {
	PaperID          string `json:"paper_id"`
	ViaConcept       string `json:"via_concept"`
	RelationshipType string `json:"relationship_type"`
	Summary          string `json:"summary"`
}

var projectPapersCmd = &cobra.Command{
	Use:   "papers <id>",
	Short: "List papers relevant to a project (via concepts)",
	Long:  `Query all papers linked to concepts that are linked to the project.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectPapers,
}

func runProjectPapers(cmd *cobra.Command, args []string) error {
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

	// Get papers transitively via concepts
	papers, err := getPapersForProjectTransitive(repoRoot, projectID)
	if err != nil {
		exitWithError(ExitDataError, "querying papers: %v", err)
	}

	if humanOutput {
		fmt.Printf("Papers relevant to project: %s (via concepts)\n", projectID)
		if len(papers) == 0 {
			fmt.Println("\n(no papers found via linked concepts)")
		} else {
			fmt.Println()
			for _, pe := range papers {
				fmt.Printf("  %s\n", pe.PaperID)
				fmt.Printf("    via %s --[%s]--> paper\n", pe.ViaConcept, pe.RelationshipType)
				fmt.Printf("    %q\n", pe.Summary)
			}
		}
		fmt.Printf("\nTotal: %d papers\n", len(papers))
	} else {
		if papers == nil {
			papers = []ProjectPaperEdge{}
		}
		outputJSON(ProjectPapersResult{
			ProjectID: projectID,
			Papers:    papers,
			Count:     len(papers),
		})
	}

	return nil
}

// getPapersForProjectTransitive finds papers via: project ← concepts ← papers
func getPapersForProjectTransitive(repoRoot, projectID string) ([]ProjectPaperEdge, error) {
	// Step 1: Find all concepts linked to this project
	concepts, err := getConceptsForProject(repoRoot, projectID, "")
	if err != nil {
		return nil, err
	}

	if len(concepts) == 0 {
		return nil, nil
	}

	// Build set of concept IDs (keeping concept: prefix for edge lookup)
	// Since paper→concept edges now use prefixed targets like "concept:vi"
	conceptIDs := make(map[string]bool)
	for _, c := range concepts {
		conceptIDs[c.ConceptID] = true
	}

	// Step 2: Find all papers linked to those concepts
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		return nil, err
	}

	var results []ProjectPaperEdge
	seen := make(map[string]bool) // Deduplicate paper+concept combinations

	for _, e := range edges {
		// Paper→concept edges have unprefixed paper ID as source and "concept:X" as target
		// Check if target is one of our concepts
		if conceptIDs[e.TargetID] {
			// Source should be a paper (not prefixed with concept: or project:)
			if !strings.Contains(e.SourceID, ":") {
				key := e.SourceID + "|" + e.TargetID
				if !seen[key] {
					seen[key] = true
					results = append(results, ProjectPaperEdge{
						PaperID:          e.SourceID,
						ViaConcept:       e.TargetID, // Already prefixed with "concept:"
						RelationshipType: e.RelationshipType,
						Summary:          e.Summary,
					})
				}
			}
		}
	}

	return results, nil
}
