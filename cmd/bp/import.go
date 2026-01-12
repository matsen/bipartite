package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/importer"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	importFormat string
	importDryRun bool
)

func init() {
	importCmd.Flags().StringVar(&importFormat, "format", "", "Import format (paperpile)")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be imported without writing")
	importCmd.MarkFlagRequired("format")
	rootCmd.AddCommand(importCmd)
}

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import references from an external format",
	Long: `Import references from an external format.

Usage:
  bp import --format paperpile export.json
  bp import --format paperpile export.json --dry-run

Supported formats:
  paperpile  - Paperpile JSON export`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

// ImportResult represents the result of an import operation.
type ImportResult struct {
	Imported int      `json:"imported"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// DryRunResult represents the result of a dry-run import.
type DryRunResult struct {
	WouldImport int            `json:"would_import"`
	WouldUpdate int            `json:"would_update"`
	WouldSkip   int            `json:"would_skip"`
	Details     []ImportDetail `json:"details,omitempty"`
}

// ImportDetail describes a single import action.
type ImportDetail struct {
	ID     string `json:"id"`
	Action string `json:"action"` // import, update, skip
	Title  string `json:"title"`
	Reason string `json:"reason,omitempty"`
}

func runImport(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	// Validate format
	if importFormat != "paperpile" {
		exitWithError(ExitError, "unknown format: %s", importFormat)
	}

	// Read input file
	inputPath := args[0]
	data, err := os.ReadFile(inputPath)
	if err != nil {
		exitWithError(ExitError, "reading file: %v", err)
	}

	// Parse references
	newRefs, parseErrors := importer.ParsePaperpile(data)
	if len(parseErrors) > 0 && len(newRefs) == 0 {
		// Only fatal if no refs were parsed
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: failed to parse any references\n")
			for _, e := range parseErrors {
				fmt.Fprintf(os.Stderr, "  - %s\n", e.Error())
			}
		} else {
			outputJSON(ErrorResponse{Error: "failed to parse any references"})
		}
		os.Exit(ExitDataError)
	}

	// Load existing references (persisted in repository)
	refsPath := config.RefsPath(repoRoot)
	persistedRefs, err := storage.ReadAll(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading existing refs: %v", err)
	}

	// Build a working set that includes in-progress imports for deduplication.
	// This set starts with persisted refs and grows as we process imports.
	dedupeCheckRefs := make([]storage.Reference, len(persistedRefs))
	copy(dedupeCheckRefs, persistedRefs)

	var imported, updated, skipped int
	var details []ImportDetail
	var resultRefs []storage.RefWithAction

	for _, newRef := range newRefs {
		// Use dedupeCheckRefs to check DOI matches against both existing and newly imported refs
		action := classifyImport(dedupeCheckRefs, newRef)

		switch action.action {
		case "import":
			// Check for ID collision and generate unique ID if needed
			newRef.ID = storage.GenerateUniqueID(dedupeCheckRefs, newRef.ID)
			resultRefs = append(resultRefs, storage.RefWithAction{Ref: newRef, Action: "import"})
			// Add to dedupeCheckRefs so subsequent imports see this ID and DOI as taken
			dedupeCheckRefs = append(dedupeCheckRefs, newRef)
			imported++
		case "update":
			// For updates, determine if it's updating an existing ref or one from this batch
			if action.existingIdx < len(persistedRefs) {
				// Updating an existing reference in the repo
				resultRefs = append(resultRefs, storage.RefWithAction{Ref: newRef, Action: "update", ExistingIdx: action.existingIdx})
				updated++
			} else {
				// DOI match against a ref already imported in this batch - skip as duplicate
				skipped++
				action.action = "skip"
				action.reason = "duplicate_in_batch"
			}
		case "skip":
			skipped++
		}

		details = append(details, ImportDetail{
			ID:     newRef.ID,
			Action: action.action,
			Title:  truncateString(newRef.Title, TitleTruncateLen),
			Reason: action.reason,
		})
	}

	// Convert parse errors to strings
	errStrs := make([]string, len(parseErrors))
	for i, e := range parseErrors {
		errStrs[i] = e.Error()
	}
	skipped += len(parseErrors)

	// Dry run: just report what would happen
	if importDryRun {
		if humanOutput {
			fmt.Println("Dry run - would import from Paperpile export...")
			fmt.Printf("  Would import: %d new references\n", imported)
			fmt.Printf("  Would update: %d existing references (matched by DOI)\n", updated)
			fmt.Printf("  Would skip:   %d (errors or duplicates)\n", skipped)
			if len(parseErrors) > 0 {
				fmt.Println("\nParse errors:")
				for _, e := range errStrs {
					fmt.Printf("  - %s\n", e)
				}
			}
		} else {
			outputJSON(DryRunResult{
				WouldImport: imported,
				WouldUpdate: updated,
				WouldSkip:   skipped,
				Details:     details,
			})
		}
		return nil
	}

	// Actually perform the import
	if err := applyImports(refsPath, persistedRefs, resultRefs); err != nil {
		exitWithError(ExitError, "writing refs: %v", err)
	}

	// Output results
	if humanOutput {
		fmt.Println("Importing from Paperpile export...")
		fmt.Printf("  Imported: %d new references\n", imported)
		fmt.Printf("  Updated:  %d existing references (matched by DOI)\n", updated)
		fmt.Printf("  Skipped:  %d (errors or duplicates)\n", skipped)
		if len(parseErrors) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range errStrs {
				fmt.Printf("  - %s\n", e)
			}
		}
	} else {
		outputJSON(ImportResult{
			Imported: imported,
			Updated:  updated,
			Skipped:  skipped,
			Errors:   errStrs,
		})
	}

	return nil
}

type importAction struct {
	action      string // import, update, skip
	reason      string
	existingIdx int
}

// classifyImport determines what to do with an incoming reference.
func classifyImport(existing []storage.Reference, newRef storage.Reference) importAction {
	// Check for DOI match first (primary deduplication)
	if newRef.DOI != "" {
		if idx, found := storage.FindByDOI(existing, newRef.DOI); found {
			return importAction{
				action:      "update",
				reason:      "doi_match",
				existingIdx: idx,
			}
		}
	}

	// No DOI match means new import
	return importAction{action: "import"}
}

// applyImports writes the import results to the refs file.
func applyImports(path string, existing []storage.Reference, actions []storage.RefWithAction) error {
	// Build new refs list
	newRefs := make([]storage.Reference, len(existing))
	copy(newRefs, existing)

	// Apply updates first
	for _, a := range actions {
		if a.Action == "update" {
			newRefs[a.ExistingIdx] = a.Ref
		}
	}

	// Append new imports
	for _, a := range actions {
		if a.Action == "import" {
			newRefs = append(newRefs, a.Ref)
		}
	}

	return storage.WriteAll(path, newRefs)
}
