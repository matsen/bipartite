package main

import (
	"context"
	"fmt"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	s2CitationsLocalOnly bool
	s2CitationsLimit     int
)

var s2CitationsCmd = &cobra.Command{
	Use:   "citations <paper-id>",
	Short: "Find papers that cite a given paper",
	Long: `Find papers that cite a given paper (forward citation tracking).

Queries Semantic Scholar for papers that cite the specified paper.
Can filter to show only citations that are already in your collection.

Examples:
  bip s2 citations Zhang2018-vi
  bip s2 citations DOI:10.1093/sysbio/syy032 --local-only
  bip s2 citations Zhang2018-vi --limit 20 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runS2Citations,
}

func init() {
	s2Cmd.AddCommand(s2CitationsCmd)
	s2CitationsCmd.Flags().BoolVar(&s2CitationsLocalOnly, "local-only", false, "Only show citations in local collection")
	s2CitationsCmd.Flags().IntVarP(&s2CitationsLimit, "limit", "n", 50, "Maximum results")
}

// S2CitationsResult is the JSON output for the citations command.
type S2CitationsResult struct {
	PaperID   string           `json:"paper_id"`
	Citations []S2CitationInfo `json:"citations"`
	Total     int              `json:"total"`
	Error     *S2ErrorResult   `json:"error,omitempty"`
}

// S2CitationInfo represents a single citation.
type S2CitationInfo struct {
	PaperID       string   `json:"paperId"`
	DOI           string   `json:"doi,omitempty"`
	Title         string   `json:"title"`
	Authors       []string `json:"authors,omitempty"`
	Year          int      `json:"year,omitempty"`
	Venue         string   `json:"venue,omitempty"`
	ExistsLocally bool     `json:"existsLocally"`
	LocalID       *string  `json:"localId"`
}

func runS2Citations(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Find repository and load refs
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputCitationsError(ExitS2APIError, "reading refs", err)
	}

	// Create resolver and client
	resolver := s2.NewLocalResolverFromRefs(refs)
	client := s2.NewClient()

	// Resolve paper ID
	parsed := s2.ParsePaperID(paperID)
	var s2ID string
	if parsed.IsExternalID() {
		s2ID = parsed.String()
	} else {
		resolved, _, err := resolver.ResolveToS2ID(paperID)
		if err != nil {
			return outputCitationsNotFound(paperID)
		}
		s2ID = resolved
	}

	// Fetch citations from S2
	citationsResp, err := client.GetCitations(ctx, s2ID, s2CitationsLimit)
	if err != nil {
		if s2.IsNotFound(err) {
			return outputCitationsNotFound(paperID)
		}
		if s2.IsRateLimited(err) {
			return outputS2RateLimited(err)
		}
		return outputCitationsError(ExitS2APIError, "fetching citations", err)
	}

	// Build result
	result := S2CitationsResult{
		PaperID:   paperID,
		Citations: make([]S2CitationInfo, 0),
	}

	for _, c := range citationsResp.Data {
		if c.CitingPaper == nil {
			continue
		}
		paper := c.CitingPaper

		// Check local existence
		localRef, exists := resolver.ExistsLocally(*paper)

		// Filter if local-only
		if s2CitationsLocalOnly && !exists {
			continue
		}

		// Build citation info
		info := S2CitationInfo{
			PaperID:       paper.PaperID,
			DOI:           paper.ExternalIDs.DOI,
			Title:         paper.Title,
			Year:          paper.Year,
			Venue:         paper.Venue,
			ExistsLocally: exists,
		}

		// Add authors
		for _, a := range paper.Authors {
			info.Authors = append(info.Authors, a.Name)
		}

		// Set local ID if exists
		if exists && localRef != nil {
			localID := localRef.ID
			info.LocalID = &localID
		}

		result.Citations = append(result.Citations, info)
	}

	result.Total = len(result.Citations)

	// Output result
	if humanOutput {
		outputCitationsHuman(result)
	} else {
		outputJSON(result)
	}
	return nil
}

func outputCitationsHuman(result S2CitationsResult) {
	fmt.Printf("Papers citing %s:\n\n", result.PaperID)

	inCollection := 0
	for _, c := range result.Citations {
		if c.ExistsLocally {
			inCollection++
			fmt.Printf("  [IN COLLECTION: %s] %s (%d)\n", *c.LocalID, c.Title, c.Year)
		} else {
			fmt.Printf("  [NOT IN COLLECTION] %s (%d)\n", c.Title, c.Year)
		}
		if c.DOI != "" {
			fmt.Printf("    DOI: %s\n", c.DOI)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d citations", result.Total)
	if result.Total > 0 {
		fmt.Printf(" (%d in collection)", inCollection)
	}
	fmt.Println()
}

func outputCitationsNotFound(paperID string) error {
	return outputGenericNotFound(paperID, "Paper not found")
}

func outputCitationsError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
