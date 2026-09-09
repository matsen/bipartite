package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify repository integrity",
	Long:  `Verify repository integrity, checking for missing PDFs and duplicate DOIs (normalized, case-insensitive).`,
	RunE:  runCheck,
}

// CheckResult is the response for the check command.
type CheckResult struct {
	Status     string       `json:"status"`
	References int          `json:"references"`
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
	Reason   string   `json:"reason,omitempty"`
}

// findDuplicateDOIs groups references by normalized DOI and returns a
// duplicate_doi issue for every DOI held by 2+ references. Issues are sorted
// by DOI so output is stable across runs.
func findDuplicateDOIs(refs []reference.Reference) []CheckIssue {
	doiMap := make(map[string][]string) // normalized DOI -> list of IDs
	for _, ref := range refs {
		key := s2.NormalizeDOI(ref.DOI)
		if key == "" {
			continue
		}
		doiMap[key] = append(doiMap[key], ref.ID)
	}

	var dois []string
	for doi, ids := range doiMap {
		if len(ids) > 1 {
			dois = append(dois, doi)
		}
	}
	sort.Strings(dois)

	issues := make([]CheckIssue, 0, len(dois))
	for _, doi := range dois {
		issues = append(issues, CheckIssue{
			Type: "duplicate_doi",
			IDs:  doiMap[doi],
			DOI:  doi,
		})
	}
	return issues
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

	var issues []CheckIssue

	// Check for duplicate DOIs
	issues = append(issues, findDuplicateDOIs(refs)...)

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
			fmt.Printf("Repository check: OK\n\n%d references, %d projects, %d repos checked\n", len(refs), len(projects), len(repos))
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
				case "orphaned_repo":
					fmt.Printf("  [WARN] Orphaned repo: %s (%s)\n\n", issue.ID, issue.Reason)
				}
			}
			fmt.Printf("%d references, %d projects, %d repos checked\n", len(refs), len(projects), len(repos))
		}
	} else {
		outputJSON(CheckResult{
			Status:     status,
			References: len(refs),
			Projects:   len(projects),
			Repos:      len(repos),
			Issues:     issues,
		})
	}

	return nil
}
