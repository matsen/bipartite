package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/pdf"
	"github.com/matsen/bipartite/internal/storage"
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
  bp open Ahn2026-rs
  bp open Ahn2026-rs --supplement 1`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

// OpenResult is the response for the open command.
type OpenResult struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

func runOpen(cmd *cobra.Command, args []string) error {
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

	// Check PDF root is configured
	if cfg.PDFRoot == "" {
		if humanOutput {
			fmt.Fprintln(os.Stderr, "error: pdf_root not configured (use 'bp config pdf-root /path/to/pdfs')")
		} else {
			outputJSON(ErrorResponse{Error: "pdf_root not configured"})
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

	// Get reference
	id := args[0]
	ref, err := db.GetByID(id)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: getting reference: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("getting reference: %v", err)})
		}
		os.Exit(ExitError)
	}
	if ref == nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: reference not found: %s\n", id)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("reference not found: %s", id)})
		}
		os.Exit(ExitError)
	}

	// Determine which PDF to open
	var pdfPath string
	if openSupplement > 0 {
		idx := openSupplement - 1 // Convert to 0-indexed
		if idx >= len(ref.SupplementPaths) {
			if humanOutput {
				fmt.Fprintf(os.Stderr, "error: no supplement at index %d (have %d supplements)\n", openSupplement, len(ref.SupplementPaths))
			} else {
				outputJSON(ErrorResponse{Error: fmt.Sprintf("no supplement at index %d", openSupplement)})
			}
			os.Exit(ExitError)
		}
		pdfPath = ref.SupplementPaths[idx]
	} else {
		pdfPath = ref.PDFPath
		if pdfPath == "" {
			if humanOutput {
				fmt.Fprintf(os.Stderr, "error: no PDF path for reference: %s\n", id)
			} else {
				outputJSON(ErrorResponse{Error: fmt.Sprintf("no PDF path for reference: %s", id)})
			}
			os.Exit(ExitError)
		}
	}

	// Create opener and resolve path
	opener := pdf.NewOpener(cfg.PDFRoot, cfg.PDFReader)
	fullPath, err := opener.ResolvePath(pdfPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: err.Error()})
		}
		os.Exit(ExitError)
	}

	// Open the PDF
	if err := opener.Open(fullPath); err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: opening PDF: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("opening PDF: %v", err)})
		}
		os.Exit(ExitError)
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
