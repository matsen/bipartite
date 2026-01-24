package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/edge"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify repository integrity",
	Long:  `Verify repository integrity, checking for missing PDFs and duplicate DOIs.`,
	RunE:  runCheck,
}

// CheckResult is the response for the check command.
type CheckResult struct {
	Status     string       `json:"status"`
	References int          `json:"references"`
	Edges      int          `json:"edges"`
	Projects   int          `json:"projects"`
	Repos      int          `json:"repos"`
	Issues     []CheckIssue `json:"issues"`
}

// CheckIssue represents a single issue found during check.
type CheckIssue struct {
	Type     string   `json:"type"`
	ID       string   `json:"id,omitempty"`
	IDs      []string `json:"ids,omitempty"`
	Expected string   `json:"expected,omitempty"`
	DOI      string   `json:"doi,omitempty"`
	SourceID string   `json:"source_id,omitempty"`
	TargetID string   `json:"target_id,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

func runCheck(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	cfg := mustLoadConfig(repoRoot)

	// Read all references from JSONL (source of truth)
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading refs: %v", err)
	}

	// Build set of valid paper IDs for edge checking
	validIDs := make(map[string]bool)
	for _, ref := range refs {
		validIDs[ref.ID] = true
	}

	var issues []CheckIssue

	// Check for duplicate DOIs
	doiMap := make(map[string][]string) // DOI -> list of IDs
	for _, ref := range refs {
		if ref.DOI != "" {
			doiMap[ref.DOI] = append(doiMap[ref.DOI], ref.ID)
		}
	}
	for doi, ids := range doiMap {
		if len(ids) > 1 {
			issues = append(issues, CheckIssue{
				Type: "duplicate_doi",
				IDs:  ids,
				DOI:  doi,
			})
		}
	}

	// Check for missing PDFs (only if pdf_root is configured)
	if cfg.PDFRoot != "" {
		pdfRoot := config.ExpandPath(cfg.PDFRoot)
		for _, ref := range refs {
			if ref.PDFPath != "" {
				fullPath := filepath.Join(pdfRoot, ref.PDFPath)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					issues = append(issues, CheckIssue{
						Type:     "missing_pdf",
						ID:       ref.ID,
						Expected: ref.PDFPath,
					})
				}
			}
		}
	}

	// Check edges for integrity
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil && !os.IsNotExist(err) {
		exitWithError(ExitDataError, "reading edges: %v", err)
	}

	// Check for orphaned edges using shared detection function
	orphaned, _ := edge.DetectOrphanedEdges(edges, validIDs)
	for _, o := range orphaned {
		issues = append(issues, CheckIssue{
			Type:     "orphaned_edge",
			SourceID: o.SourceID,
			TargetID: o.TargetID,
			Reason:   o.Reason,
		})
	}

	// Check for duplicate edges using shared detection function
	duplicates := edge.FindDuplicateEdges(edges)
	for key, count := range duplicates {
		issues = append(issues, CheckIssue{
			Type:     "duplicate_edge",
			SourceID: key.SourceID,
			TargetID: key.TargetID,
			Reason:   fmt.Sprintf("type=%s, count=%d", key.RelationshipType, count),
		})
	}

	// Load projects and repos
	projectsPath := config.ProjectsPath(repoRoot)
	projects, err := storage.ReadAllProjects(projectsPath)
	if err != nil && !os.IsNotExist(err) {
		exitWithError(ExitDataError, "reading projects: %v", err)
	}

	reposPath := config.ReposPath(repoRoot)
	repos, err := storage.ReadAllRepos(reposPath)
	if err != nil && !os.IsNotExist(err) {
		exitWithError(ExitDataError, "reading repos: %v", err)
	}

	// Build project ID set for validation
	projectIDs := make(map[string]bool)
	for _, p := range projects {
		projectIDs[p.ID] = true
	}

	// T073: Check that repos reference valid projects
	for _, r := range repos {
		if !projectIDs[r.Project] {
			issues = append(issues, CheckIssue{
				Type:   "orphaned_repo",
				ID:     r.ID,
				Reason: fmt.Sprintf("references non-existent project %q", r.Project),
			})
		}
	}

	// Build concept ID set for validation
	conceptsPath := config.ConceptsPath(repoRoot)
	conceptIDs, err := storage.LoadConceptIDSet(conceptsPath)
	if err != nil && !os.IsNotExist(err) {
		exitWithError(ExitDataError, "reading concepts: %v", err)
	}
	if conceptIDs == nil {
		conceptIDs = make(map[string]bool)
	}

	// T074: Check for orphaned project edges (project deleted but edge remains)
	// T075: Check for invalid paper↔project or *↔repo edges
	for _, e := range edges {
		sourceType, sourceBareID := parseNodeType(e.SourceID)
		targetType, targetBareID := parseNodeType(e.TargetID)

		// Check for repo edges (invalid - repos have no edges)
		if sourceType == "repo" {
			issues = append(issues, CheckIssue{
				Type:     "invalid_repo_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   "source is a repo (repos have no edges)",
			})
		}
		if targetType == "repo" {
			issues = append(issues, CheckIssue{
				Type:     "invalid_repo_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   "target is a repo (repos have no edges)",
			})
		}

		// Check for direct paper↔project edges (invalid - must go through concept)
		if (sourceType == "paper" && targetType == "project") || (sourceType == "project" && targetType == "paper") {
			issues = append(issues, CheckIssue{
				Type:     "invalid_paper_project_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   "paper↔project edges must go through concept",
			})
		}

		// Check for orphaned project references
		if sourceType == "project" && !projectIDs[sourceBareID] {
			issues = append(issues, CheckIssue{
				Type:     "orphaned_project_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   fmt.Sprintf("source project %q does not exist", sourceBareID),
			})
		}
		if targetType == "project" && !projectIDs[targetBareID] {
			issues = append(issues, CheckIssue{
				Type:     "orphaned_project_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   fmt.Sprintf("target project %q does not exist", targetBareID),
			})
		}

		// Check for orphaned concept references in edges
		if sourceType == "concept" && !conceptIDs[sourceBareID] {
			issues = append(issues, CheckIssue{
				Type:     "orphaned_concept_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   fmt.Sprintf("source concept %q does not exist", sourceBareID),
			})
		}
		if targetType == "concept" && !conceptIDs[targetBareID] {
			issues = append(issues, CheckIssue{
				Type:     "orphaned_concept_edge",
				SourceID: e.SourceID,
				TargetID: e.TargetID,
				Reason:   fmt.Sprintf("target concept %q does not exist", targetBareID),
			})
		}
	}

	// Determine status
	status := "ok"
	if len(issues) > 0 {
		status = "issues"
	}

	// Ensure issues is an empty array, not null
	if issues == nil {
		issues = []CheckIssue{}
	}

	// Output results
	if humanOutput {
		if len(issues) == 0 {
			fmt.Printf("Repository check: OK\n\n%d references, %d edges, %d projects, %d repos checked\n", len(refs), len(edges), len(projects), len(repos))
		} else {
			fmt.Printf("Repository check: %d issues found\n\n", len(issues))
			for _, issue := range issues {
				switch issue.Type {
				case "missing_pdf":
					fmt.Printf("  [WARN] Missing PDF for %s\n", issue.ID)
					fmt.Printf("         Expected: %s\n\n", issue.Expected)
				case "duplicate_doi":
					fmt.Printf("  [WARN] Duplicate DOI %s\n", issue.DOI)
					fmt.Printf("         Found in: %s\n\n", formatIDList(issue.IDs))
				case "orphaned_edge":
					fmt.Printf("  [WARN] Orphaned edge: %s --> %s (%s)\n\n", issue.SourceID, issue.TargetID, issue.Reason)
				case "duplicate_edge":
					fmt.Printf("  [WARN] Duplicate edge: %s --> %s (%s)\n\n", issue.SourceID, issue.TargetID, issue.Reason)
				case "orphaned_repo":
					fmt.Printf("  [WARN] Orphaned repo: %s (%s)\n\n", issue.ID, issue.Reason)
				case "invalid_repo_edge":
					fmt.Printf("  [WARN] Invalid repo edge: %s --> %s (%s)\n\n", issue.SourceID, issue.TargetID, issue.Reason)
				case "invalid_paper_project_edge":
					fmt.Printf("  [WARN] Invalid paper-project edge: %s --> %s (%s)\n\n", issue.SourceID, issue.TargetID, issue.Reason)
				case "orphaned_project_edge":
					fmt.Printf("  [WARN] Orphaned project edge: %s --> %s (%s)\n\n", issue.SourceID, issue.TargetID, issue.Reason)
				case "orphaned_concept_edge":
					fmt.Printf("  [WARN] Orphaned concept edge: %s --> %s (%s)\n\n", issue.SourceID, issue.TargetID, issue.Reason)
				}
			}
			fmt.Printf("%d references, %d edges, %d projects, %d repos checked\n", len(refs), len(edges), len(projects), len(repos))
		}
	} else {
		outputJSON(CheckResult{
			Status:     status,
			References: len(refs),
			Edges:      len(edges),
			Projects:   len(projects),
			Repos:      len(repos),
			Issues:     issues,
		})
	}

	return nil
}
