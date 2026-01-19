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
	astaReferencesMissing bool
	astaReferencesLimit   int
)

var astaReferencesCmd = &cobra.Command{
	Use:   "references <paper-id>",
	Short: "Find papers referenced by a given paper",
	Long: `Find papers referenced by a given paper (backward exploration).

Queries Semantic Scholar for papers that the specified paper cites.
Can filter to show only references that are missing from your collection.

Examples:
  bp asta references Zhang2018-vi
  bp asta references DOI:10.1093/sysbio/syy032 --missing
  bp asta references Zhang2018-vi --limit 50 --human`,
	Args: cobra.ExactArgs(1),
	RunE: runAstaReferences,
}

func init() {
	astaCmd.AddCommand(astaReferencesCmd)
	astaReferencesCmd.Flags().BoolVar(&astaReferencesMissing, "missing", false, "Only show references NOT in local collection")
	astaReferencesCmd.Flags().IntVarP(&astaReferencesLimit, "limit", "n", 100, "Maximum results")
}

// AstaReferencesResult is the JSON output for the references command.
type AstaReferencesResult struct {
	PaperID      string             `json:"paper_id"`
	References   []AstaCitationInfo `json:"references"`
	Total        int                `json:"total"`
	InCollection int                `json:"inCollection"`
	Missing      int                `json:"missing"`
	Error        *AstaErrorResult   `json:"error,omitempty"`
}

func runAstaReferences(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	ctx := context.Background()

	// Find repository and load refs
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputReferencesError(ExitAstaAPIError, "reading refs", err)
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
			return outputReferencesNotFound(paperID)
		}
		s2ID = resolved
	}

	// Fetch references from S2
	refsResp, err := client.GetReferences(ctx, s2ID, astaReferencesLimit)
	if err != nil {
		if asta.IsNotFound(err) {
			return outputReferencesNotFound(paperID)
		}
		if asta.IsRateLimited(err) {
			return outputAstaRateLimited(err)
		}
		return outputReferencesError(ExitAstaAPIError, "fetching references", err)
	}

	// Build result
	result := AstaReferencesResult{
		PaperID:    paperID,
		References: make([]AstaCitationInfo, 0),
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
		if astaReferencesMissing && exists {
			continue
		}

		// Build reference info
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

		result.References = append(result.References, info)
	}

	result.Total = totalRefs
	result.InCollection = inCollection
	result.Missing = totalRefs - inCollection

	// Output result
	if humanOutput {
		outputReferencesHuman(result)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
	return nil
}

func outputReferencesHuman(result AstaReferencesResult) {
	if astaReferencesMissing {
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
	result := AstaReferencesResult{
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

func outputReferencesError(exitCode int, context string, err error) error {
	result := AstaReferencesResult{
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
