package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/pdf"
	"github.com/spf13/cobra"
)

var openSupplement int

func init() {
	openCmd.Flags().IntVar(&openSupplement, "supplement", 0, "Open Nth supplementary PDF (1-indexed)")
	rootCmd.AddCommand(openCmd)
}

var openCmd = &cobra.Command{
	Use:   "open <id>",
	Short: "Open a paper's PDF in the configured viewer",
	Long: `Open a paper's PDF in the configured viewer.

Examples:
  bip open Ahn2026-rs
  bip open Ahn2026-rs --supplement 1`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

// OpenResult is the response for the open command.
type OpenResult struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

func runOpen(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	cfg := mustLoadConfig(repoRoot)

	// Check PDF root is configured
	if cfg.PDFRoot == "" {
		exitWithError(ExitConfigError, "pdf_root not configured (use 'bip config pdf-root /path/to/pdfs')")
	}

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Get reference
	id := args[0]
	ref, err := db.GetByID(id)
	if err != nil {
		exitWithError(ExitError, "getting reference: %v", err)
	}
	if ref == nil {
		exitWithError(ExitError, "reference not found: %s", id)
	}

	// Determine which PDF to open
	var pdfPath string
	if openSupplement > 0 {
		idx := openSupplement - 1 // Convert to 0-indexed
		if idx >= len(ref.SupplementPaths) {
			exitWithError(ExitError, "no supplement at index %d (have %d supplements)", openSupplement, len(ref.SupplementPaths))
		}
		pdfPath = ref.SupplementPaths[idx]
	} else {
		pdfPath = ref.PDFPath
		if pdfPath == "" {
			exitWithError(ExitError, "no PDF path for reference: %s", id)
		}
	}

	// Create opener and resolve path
	opener := pdf.NewOpener(cfg.PDFRoot, cfg.PDFReader)
	fullPath, err := opener.ResolvePath(pdfPath)
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	// Open the PDF
	if err := opener.Open(fullPath); err != nil {
		exitWithError(ExitError, "opening PDF: %v", err)
	}

	// Output success
	if humanOutput {
		fmt.Printf("Opening: %s\n", pdfPath)
	} else {
		outputJSON(OpenResult{
			Status: "opened",
			Path:   fullPath,
		})
	}

	return nil
}
