package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	s2AddUpdate bool
	s2AddLink   string
)

var s2AddCmd = &cobra.Command{
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
  bp s2 add DOI:10.1038/nature12373
  bp s2 add ARXIV:2106.15928 --link ~/papers/paper.pdf
  bp s2 add DOI:10.1093/sysbio/syy032 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runS2Add,
}

func init() {
	s2Cmd.AddCommand(s2AddCmd)
	s2AddCmd.Flags().BoolVarP(&s2AddUpdate, "update", "u", false, "Update metadata if paper already exists")
	s2AddCmd.Flags().StringVarP(&s2AddLink, "link", "l", "", "Set pdf_path to the given file path")
}

// S2AddResult is the JSON output for the add command.
type S2AddResult struct {
	Action string             `json:"action"` // added, updated, skipped
	Paper  *S2AddPaperSummary `json:"paper,omitempty"`
	Error  *S2ErrorResult     `json:"error,omitempty"`
}

// S2AddPaperSummary is a summary of the added paper.
type S2AddPaperSummary struct {
	ID      string   `json:"id"`
	DOI     string   `json:"doi,omitempty"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Year    int      `json:"year"`
	Venue   string   `json:"venue,omitempty"`
}

// S2ErrorResult is the JSON output for errors.
type S2ErrorResult struct {
	Code       string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	RetryAfter int    `json:"retry_after,omitempty"`
	PaperID    string `json:"paper_id,omitempty"`
}

func runS2Add(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputS2Error(ExitS2APIError, "reading refs", err)
	}

	// Create resolver and client
	resolver := s2.NewLocalResolverFromRefs(refs)
	client := s2.NewClient()

	// Parse and resolve paper ID
	parsed := s2.ParsePaperID(paperID)
	var s2ID string
	if parsed.IsExternalID() {
		s2ID = parsed.String()
	} else {
		// Try to resolve local ID
		resolved, _, err := resolver.ResolveToS2ID(paperID)
		if err != nil {
			return outputS2NotFound(paperID, "Local paper ID not found")
		}
		s2ID = resolved
	}

	// Fetch paper from S2
	paper, err := client.GetPaper(ctx, s2ID)
	if err != nil {
		if s2.IsNotFound(err) {
			return outputS2NotFound(paperID, "Paper not found in Semantic Scholar")
		}
		if s2.IsRateLimited(err) {
			return outputS2RateLimited(err)
		}
		return outputS2Error(ExitS2APIError, "fetching paper", err)
	}

	// Map to reference
	ref := s2.MapS2ToReference(*paper)

	// Set PDF path if requested
	if s2AddLink != "" {
		ref.PDFPath = s2AddLink
	}

	// Check for duplicates
	if paper.ExternalIDs.DOI != "" {
		if existingRef, found := resolver.FindByDOI(paper.ExternalIDs.DOI); found {
			if !s2AddUpdate {
				return outputS2Duplicate(existingRef.ID, paper.ExternalIDs.DOI)
			}
			// Update existing
			return updateExistingPaper(refsPath, refs, existingRef.ID, ref)
		}
	}

	// Check by S2 ID
	if existingRef, found := resolver.FindByS2ID(paper.PaperID); found {
		if !s2AddUpdate {
			return outputS2Duplicate(existingRef.ID, "")
		}
		// Update existing
		return updateExistingPaper(refsPath, refs, existingRef.ID, ref)
	}

	// Generate unique ID
	ref.ID = storage.GenerateUniqueID(refs, ref.ID)

	// Append to refs
	if err := storage.Append(refsPath, ref); err != nil {
		return outputS2Error(ExitS2APIError, "saving reference", err)
	}

	// Output result
	return outputS2AddResult("added", ref)
}

func updateExistingPaper(refsPath string, refs []reference.Reference, existingID string, newRef reference.Reference) error {
	// Find and update the existing reference
	for i, ref := range refs {
		if ref.ID == existingID {
			// Preserve the original ID
			newRef.ID = existingID
			// Preserve PDF path if not being updated
			if s2AddLink == "" && ref.PDFPath != "" {
				newRef.PDFPath = ref.PDFPath
			}
			refs[i] = newRef
			break
		}
	}

	// Write all refs
	if err := storage.WriteAll(refsPath, refs); err != nil {
		return outputS2Error(ExitS2APIError, "saving reference", err)
	}

	return outputS2AddResult("updated", newRef)
}

func outputS2AddResult(action string, ref reference.Reference) error {
	authors := formatAuthors(ref.Authors)

	result := S2AddResult{
		Action: action,
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
	} else {
		outputJSON(result)
	}
	return nil
}

func outputS2NotFound(paperID, message string) error {
	return outputGenericNotFound(paperID, message)
}

func outputS2Duplicate(existingID, doi string) error {
	result := S2AddResult{
		Action: "skipped",
		Error: &S2ErrorResult{
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
	os.Exit(ExitS2Duplicate)
	return nil
}

func outputS2RateLimited(err error) error {
	return outputGenericRateLimited(err)
}

func outputS2Error(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
