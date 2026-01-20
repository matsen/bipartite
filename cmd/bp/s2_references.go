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
	s2ReferencesMissing bool
	s2ReferencesLimit   int
)

var s2ReferencesCmd = &cobra.Command{
	Use:   "references <paper-id>",
	Short: "Find papers referenced by a given paper",
	Long: `Find papers referenced by a given paper (backward exploration).

Queries Semantic Scholar for papers that the specified paper cites.
Can filter to show only references that are missing from your collection.

Examples:
  bp s2 references Zhang2018-vi
  bp s2 references DOI:10.1093/sysbio/syy032 --missing
  bp s2 references Zhang2018-vi --limit 50 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runS2References,
}

func init() {
	s2Cmd.AddCommand(s2ReferencesCmd)
	s2ReferencesCmd.Flags().BoolVar(&s2ReferencesMissing, "missing", false, "Only show references NOT in local collection")
	s2ReferencesCmd.Flags().IntVarP(&s2ReferencesLimit, "limit", "n", 100, "Maximum results")
}

// S2ReferencesResult is the JSON output for the references command.
type S2ReferencesResult struct {
	PaperID      string           `json:"paper_id"`
	References   []S2CitationInfo `json:"references"`
	Total        int              `json:"total"`
	InCollection int              `json:"inCollection"`
	Missing      int              `json:"missing"`
	Error        *S2ErrorResult   `json:"error,omitempty"`
}

func runS2References(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Find repository and load refs
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputReferencesError(ExitS2APIError, "reading refs", err)
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
			return outputReferencesNotFound(paperID)
		}
		s2ID = resolved
	}

	// Fetch references from S2
	refsResp, err := client.GetReferences(ctx, s2ID, s2ReferencesLimit)
	if err != nil {
		if s2.IsNotFound(err) {
			return outputReferencesNotFound(paperID)
		}
		if s2.IsRateLimited(err) {
			return outputS2RateLimited(err)
		}
		return outputReferencesError(ExitS2APIError, "fetching references", err)
	}

	// Build result
	result := S2ReferencesResult{
		PaperID:    paperID,
		References: make([]S2CitationInfo, 0),
	}

	totalRefs := 0
	inCollection := 0

	for _, r := range refsResp.Data {
		if r.CitedPaper == nil {
			continue
		}
		paper := r.CitedPaper
		totalRefs++

		// Check local existence
		localRef, exists := resolver.ExistsLocally(*paper)
		if exists {
			inCollection++
		}

		// Filter if missing-only
		if s2ReferencesMissing && exists {
			continue
		}

		// Build reference info
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

		result.References = append(result.References, info)
	}

	result.Total = totalRefs
	result.InCollection = inCollection
	result.Missing = totalRefs - inCollection

	// Output result
	if humanOutput {
		outputReferencesHuman(result)
	} else {
		outputJSON(result)
	}
	return nil
}

func outputReferencesHuman(result S2ReferencesResult) {
	if s2ReferencesMissing {
		fmt.Printf("Missing references from %s:\n\n", result.PaperID)
	} else {
		fmt.Printf("References from %s:\n\n", result.PaperID)
	}

	for _, r := range result.References {
		if r.ExistsLocally {
			fmt.Printf("  [IN COLLECTION: %s] %s (%d)\n", *r.LocalID, r.Title, r.Year)
		} else {
			fmt.Printf("  [MISSING] %s (%d)\n", r.Title, r.Year)
		}
		if r.DOI != "" {
			fmt.Printf("    DOI: %s\n", r.DOI)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d references (%d in collection, %d missing)\n",
		result.Total, result.InCollection, result.Missing)
}

func outputReferencesNotFound(paperID string) error {
	return outputGenericNotFound(paperID, "Paper not found")
}

func outputReferencesError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
