package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/importer"
	"github.com/matsen/bipartite/internal/reference"
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
  bip import --format paperpile export.json
  bip import --format paperpile export.json --dry-run

Supported formats:
  paperpile  - Paperpile JSON export`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

// ImportResult represents the result of an import operation.
type ImportResult struct {
	New     int      `json:"new"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// DryRunResult represents the result of a dry-run import.
type DryRunResult struct {
	WouldAdd    int            `json:"would_add"`
	WouldUpdate int            `json:"would_update"`
	WouldSkip   int            `json:"would_skip"`
	Details     []ImportDetail `json:"details,omitempty"`
}

// ImportDetail describes a single import action.
type ImportDetail struct {
	ID     string `json:"id"`
	Action string `json:"action"` // new, update, skip
	Title  string `json:"title"`
	Reason string `json:"reason,omitempty"`
}

// importStats tracks import operation counts.
type importStats struct {
	newCount int
	updated  int
	skipped  int
}

func runImport(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	// Validate format
	if importFormat != "paperpile" {
		exitWithError(ExitError, "unknown format: %s", importFormat)
	}

	// Parse input file
	newRefs, parseErrors := parseImportFile(args[0])

	// Load existing references
	refsPath := config.RefsPath(repoRoot)
	persistedRefs, err := storage.ReadAll(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading existing refs: %v", err)
	}

	// Process imports and classify each reference
	stats, details, resultRefs := processImports(newRefs, persistedRefs)

	// Add parse errors to skipped count
	errStrs := errorsToStrings(parseErrors)
	stats.skipped += len(parseErrors)

	// Report results (dry-run or actual)
	if importDryRun {
		reportDryRun(stats, details, errStrs)
		return nil
	}

	// Actually perform the import
	if err := persistImports(refsPath, persistedRefs, resultRefs); err != nil {
		exitWithError(ExitError, "writing refs: %v", err)
	}

	reportImportResults(stats, errStrs)
	return nil
}

// parseImportFile reads and parses the import file.
func parseImportFile(path string) ([]reference.Reference, []error) {
	data, err := os.ReadFile(path)
	if err != nil {
		exitWithError(ExitError, "reading file: %v", err)
	}

	newRefs, parseErrors := importer.ParsePaperpile(data)
	if len(parseErrors) > 0 && len(newRefs) == 0 {
		exitWithError(ExitDataError, "failed to parse any references: %v", parseErrors[0])
	}

	return newRefs, parseErrors
}

// processImports classifies each reference and builds the action list.
func processImports(newRefs, persistedRefs []reference.Reference) (importStats, []ImportDetail, []storage.RefWithAction) {
	// Build a working set that includes both persisted refs AND in-progress imports.
	// This enables deduplication within a single import batch.
	workingRefSet := make([]reference.Reference, len(persistedRefs))
	copy(workingRefSet, persistedRefs)

	var stats importStats
	var details []ImportDetail
	var resultRefs []storage.RefWithAction

	for _, newRef := range newRefs {
		action := classifyImport(workingRefSet, newRef)

		switch action.action {
		case "new":
			newRef.ID = storage.GenerateUniqueID(workingRefSet, newRef.ID)
			resultRefs = append(resultRefs, storage.RefWithAction{Ref: newRef, Action: "new"})
			workingRefSet = append(workingRefSet, newRef)
			stats.newCount++
		case "update":
			// If existingIdx is within persistedRefs bounds, it's a match against
			// an already-persisted reference. Otherwise, it matched something we
			// added earlier in this same import batch (workingRefSet grows as we go).
			if action.existingIdx < len(persistedRefs) {
				resultRefs = append(resultRefs, storage.RefWithAction{Ref: newRef, Action: "update", ExistingIdx: action.existingIdx})
				stats.updated++
			} else {
				// DOI/ID match within batch - skip as duplicate
				stats.skipped++
				action.action = "skip"
				action.reason = "duplicate_in_batch"
			}
		case "skip":
			stats.skipped++
		}

		details = append(details, ImportDetail{
			ID:     newRef.ID,
			Action: action.action,
			Title:  truncateString(newRef.Title, ImportTitleMaxLen),
			Reason: action.reason,
		})
	}

	return stats, details, resultRefs
}

// errorsToStrings converts a slice of errors to strings.
func errorsToStrings(errs []error) []string {
	strs := make([]string, len(errs))
	for i, e := range errs {
		strs[i] = e.Error()
	}
	return strs
}

// reportDryRun outputs the dry-run results.
func reportDryRun(stats importStats, details []ImportDetail, errStrs []string) {
	if humanOutput {
		fmt.Println("Dry run - would import from Paperpile export...")
		fmt.Printf("  Would add:    %d new references\n", stats.newCount)
		fmt.Printf("  Would update: %d existing references (matched by DOI or ID)\n", stats.updated)
		fmt.Printf("  Would skip:   %d (errors or duplicates)\n", stats.skipped)
		if len(errStrs) > 0 {
			fmt.Println("\nParse errors:")
			for _, e := range errStrs {
				fmt.Printf("  - %s\n", e)
			}
		}
	} else {
		outputJSON(DryRunResult{
			WouldAdd:    stats.newCount,
			WouldUpdate: stats.updated,
			WouldSkip:   stats.skipped,
			Details:     details,
		})
	}
}

// reportImportResults outputs the actual import results.
func reportImportResults(stats importStats, errStrs []string) {
	if humanOutput {
		fmt.Println("Imported from Paperpile export:")
		fmt.Printf("  Added:   %d new references\n", stats.newCount)
		fmt.Printf("  Updated: %d existing references (matched by DOI or ID)\n", stats.updated)
		fmt.Printf("  Skipped: %d (errors or duplicates)\n", stats.skipped)
		if len(errStrs) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range errStrs {
				fmt.Printf("  - %s\n", e)
			}
		}
	} else {
		outputJSON(ImportResult{
			New:     stats.newCount,
			Updated: stats.updated,
			Skipped: stats.skipped,
			Errors:  errStrs,
		})
	}
}

type importAction struct {
	action      string // new, update, skip
	reason      string
	existingIdx int
}

// classifyImport determines what to do with an incoming reference.
// Panics if newRef has an empty ID, as this indicates a bug in the parser.
func classifyImport(existing []reference.Reference, newRef reference.Reference) importAction {
	// Fail-fast validation: every reference must have an ID
	if newRef.ID == "" {
		panic("classifyImport called with empty ID - parser bug")
	}

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

	// Check for ID match (secondary deduplication)
	if idx, found := storage.FindByID(existing, newRef.ID); found {
		return importAction{
			action:      "update",
			reason:      "id_match",
			existingIdx: idx,
		}
	}

	// No match - genuinely new
	return importAction{action: "new"}
}

// persistImports writes the import results to the refs file.
func persistImports(path string, existing []reference.Reference, actions []storage.RefWithAction) error {
	// Build new refs list
	newRefs := make([]reference.Reference, len(existing))
	copy(newRefs, existing)

	// Apply updates first
	for _, a := range actions {
		if a.Action == "update" {
			newRefs[a.ExistingIdx] = a.Ref
		}
	}

	// Append new entries
	for _, a := range actions {
		if a.Action == "new" {
			newRefs = append(newRefs, a.Ref)
		}
	}

	return storage.WriteAll(path, newRefs)
}
