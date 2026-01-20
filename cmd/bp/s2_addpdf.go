package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/pdf"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	s2AddPdfLink bool
)

var s2AddPdfCmd = &cobra.Command{
	Use:   "add-pdf <pdf-path>",
	Short: "Add a paper by extracting DOI from a PDF file",
	Long: `Add a paper to the collection by extracting its DOI from a PDF file.

First attempts to extract a DOI from the PDF text. If no DOI is found,
falls back to extracting the title and searching Semantic Scholar.

Examples:
  bp s2 add-pdf ~/papers/paper.pdf
  bp s2 add-pdf ~/papers/paper.pdf --link
  bp s2 add-pdf ~/papers/paper.pdf --human`,
	Args: cobra.ExactArgs(1),
	RunE: runS2AddPdf,
}

func init() {
	s2Cmd.AddCommand(s2AddPdfCmd)
	s2AddPdfCmd.Flags().BoolVar(&s2AddPdfLink, "link", false, "Set pdf_path to the PDF file")
}

// S2AddPdfResult is the JSON output for the add-pdf command.
type S2AddPdfResult struct {
	Action    string             `json:"action"`               // added, skipped
	DOISource string             `json:"doi_source,omitempty"` // extracted, title_search
	Paper     *S2AddPaperSummary `json:"paper,omitempty"`
	Error     *S2ErrorResult     `json:"error,omitempty"`
}

func runS2AddPdf(cmd *cobra.Command, args []string) error {
	pdfPath := args[0]
	ctx := context.Background()

	// Resolve path
	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		return outputAddPdfError(ExitS2APIError, "resolving path", err)
	}

	// Check file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return outputAddPdfError(ExitS2NotFound, "PDF not found", fmt.Errorf("%s", absPath))
	}

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputAddPdfError(ExitS2APIError, "reading refs", err)
	}

	// Create resolver and client
	resolver := s2.NewLocalResolverFromRefs(refs)
	client := s2.NewClient()

	// Try to extract DOI from PDF
	doi, err := pdf.ExtractDOI(absPath)
	if err != nil {
		// Log but continue - DOI extraction can fail for various reasons
		if humanOutput {
			if errors.Is(err, pdf.ErrNoDOIFound) {
				fmt.Fprintf(os.Stderr, "No DOI found in PDF, will try title search\n")
			} else if errors.Is(err, pdf.ErrNoTextExtracted) {
				fmt.Fprintf(os.Stderr, "Warning: Could not extract text from PDF (may be scanned/image-based)\n")
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Could not extract DOI from PDF: %v\n", err)
			}
		}
	}

	var paper *s2.S2Paper
	var doiSource string

	if doi != "" {
		// Found DOI, look up in S2
		doiSource = "extracted"
		paper, err = client.GetPaper(ctx, "DOI:"+doi)
		if err != nil {
			if s2.IsNotFound(err) {
				// DOI not in S2, try title search as fallback
				doi = ""
			} else if s2.IsRateLimited(err) {
				return outputS2RateLimited(err)
			} else {
				return outputAddPdfError(ExitS2APIError, "fetching paper", err)
			}
		}
	}

	if paper == nil {
		// No DOI found or DOI lookup failed, try title search
		title, err := pdf.ExtractTitle(absPath)
		if err != nil || title == "" {
			return outputAddPdfError(ExitS2NotFound, "Could not extract DOI or title from PDF", nil)
		}

		if humanOutput {
			fmt.Fprintf(os.Stderr, "No DOI found, searching by title: %s\n", title)
		}

		// Search by title
		searchResp, err := client.SearchByTitle(ctx, title, PDFSearchLimit)
		if err != nil {
			if s2.IsRateLimited(err) {
				return outputS2RateLimited(err)
			}
			return outputAddPdfError(ExitS2APIError, "searching by title", err)
		}

		if len(searchResp.Data) == 0 {
			return outputAddPdfError(ExitS2NotFound, "No matching papers found for title", fmt.Errorf("%s", title))
		}

		if len(searchResp.Data) > 1 {
			// Multiple matches - report ambiguity
			return outputAddPdfMultipleMatches(searchResp.Data)
		}

		paper = &searchResp.Data[0]
		doiSource = "title_search"
	}

	// Map to reference
	ref := s2.MapS2ToReference(*paper)

	// Set PDF path if requested
	if s2AddPdfLink {
		ref.PDFPath = absPath
	}

	// Check for duplicates
	if paper.ExternalIDs.DOI != "" {
		if existingRef, found := resolver.FindByDOI(paper.ExternalIDs.DOI); found {
			return outputAddPdfDuplicate(existingRef.ID, paper.ExternalIDs.DOI)
		}
	}

	// Check by S2 ID
	if existingRef, found := resolver.FindByS2ID(paper.PaperID); found {
		return outputAddPdfDuplicate(existingRef.ID, "")
	}

	// Generate unique ID
	ref.ID = storage.GenerateUniqueID(refs, ref.ID)

	// Append to refs
	if err := storage.Append(refsPath, ref); err != nil {
		return outputAddPdfError(ExitS2APIError, "saving reference", err)
	}

	// Output result
	return outputAddPdfResult("added", doiSource, ref)
}

func outputAddPdfResult(action, doiSource string, ref reference.Reference) error {
	authors := formatAuthors(ref.Authors)

	result := S2AddPdfResult{
		Action:    action,
		DOISource: doiSource,
		Paper: &S2AddPaperSummary{
			ID:      ref.ID,
			DOI:     ref.DOI,
			Title:   ref.Title,
			Authors: authors,
			Year:    ref.Published.Year,
			Venue:   ref.Venue,
		},
	}

	if humanOutput {
		fmt.Printf("%s: %s\n", capitalizeFirst(action), ref.ID)
		fmt.Printf("  Title: %s\n", ref.Title)
		fmt.Printf("  Authors: %s\n", joinAuthorsDisplay(authors))
		fmt.Printf("  Year: %d\n", ref.Published.Year)
		if ref.Venue != "" {
			fmt.Printf("  Venue: %s\n", ref.Venue)
		}
		fmt.Printf("  DOI source: %s\n", doiSource)
	} else {
		outputJSON(result)
	}
	return nil
}

func outputAddPdfDuplicate(existingID, doi string) error {
	result := S2AddPdfResult{
		Action: "skipped",
		Error: &S2ErrorResult{
			Code:       "duplicate",
			Message:    "Paper already exists in collection",
			PaperID:    existingID,
			Suggestion: "Use 'bp s2 add --update' to refresh metadata",
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Paper already exists: %s\n", existingID)
		if doi != "" {
			fmt.Fprintf(os.Stderr, "  DOI: %s\n", doi)
		}
	} else {
		outputJSON(result)
	}
	os.Exit(ExitS2Duplicate)
	return nil
}

func outputAddPdfMultipleMatches(papers []s2.S2Paper) error {
	// Build list of matches for error output
	matches := make([]map[string]interface{}, 0, len(papers))
	for _, p := range papers {
		matches = append(matches, map[string]interface{}{
			"paperId": p.PaperID,
			"doi":     p.ExternalIDs.DOI,
			"title":   p.Title,
			"year":    p.Year,
		})
	}

	result := map[string]interface{}{
		"error": map[string]interface{}{
			"code":       "multiple_matches",
			"message":    "Multiple papers match the extracted title",
			"suggestion": "Use 'bp s2 add DOI:...' with the correct DOI",
			"matches":    matches,
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: Multiple papers match the title\n\n")
		fmt.Fprintf(os.Stderr, "Candidates:\n")
		for i, p := range papers {
			fmt.Fprintf(os.Stderr, "  %d. %s (%d)\n", i+1, p.Title, p.Year)
			if p.ExternalIDs.DOI != "" {
				fmt.Fprintf(os.Stderr, "     DOI: %s\n", p.ExternalIDs.DOI)
			}
		}
		fmt.Fprintf(os.Stderr, "\nUse 'bp s2 add DOI:...' with the correct DOI\n")
	} else {
		outputJSON(result)
	}
	os.Exit(ExitS2Duplicate) // Exit code 2 for multiple matches
	return nil
}

func outputAddPdfError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "error", context, err)
}
