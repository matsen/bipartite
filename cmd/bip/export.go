package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/export"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/spf13/cobra"
)

var (
	exportBibtex bool
	exportKeys   string
)

func init() {
	exportCmd.Flags().BoolVar(&exportBibtex, "bibtex", false, "Export to BibTeX format")
	exportCmd.Flags().StringVar(&exportKeys, "keys", "", "Export only specified IDs (comma-separated)")
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export references to BibTeX format",
	Long: `Export references to BibTeX format.

Examples:
  bip export --bibtex
  bip export --bibtex --keys Ahn2026-rs,Gao2026-gi
  bip export --bibtex > refs.bib`,
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

	if exportKeys != "" {
		// Export specific keys
		keys := strings.Split(exportKeys, ",")
		for _, key := range keys {
			key = strings.TrimSpace(key)
			ref, err := db.GetByID(key)
			if err != nil {
				exitWithError(ExitError, "getting reference %s: %v", key, err)
			}
			if ref == nil {
				exitWithError(ExitError, "unknown key: %s", key)
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

	// Convert to BibTeX and output
	// Note: BibTeX is always text output, never JSON
	bibtex := export.ToBibTeXList(refs)
	fmt.Print(bibtex)

	return nil
}
