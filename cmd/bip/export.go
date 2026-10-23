package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matsen/bipartite/internal/export"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	exportBibtex bool
	exportKeys   string
	exportAppend string
)

func init() {
	exportCmd.Flags().BoolVar(&exportBibtex, "bibtex", false, "Export to BibTeX format")
	exportCmd.Flags().StringVar(&exportKeys, "keys", "", "Export only specified IDs (comma-separated) [deprecated: use positional args]")
	exportCmd.Flags().StringVar(&exportAppend, "append", "", "Append to existing .bib file (with deduplication)")
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export [<id>...] [flags]",
	Short: "Export references to BibTeX format",
	Long: `Export references to BibTeX format.

Without IDs, exports all papers. With IDs, exports only specified papers.
Use --append to add to an existing .bib file with automatic deduplication.

Examples:
  bip export --bibtex
  bip export --bibtex Smith2024-ab Lee2024-cd
  bip export --bibtex --keys Ahn2026-rs,Gao2026-gi  # deprecated
  bip export --bibtex > refs.bib
  bip export --bibtex --append refs.bib Smith2024-ab`,
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	if !exportBibtex {
		exitWithError(ExitError, "--bibtex flag is required")
	}

	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	var refs []reference.Reference
	var err error

	// Collect IDs from positional args and --keys flag
	var ids []string
	if len(args) > 0 {
		ids = args
	}
	if exportKeys != "" {
		// Support legacy --keys flag
		keys := strings.Split(exportKeys, ",")
		for _, key := range keys {
			key = strings.TrimSpace(key)
			if key != "" {
				ids = append(ids, key)
			}
		}
	}

	if len(ids) > 0 {
		// Export specific IDs
		for _, id := range ids {
			ref, err := db.GetByID(id)
			if err != nil {
				exitWithError(ExitError, "getting reference %s: %v", id, err)
			}
			if ref == nil {
				exitWithError(ExitError, "unknown key: %s\n  Hint: Use 'bip list' to see available references", id)
			}
			refs = append(refs, *ref)
		}
	} else {
		// Export all
		refs, err = db.ListAll(0)
		if err != nil {
			exitWithError(ExitError, "listing references: %v", err)
		}
	}

	warnSuspectPages(refs)
	warnAllCapsTitles(refs)
	lex := mustTitleLexicon(db)

	// Handle --append mode
	if exportAppend != "" {
		return runExportAppend(refs, exportAppend, lex)
	}

	// Standard export to stdout (no JSON wrapper)
	bibtex := export.ToBibTeXList(refs, lex)
	fmt.Print(bibtex)

	return nil
}

// warnSuspectPages prints a stderr warning for each reference whose Pages
// looks like a Paperpile advance-access placeholder rather than real
// pagination. Stdout (the BibTeX itself) is left untouched.
func warnSuspectPages(refs []reference.Reference) {
	for _, ref := range refs {
		if export.SuspectPages(ref) {
			fmt.Fprintf(os.Stderr, "warning: %s: pages %q with no volume/issue — likely a Paperpile placeholder from an advance-access import, verify at the publisher before citing\n", ref.ID, ref.Pages)
		}
	}
}

// mustTitleLexicon learns title proper nouns from the whole library, whatever subset is exported.
func mustTitleLexicon(db *storage.DB) *export.Lexicon {
	all, err := db.ListAll(0)
	if err != nil {
		exitWithError(ExitError, "listing references: %v", err)
	}
	titles := make([]string, len(all))
	for i, ref := range all {
		titles[i] = ref.Title
	}
	return export.NewLexicon(titles)
}

// warnAllCapsTitles prints a stderr warning for each reference whose title is
// in capitals: the style will lowercase all of it, acronyms included.
func warnAllCapsTitles(refs []reference.Reference) {
	for _, ref := range refs {
		if _, allCaps := export.ProtectTitle(ref.Title, nil); allCaps {
			fmt.Fprintf(os.Stderr, "warning: %s: title is in capitals, so nothing in it is protected from lowercasing — check acronyms and proper nouns\n", ref.ID)
		}
	}
}

func runExportAppend(refs []reference.Reference, outputPath string, lex *export.Lexicon) error {
	// Make path absolute for output
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		exitWithError(ExitError, "resolving path: %v", err)
	}

	// Parse existing .bib file for deduplication
	idx, err := export.ParseBibTeXFile(absPath)
	if err != nil {
		exitWithError(ExitError, "cannot read file: %s\n  Hint: Check file exists and has read permissions", outputPath)
	}

	var toExport []reference.Reference
	var skippedIDs []string

	for _, ref := range refs {
		if idx.HasEntry(ref.ID, ref.DOI) {
			skippedIDs = append(skippedIDs, ref.ID)
		} else {
			toExport = append(toExport, ref)
		}
	}

	// Append non-duplicate entries
	if len(toExport) > 0 {
		bibtex := export.ToBibTeXList(toExport, lex)
		if err := export.AppendToBibFile(absPath, bibtex); err != nil {
			exitWithError(ExitError, "cannot write to file: %s\n  Hint: Check file has write permissions", outputPath)
		}
	}

	// Output result as JSON
	result := ExportResult{
		Exported:   len(toExport),
		Skipped:    len(skippedIDs),
		SkippedIDs: skippedIDs,
		OutputPath: absPath,
	}

	if humanOutput {
		fmt.Printf("Exported %d entries to %s\n", result.Exported, result.OutputPath)
		if result.Skipped > 0 {
			fmt.Printf("Skipped %d duplicates: %s\n", result.Skipped, strings.Join(result.SkippedIDs, ", "))
		}
	} else {
		outputJSON(result)
	}

	return nil
}
