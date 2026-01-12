package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/export"
	"github.com/matsen/bipartite/internal/storage"
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
  bp export --bibtex
  bp export --bibtex --keys Ahn2026-rs,Gao2026-gi
  bp export --bibtex > refs.bib`,
	RunE: runExport,
}

func runExport(cmd *cobra.Command, args []string) error {
	if !exportBibtex {
		if humanOutput {
			fmt.Fprintln(os.Stderr, "error: --bibtex flag is required")
		} else {
			outputJSON(ErrorResponse{Error: "--bibtex flag is required"})
		}
		os.Exit(ExitError)
	}

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

	// Open database
	dbPath := config.DBPath(repoRoot)
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: opening database: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("opening database: %v", err)})
		}
		os.Exit(ExitError)
	}
	defer db.Close()

	var refs []storage.Reference

	if exportKeys != "" {
		// Export specific keys
		keys := strings.Split(exportKeys, ",")
		for _, key := range keys {
			key = strings.TrimSpace(key)
			ref, err := db.GetByID(key)
			if err != nil {
				if humanOutput {
					fmt.Fprintf(os.Stderr, "error: getting reference %s: %v\n", key, err)
				} else {
					outputJSON(ErrorResponse{Error: fmt.Sprintf("getting reference %s: %v", key, err)})
				}
				os.Exit(ExitError)
			}
			if ref == nil {
				if humanOutput {
					fmt.Fprintf(os.Stderr, "error: unknown key: %s\n", key)
				} else {
					outputJSON(ErrorResponse{Error: fmt.Sprintf("unknown key: %s", key)})
				}
				os.Exit(ExitError)
			}
			refs = append(refs, *ref)
		}
	} else {
		// Export all
		refs, err = db.ListAll(0)
		if err != nil {
			if humanOutput {
				fmt.Fprintf(os.Stderr, "error: listing references: %v\n", err)
			} else {
				outputJSON(ErrorResponse{Error: fmt.Sprintf("listing references: %v", err)})
			}
			os.Exit(ExitError)
		}
	}

	// Convert to BibTeX and output
	// Note: BibTeX is always text output, never JSON
	bibtex := export.ToBibTeXList(refs)
	fmt.Print(bibtex)

	return nil
}
