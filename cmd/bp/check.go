package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsen/bipartite/internal/config"
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
	Issues     []CheckIssue `json:"issues"`
}

// CheckIssue represents a single issue found during check.
type CheckIssue struct {
	Type     string   `json:"type"`
	ID       string   `json:"id,omitempty"`
	IDs      []string `json:"ids,omitempty"`
	Expected string   `json:"expected,omitempty"`
	DOI      string   `json:"doi,omitempty"`
}

func runCheck(cmd *cobra.Command, args []string) error {
	root, exitCode := getRepoRoot()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	// Find repository
	repoRoot, err := config.FindRepository(root)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: err.Error()})
		}
		os.Exit(ExitConfigError)
	}

	// Load config
	cfg, err := config.Load(repoRoot)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("loading config: %v", err)})
		}
		os.Exit(ExitConfigError)
	}

	// Read all references from JSONL (source of truth)
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: reading refs: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("reading refs: %v", err)})
		}
		os.Exit(ExitDataError)
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
		pdfRoot := expandPath(cfg.PDFRoot)
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
			fmt.Printf("Repository check: OK\n\n%d references checked\n", len(refs))
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
				}
			}
			fmt.Printf("%d references checked\n", len(refs))
		}
	} else {
		outputJSON(CheckResult{
			Status:     status,
			References: len(refs),
			Issues:     issues,
		})
	}

	return nil
}

// formatIDList formats a list of IDs as a comma-separated string.
func formatIDList(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	result := ids[0]
	for i := 1; i < len(ids); i++ {
		result += ", " + ids[i]
	}
	return result
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
