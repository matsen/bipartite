package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/git"
	"github.com/matsen/bipartite/internal/pdf"
	"github.com/spf13/cobra"
)

var (
	openSupplement int
	openRecent     int
	openSince      string
)

func init() {
	openCmd.Flags().IntVar(&openSupplement, "supplement", 0, "Open Nth supplementary PDF (1-indexed)")
	openCmd.Flags().IntVar(&openRecent, "recent", 0, "Open the N most recently added papers")
	openCmd.Flags().StringVar(&openSince, "since", "", "Open papers added after this git commit")
	rootCmd.AddCommand(openCmd)
}

var openCmd = &cobra.Command{
	Use:   "open [<id>...] [flags]",
	Short: "Open papers' PDFs in the configured viewer",
	Long: `Open papers' PDFs in the configured viewer.

Supports opening multiple papers by ID, the N most recently added papers,
or papers added since a specific git commit.

Examples:
  bip open Ahn2026-rs
  bip open Ahn2026-rs Smith2024-ab Lee2024-cd
  bip open --recent 5
  bip open --since HEAD~3
  bip open --since abc123f`,
	RunE: runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	cfg := mustLoadConfig(repoRoot)

	// Check PDF root is configured
	if cfg.PDFRoot == "" {
		exitWithError(ExitConfigError, "pdf_root not configured\n  Hint: Use 'bip config pdf-root /path/to/pdfs' to set the PDF directory")
	}

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Validate mutual exclusivity
	hasIDs := len(args) > 0
	hasRecent := openRecent > 0
	hasSince := openSince != ""

	exclusiveCount := 0
	if hasIDs {
		exclusiveCount++
	}
	if hasRecent {
		exclusiveCount++
	}
	if hasSince {
		exclusiveCount++
	}

	if exclusiveCount > 1 {
		exitWithError(ExitError, "positional IDs, --recent, and --since are mutually exclusive")
	}

	if exclusiveCount == 0 {
		exitWithError(ExitError, "specify paper IDs, --recent N, or --since <commit>")
	}

	// Validate --supplement only valid with single ID
	if openSupplement > 0 && (hasRecent || hasSince || len(args) > 1) {
		exitWithError(ExitError, "--supplement is only valid with a single paper ID")
	}

	// Collect paper IDs to open
	var paperIDs []string

	if hasIDs {
		paperIDs = args
	} else if hasRecent {
		// Get N most recently added papers
		gitRoot := mustFindGitRepo(repoRoot)
		recentPapers, err := git.GetRecentlyAddedPapers(gitRoot, openRecent)
		if err != nil {
			exitWithError(ExitError, "getting recent papers: %v", err)
		}
		for _, rp := range recentPapers {
			paperIDs = append(paperIDs, rp.Reference.ID)
		}
		if len(paperIDs) == 0 {
			exitWithError(ExitError, "no recently added papers found")
		}
	} else if hasSince {
		// Get papers added since commit
		gitRoot := mustFindGitRepo(repoRoot)
		mustValidateCommit(gitRoot, openSince)
		addedPapers, err := git.GetPapersAddedSince(gitRoot, openSince)
		if err != nil {
			exitWithError(ExitError, "getting papers since %s: %v", openSince, err)
		}
		for _, rp := range addedPapers {
			paperIDs = append(paperIDs, rp.Reference.ID)
		}
		if len(paperIDs) == 0 {
			exitWithError(ExitError, "no papers added since %s", openSince)
		}
	}

	// Open papers
	opener := pdf.NewOpener(cfg.PDFRoot, cfg.PDFReader)
	var opened []OpenedPaper
	var errors []OpenError

	for _, id := range paperIDs {
		ref, err := db.GetByID(id)
		if err != nil {
			errors = append(errors, OpenError{ID: id, Error: fmt.Sprintf("getting reference: %v", err)})
			continue
		}
		if ref == nil {
			errors = append(errors, OpenError{ID: id, Error: "reference not found"})
			continue
		}

		// Determine which PDF to open
		var pdfPath string
		if openSupplement > 0 {
			idx := openSupplement - 1 // Convert to 0-indexed
			if idx >= len(ref.SupplementPaths) {
				errors = append(errors, OpenError{ID: id, Error: fmt.Sprintf("no supplement at index %d (have %d supplements)", openSupplement, len(ref.SupplementPaths))})
				continue
			}
			pdfPath = ref.SupplementPaths[idx]
		} else {
			pdfPath = ref.PDFPath
			if pdfPath == "" {
				errors = append(errors, OpenError{ID: id, Error: "no PDF path"})
				continue
			}
		}

		// Resolve and open
		fullPath, err := opener.ResolvePath(pdfPath)
		if err != nil {
			errors = append(errors, OpenError{ID: id, Error: fmt.Sprintf("PDF not found: %s", pdfPath)})
			continue
		}

		if err := opener.Open(fullPath); err != nil {
			errors = append(errors, OpenError{ID: id, Error: fmt.Sprintf("opening PDF: %v", err)})
			continue
		}

		opened = append(opened, OpenedPaper{ID: id, Path: fullPath})
	}

	// Check if at least one paper was opened
	if len(opened) == 0 {
		// All failed
		if humanOutput {
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to open any papers:\n")
			for _, e := range errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s: %s\n", e.ID, e.Error)
			}
		} else {
			outputJSON(OpenMultipleResult{Errors: errors})
		}
		return exitErrorSilent(ExitError)
	}

	// Output results
	if humanOutput {
		fmt.Printf("Opening %d paper(s):\n", len(opened))
		for _, o := range opened {
			fmt.Printf("  \u2713 %s: %s\n", o.ID, o.Path)
		}
		for _, e := range errors {
			fmt.Printf("  \u2717 %s: %s\n", e.ID, e.Error)
		}
	} else {
		outputJSON(OpenMultipleResult{
			Opened: opened,
			Errors: errors,
		})
	}

	return nil
}

// exitErrorSilent returns an error that signals the exit code without printing a message.
type silentExitError struct {
	code int
}

func (e silentExitError) Error() string {
	return ""
}

func exitErrorSilent(code int) error {
	return silentExitError{code: code}
}
