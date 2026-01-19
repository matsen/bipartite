package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	astaLookupFields string
	astaLookupExists bool
)

var astaLookupCmd = &cobra.Command{
	Use:   "lookup <paper-id>",
	Short: "Query Semantic Scholar for paper information without adding",
	Long: `Query Semantic Scholar for paper information without adding to collection.

Useful for checking paper details before adding, or for verifying citation counts.

Supported paper ID formats:
  DOI:10.1038/nature12373      DOI
  ARXIV:2106.15928             arXiv ID
  PMID:19872477                PubMed ID
  <local-id>                   Local paper ID from collection

Examples:
  bp asta lookup DOI:10.1038/nature12373
  bp asta lookup DOI:10.1038/nature12373 --fields title,citationCount
  bp asta lookup Zhang2018-vi --exists --human`,
	Args: cobra.ExactArgs(1),
	RunE: runAstaLookup,
}

func init() {
	astaCmd.AddCommand(astaLookupCmd)
	astaLookupCmd.Flags().StringVarP(&astaLookupFields, "fields", "f", "", "Comma-separated fields to return (default: all)")
	astaLookupCmd.Flags().BoolVarP(&astaLookupExists, "exists", "e", false, "Include whether paper exists in local collection")
}

// AstaLookupResult is the JSON output for the lookup command.
type AstaLookupResult struct {
	PaperID        string           `json:"paperId"`
	DOI            string           `json:"doi,omitempty"`
	ArXiv          string           `json:"arxiv,omitempty"`
	Title          string           `json:"title"`
	Authors        []AstaAuthor     `json:"authors,omitempty"`
	Abstract       string           `json:"abstract,omitempty"`
	Year           int              `json:"year,omitempty"`
	Venue          string           `json:"venue,omitempty"`
	CitationCount  int              `json:"citationCount,omitempty"`
	ReferenceCount int              `json:"referenceCount,omitempty"`
	Fields         []string         `json:"fieldsOfStudy,omitempty"`
	IsOpenAccess   bool             `json:"isOpenAccess,omitempty"`
	ExistsLocally  *bool            `json:"existsLocally,omitempty"`
	LocalID        string           `json:"localId,omitempty"`
	Error          *AstaErrorResult `json:"error,omitempty"`
}

// AstaAuthor represents an author in lookup results.
type AstaAuthor struct {
	Name     string `json:"name"`
	AuthorID string `json:"authorId,omitempty"`
}

func runAstaLookup(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Create client
	client := asta.NewClient()

	// Parse paper ID
	parsed := asta.ParsePaperID(paperID)
	var s2ID string

	// Handle local ID resolution if needed
	var resolver *asta.LocalResolver
	if astaLookupExists || !parsed.IsExternalID() {
		repoRoot := mustFindRepository()
		refsPath := config.RefsPath(repoRoot)
		refs, err := storage.ReadAll(refsPath)
		if err != nil {
			return outputLookupError(ExitAstaAPIError, "reading refs", err)
		}
		resolver = asta.NewLocalResolverFromRefs(refs)
	}

	// Resolve the ID
	if parsed.IsExternalID() {
		s2ID = parsed.String()
	} else {
		if resolver == nil {
			return outputLookupNotFound(paperID)
		}
		resolved, _, err := resolver.ResolveToS2ID(paperID)
		if err != nil {
			return outputLookupNotFound(paperID)
		}
		s2ID = resolved
	}

	// Fetch paper from S2
	paper, err := client.GetPaper(ctx, s2ID)
	if err != nil {
		if asta.IsNotFound(err) {
			return outputLookupNotFound(paperID)
		}
		if asta.IsRateLimited(err) {
			return outputAstaRateLimited(err)
		}
		return outputLookupError(ExitAstaAPIError, "fetching paper", err)
	}

	// Build result
	result := AstaLookupResult{
		PaperID:        paper.PaperID,
		DOI:            paper.ExternalIDs.DOI,
		ArXiv:          paper.ExternalIDs.ArXiv,
		Title:          paper.Title,
		Abstract:       paper.Abstract,
		Year:           paper.Year,
		Venue:          paper.Venue,
		CitationCount:  paper.Citations,
		ReferenceCount: paper.References,
		Fields:         paper.Fields,
		IsOpenAccess:   paper.IsOpenAccess,
	}

	// Map authors
	for _, a := range paper.Authors {
		result.Authors = append(result.Authors, AstaAuthor{
			Name:     a.Name,
			AuthorID: a.AuthorID,
		})
	}

	// Check local existence if requested
	if astaLookupExists && resolver != nil {
		localRef, exists := resolver.ExistsLocally(*paper)
		result.ExistsLocally = &exists
		if exists && localRef != nil {
			result.LocalID = localRef.ID
		}
	}

	// Filter fields if requested
	if astaLookupFields != "" {
		result = filterLookupFields(result, astaLookupFields)
	}

	// Output result
	if humanOutput {
		outputLookupHuman(result)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
	return nil
}

func filterLookupFields(result AstaLookupResult, fieldsStr string) AstaLookupResult {
	fields := make(map[string]bool)
	for _, f := range strings.Split(fieldsStr, ",") {
		fields[strings.TrimSpace(strings.ToLower(f))] = true
	}

	filtered := AstaLookupResult{
		PaperID: result.PaperID, // Always include
	}

	if fields["doi"] {
		filtered.DOI = result.DOI
	}
	if fields["arxiv"] {
		filtered.ArXiv = result.ArXiv
	}
	if fields["title"] {
		filtered.Title = result.Title
	}
	if fields["authors"] {
		filtered.Authors = result.Authors
	}
	if fields["abstract"] {
		filtered.Abstract = result.Abstract
	}
	if fields["year"] {
		filtered.Year = result.Year
	}
	if fields["venue"] {
		filtered.Venue = result.Venue
	}
	if fields["citationcount"] {
		filtered.CitationCount = result.CitationCount
	}
	if fields["referencecount"] {
		filtered.ReferenceCount = result.ReferenceCount
	}
	if fields["fieldsofstudy"] {
		filtered.Fields = result.Fields
	}
	if fields["isopenaccess"] {
		filtered.IsOpenAccess = result.IsOpenAccess
	}
	if fields["existslocally"] || result.ExistsLocally != nil {
		filtered.ExistsLocally = result.ExistsLocally
		filtered.LocalID = result.LocalID
	}

	return filtered
}

func outputLookupHuman(result AstaLookupResult) {
	fmt.Printf("%s\n", result.Title)
	if len(result.Authors) > 0 {
		names := make([]string, 0, len(result.Authors))
		for _, a := range result.Authors {
			names = append(names, a.Name)
		}
		fmt.Printf("  Authors: %s\n", strings.Join(names, ", "))
	}
	if result.Year > 0 {
		fmt.Printf("  Year: %d\n", result.Year)
	}
	if result.Venue != "" {
		fmt.Printf("  Venue: %s\n", result.Venue)
	}
	if result.DOI != "" {
		fmt.Printf("  DOI: %s\n", result.DOI)
	}
	if result.CitationCount > 0 {
		fmt.Printf("  Citations: %d\n", result.CitationCount)
	}
	if result.ReferenceCount > 0 {
		fmt.Printf("  References: %d\n", result.ReferenceCount)
	}
	if result.ExistsLocally != nil {
		if *result.ExistsLocally {
			fmt.Printf("  In collection: Yes (%s)\n", result.LocalID)
		} else {
			fmt.Printf("  In collection: No\n")
		}
	}
}

func outputLookupNotFound(paperID string) error {
	result := AstaLookupResult{
		Error: &AstaErrorResult{
			Code:       "not_found",
			Message:    "Paper not found in Semantic Scholar",
			PaperID:    paperID,
			Suggestion: "Verify the paper ID is correct",
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: Paper not found\n")
		fmt.Fprintf(os.Stderr, "  Paper ID: %s\n", paperID)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
	os.Exit(ExitAstaNotFound)
	return nil
}

func outputLookupError(exitCode int, context string, err error) error {
	result := AstaLookupResult{
		Error: &AstaErrorResult{
			Code:    "api_error",
			Message: fmt.Sprintf("%s: %v", context, err),
		},
	}

	if humanOutput {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", context, err)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
	os.Exit(exitCode)
	return nil
}
