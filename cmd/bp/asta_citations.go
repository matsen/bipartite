package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	astaCitationsLocalOnly bool
	astaCitationsLimit     int
)

var astaCitationsCmd = &cobra.Command{
	Use:   "citations <paper-id>",
	Short: "Find papers that cite a given paper",
	Long: `Find papers that cite a given paper (forward citation tracking).

Queries Semantic Scholar for papers that cite the specified paper.
Can filter to show only citations that are already in your collection.

Examples:
  bp asta citations Zhang2018-vi
  bp asta citations DOI:10.1093/sysbio/syy032 --local-only
  bp asta citations Zhang2018-vi --limit 20 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runAstaCitations,
}

func init() {
	astaCmd.AddCommand(astaCitationsCmd)
	astaCitationsCmd.Flags().BoolVar(&astaCitationsLocalOnly, "local-only", false, "Only show citations in local collection")
	astaCitationsCmd.Flags().IntVarP(&astaCitationsLimit, "limit", "n", 50, "Maximum results")
}

// AstaCitationsResult is the JSON output for the citations command.
type AstaCitationsResult struct {
	PaperID   string             `json:"paper_id"`
	Citations []AstaCitationInfo `json:"citations"`
	Total     int                `json:"total"`
	Error     *AstaErrorResult   `json:"error,omitempty"`
}

// AstaCitationInfo represents a single citation.
type AstaCitationInfo struct {
	PaperID       string   `json:"paperId"`
	DOI           string   `json:"doi,omitempty"`
	Title         string   `json:"title"`
	Authors       []string `json:"authors,omitempty"`
	Year          int      `json:"year,omitempty"`
	Venue         string   `json:"venue,omitempty"`
	ExistsLocally bool     `json:"existsLocally"`
	LocalID       *string  `json:"localId"`
}

func runAstaCitations(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Find repository and load refs
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputCitationsError(ExitAstaAPIError, "reading refs", err)
	}

	// Create resolver and client
	resolver := asta.NewLocalResolverFromRefs(refs)
	client := asta.NewClient()

	// Resolve paper ID
	parsed := asta.ParsePaperID(paperID)
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
	citationsResp, err := client.GetCitations(ctx, s2ID, astaCitationsLimit)
	if err != nil {
		if asta.IsNotFound(err) {
			return outputCitationsNotFound(paperID)
		}
		if asta.IsRateLimited(err) {
			return outputAstaRateLimited(err)
		}
		return outputCitationsError(ExitAstaAPIError, "fetching citations", err)
	}

	// Build result
	result := AstaCitationsResult{
		PaperID:   paperID,
		Citations: make([]AstaCitationInfo, 0),
	}

	for _, c := range citationsResp.Data {
		if c.CitingPaper == nil {
			continue
		}
		paper := c.CitingPaper

		// Check local existence
		localRef, exists := resolver.ExistsLocally(*paper)

		// Filter if local-only
		if astaCitationsLocalOnly && !exists {
			continue
		}

		// Build citation info
		info := AstaCitationInfo{
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
	return nil
}

func outputCitationsHuman(result AstaCitationsResult) {
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
	result := AstaCitationsResult{
		PaperID: paperID,
		Error: &AstaErrorResult{
			Code:       "not_found",
			Message:    "Paper not found",
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

func outputCitationsError(exitCode int, context string, err error) error {
	result := AstaCitationsResult{
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
