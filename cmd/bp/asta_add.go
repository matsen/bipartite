package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	astaAddUpdate bool
	astaAddLink   string
)

var astaAddCmd = &cobra.Command{
	Use:   "add <paper-id>",
	Short: "Add a paper by fetching metadata from Semantic Scholar",
	Long: `Add a paper to the collection by fetching its metadata from Semantic Scholar.

Supported paper ID formats:
  DOI:10.1038/nature12373      DOI
  ARXIV:2106.15928             arXiv ID
  PMID:19872477                PubMed ID
  PMCID:2323736                PubMed Central ID
  CorpusId:215416146           S2 Corpus ID
  <40-char-hex>                Raw S2 paper ID

Examples:
  bp asta add DOI:10.1038/nature12373
  bp asta add ARXIV:2106.15928 --link ~/papers/paper.pdf
  bp asta add DOI:10.1093/sysbio/syy032 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runAstaAdd,
}

func init() {
	astaCmd.AddCommand(astaAddCmd)
	astaAddCmd.Flags().BoolVarP(&astaAddUpdate, "update", "u", false, "Update metadata if paper already exists")
	astaAddCmd.Flags().StringVarP(&astaAddLink, "link", "l", "", "Set pdf_path to the given file path")
}

// AstaAddResult is the JSON output for the add command.
type AstaAddResult struct {
	Action string               `json:"action"` // added, updated, skipped
	Paper  *AstaAddPaperSummary `json:"paper,omitempty"`
	Error  *AstaErrorResult     `json:"error,omitempty"`
}

// AstaAddPaperSummary is a summary of the added paper.
type AstaAddPaperSummary struct {
	ID      string   `json:"id"`
	DOI     string   `json:"doi,omitempty"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Year    int      `json:"year"`
	Venue   string   `json:"venue,omitempty"`
}

// AstaErrorResult is the JSON output for errors.
type AstaErrorResult struct {
	Code       string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	RetryAfter int    `json:"retry_after,omitempty"`
	PaperID    string `json:"paper_id,omitempty"`
}

func runAstaAdd(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputAstaError(ExitAstaAPIError, "reading refs", err)
	}

	// Create resolver and client
	resolver := asta.NewLocalResolverFromRefs(refs)
	client := asta.NewClient()

	// Parse and resolve paper ID
	parsed := asta.ParsePaperID(paperID)
	var s2ID string
	if parsed.IsExternalID() {
		s2ID = parsed.String()
	} else {
		// Try to resolve local ID
		resolved, _, err := resolver.ResolveToS2ID(paperID)
		if err != nil {
			return outputAstaNotFound(paperID, "Local paper ID not found")
		}
		s2ID = resolved
	}

	// Fetch paper from S2
	paper, err := client.GetPaper(ctx, s2ID)
	if err != nil {
		if asta.IsNotFound(err) {
			return outputAstaNotFound(paperID, "Paper not found in Semantic Scholar")
		}
		if asta.IsRateLimited(err) {
			return outputAstaRateLimited(err)
		}
		return outputAstaError(ExitAstaAPIError, "fetching paper", err)
	}

	// Map to reference
	ref := asta.MapS2ToReference(*paper)

	// Set PDF path if requested
	if astaAddLink != "" {
		ref.PDFPath = astaAddLink
	}

	// Check for duplicates
	if paper.ExternalIDs.DOI != "" {
		if existingRef, found := resolver.FindByDOI(paper.ExternalIDs.DOI); found {
			if !astaAddUpdate {
				return outputAstaDuplicate(existingRef.ID, paper.ExternalIDs.DOI)
			}
			// Update existing
			return updateExistingPaper(refsPath, refs, existingRef.ID, ref)
		}
	}

	// Check by S2 ID
	if existingRef, found := resolver.FindByS2ID(paper.PaperID); found {
		if !astaAddUpdate {
			return outputAstaDuplicate(existingRef.ID, "")
		}
		// Update existing
		return updateExistingPaper(refsPath, refs, existingRef.ID, ref)
	}

	// Generate unique ID
	ref.ID = storage.GenerateUniqueID(refs, ref.ID)

	// Append to refs
	if err := storage.Append(refsPath, ref); err != nil {
		return outputAstaError(ExitAstaAPIError, "saving reference", err)
	}

	// Output result
	return outputAstaAddResult("added", ref)
}

func updateExistingPaper(refsPath string, refs []reference.Reference, existingID string, newRef reference.Reference) error {
	// Find and update the existing reference
	for i, ref := range refs {
		if ref.ID == existingID {
			// Preserve the original ID
			newRef.ID = existingID
			// Preserve PDF path if not being updated
			if astaAddLink == "" && ref.PDFPath != "" {
				newRef.PDFPath = ref.PDFPath
			}
			refs[i] = newRef
			break
		}
	}

	// Write all refs
	if err := storage.WriteAll(refsPath, refs); err != nil {
		return outputAstaError(ExitAstaAPIError, "saving reference", err)
	}

	return outputAstaAddResult("updated", newRef)
}

func outputAstaAddResult(action string, ref reference.Reference) error {
	authors := formatAuthors(ref.Authors)

	result := AstaAddResult{
		Action: action,
		Paper: &AstaAddPaperSummary{
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
	} else {
		outputJSON(result)
	}
	return nil
}

func outputAstaNotFound(paperID, message string) error {
	return outputGenericNotFound(paperID, message)
}

func outputAstaDuplicate(existingID, doi string) error {
	result := AstaAddResult{
		Action: "skipped",
		Error: &AstaErrorResult{
			Code:       "duplicate",
			Message:    "Paper already exists in collection",
			PaperID:    existingID,
			Suggestion: "Use --update flag to refresh metadata",
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Paper already exists: %s\n", existingID)
		if doi != "" {
			fmt.Fprintf(os.Stderr, "  DOI: %s\n", doi)
		}
		fmt.Fprintf(os.Stderr, "  Use --update to refresh metadata\n")
	} else {
		outputJSON(result)
	}
	os.Exit(ExitAstaDuplicate)
	return nil
}

func outputAstaRateLimited(err error) error {
	return outputGenericRateLimited(err)
}

func outputAstaError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
