package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	s2LookupFields string
	s2LookupExists bool
)

var s2LookupCmd = &cobra.Command{
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
  bp s2 lookup DOI:10.1038/nature12373
  bp s2 lookup DOI:10.1038/nature12373 --fields title,citationCount
  bp s2 lookup Zhang2018-vi --exists --human`,
	Args: cobra.ExactArgs(1),
	RunE: runS2Lookup,
}

func init() {
	s2Cmd.AddCommand(s2LookupCmd)
	s2LookupCmd.Flags().StringVarP(&s2LookupFields, "fields", "f", "", "Comma-separated fields to return (default: all)")
	s2LookupCmd.Flags().BoolVarP(&s2LookupExists, "exists", "e", false, "Include whether paper exists in local collection")
}

// S2LookupResult is the JSON output for the lookup command.
type S2LookupResult struct {
	PaperID        string         `json:"paperId"`
	DOI            string         `json:"doi,omitempty"`
	ArXiv          string         `json:"arxiv,omitempty"`
	Title          string         `json:"title"`
	Authors        []S2Author     `json:"authors,omitempty"`
	Abstract       string         `json:"abstract,omitempty"`
	Year           int            `json:"year,omitempty"`
	Venue          string         `json:"venue,omitempty"`
	CitationCount  int            `json:"citationCount,omitempty"`
	ReferenceCount int            `json:"referenceCount,omitempty"`
	Fields         []string       `json:"fieldsOfStudy,omitempty"`
	IsOpenAccess   bool           `json:"isOpenAccess,omitempty"`
	ExistsLocally  *bool          `json:"existsLocally,omitempty"`
	LocalID        string         `json:"localId,omitempty"`
	Error          *S2ErrorResult `json:"error,omitempty"`
}

// S2Author represents an author in lookup results.
type S2Author struct {
	Name     string `json:"name"`
	AuthorID string `json:"authorId,omitempty"`
}

func runS2Lookup(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Create client
	client := s2.NewClient()

	// Parse paper ID
	parsed := s2.ParsePaperID(paperID)
	var s2ID string

	// Handle local ID resolution if needed
	var resolver *s2.LocalResolver
	if s2LookupExists || !parsed.IsExternalID() {
		repoRoot := mustFindRepository()
		refsPath := config.RefsPath(repoRoot)
		refs, err := storage.ReadAll(refsPath)
		if err != nil {
			return outputLookupError(ExitS2APIError, "reading refs", err)
		}
		resolver = s2.NewLocalResolverFromRefs(refs)
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
		if s2.IsNotFound(err) {
			return outputLookupNotFound(paperID)
		}
		if s2.IsRateLimited(err) {
			return outputS2RateLimited(err)
		}
		return outputLookupError(ExitS2APIError, "fetching paper", err)
	}

	// Build result
	result := S2LookupResult{
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
		result.Authors = append(result.Authors, S2Author{
			Name:     a.Name,
			AuthorID: a.AuthorID,
		})
	}

	// Check local existence if requested
	if s2LookupExists && resolver != nil {
		localRef, exists := resolver.ExistsLocally(*paper)
		result.ExistsLocally = &exists
		if exists && localRef != nil {
			result.LocalID = localRef.ID
		}
	}

	// Filter fields if requested
	if s2LookupFields != "" {
		result = filterLookupFields(result, s2LookupFields)
	}

	// Output result
	if humanOutput {
		outputLookupHuman(result)
	} else {
		outputJSON(result)
	}
	return nil
}

func filterLookupFields(result S2LookupResult, fieldsStr string) S2LookupResult {
	fields := make(map[string]bool)
	for _, f := range strings.Split(fieldsStr, ",") {
		fields[strings.TrimSpace(strings.ToLower(f))] = true
	}

	filtered := S2LookupResult{
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

func outputLookupHuman(result S2LookupResult) {
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
	return outputGenericNotFound(paperID, "Paper not found in Semantic Scholar")
}

func outputLookupError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
