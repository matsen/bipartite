package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/s2"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	s2GapsMinCitations int
	s2GapsLimit        int
)

var s2GapsCmd = &cobra.Command{
	Use:   "gaps",
	Short: "Discover literature gaps - highly cited papers not in your collection",
	Long: `Discover literature gaps by finding papers that are cited by multiple
papers in your collection but are not in your collection.

This helps identify foundational papers or important references you might be missing.

Examples:
  bp s2 gaps
  bp s2 gaps --min-citations 3
  bp s2 gaps --limit 10 --human`,
	Args: cobra.NoArgs,
	RunE: runS2Gaps,
}

func init() {
	s2Cmd.AddCommand(s2GapsCmd)
	s2GapsCmd.Flags().IntVarP(&s2GapsMinCitations, "min-citations", "m", 2, "Minimum citation count within collection")
	s2GapsCmd.Flags().IntVarP(&s2GapsLimit, "limit", "n", 20, "Maximum results")
}

// S2GapsResult is the JSON output for the gaps command.
type S2GapsResult struct {
	Gaps           []S2GapInfo    `json:"gaps"`
	Total          int            `json:"total"`
	AnalyzedPapers int            `json:"analyzed_papers"`
	Error          *S2ErrorResult `json:"error,omitempty"`
}

// S2GapInfo represents a single gap (missing paper).
type S2GapInfo struct {
	PaperID            string   `json:"paperId"`
	DOI                string   `json:"doi,omitempty"`
	Title              string   `json:"title"`
	Year               int      `json:"year,omitempty"`
	Venue              string   `json:"venue,omitempty"`
	CitedByLocal       []string `json:"citedByLocal"`
	CitationCountLocal int      `json:"citationCountLocal"`
}

// gapCandidate tracks a potential gap during aggregation.
type gapCandidate struct {
	paper   *s2.S2Paper
	citedBy []string
}

func runS2Gaps(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Find repository and load refs
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputGapsError(ExitS2APIError, "reading refs", err)
	}

	// Create resolver and client
	resolver := s2.NewLocalResolverFromRefs(refs)
	client := s2.NewClient()

	// Get papers with DOIs
	refsWithDOI := resolver.RefsWithDOI()
	if len(refsWithDOI) == 0 {
		return outputGapsResult(S2GapsResult{
			Gaps:           []S2GapInfo{},
			Total:          0,
			AnalyzedPapers: 0,
		})
	}

	// Progress tracking
	totalPapers := len(refsWithDOI)
	if humanOutput {
		fmt.Fprintf(os.Stderr, "Analyzing references from %d papers...\n", totalPapers)
	}

	// Aggregate gaps
	candidates := make(map[string]*gapCandidate) // keyed by S2 paper ID

	for i, ref := range refsWithDOI {
		if humanOutput && (i+1)%10 == 0 {
			fmt.Fprintf(os.Stderr, "  Progress: %d/%d papers\n", i+1, totalPapers)
		}

		// Build S2 ID
		s2ID := "DOI:" + ref.DOI

		// Fetch references for this paper
		refsResp, err := client.GetReferences(ctx, s2ID, GapsReferencesLimit)
		if err != nil {
			if s2.IsNotFound(err) {
				continue // Skip papers not in S2
			}
			if s2.IsRateLimited(err) {
				return outputS2RateLimited(err)
			}
			// Warn about unexpected errors instead of silently ignoring
			warnAPIError("Failed to fetch references", ref.ID, err)
			continue
		}

		// Process each reference
		for _, r := range refsResp.Data {
			if r.CitedPaper == nil || r.CitedPaper.PaperID == "" {
				continue
			}
			paper := r.CitedPaper

			// Skip if already in collection
			if _, exists := resolver.ExistsLocally(*paper); exists {
				continue
			}

			// Add to candidates
			if c, ok := candidates[paper.PaperID]; ok {
				c.citedBy = append(c.citedBy, ref.ID)
			} else {
				candidates[paper.PaperID] = &gapCandidate{
					paper:   paper,
					citedBy: []string{ref.ID},
				}
			}
		}
	}

	// Filter by min citations and build result
	var gaps []S2GapInfo
	for _, c := range candidates {
		if len(c.citedBy) < s2GapsMinCitations {
			continue
		}

		gap := S2GapInfo{
			PaperID:            c.paper.PaperID,
			DOI:                c.paper.ExternalIDs.DOI,
			Title:              c.paper.Title,
			Year:               c.paper.Year,
			Venue:              c.paper.Venue,
			CitedByLocal:       c.citedBy,
			CitationCountLocal: len(c.citedBy),
		}
		gaps = append(gaps, gap)
	}

	// Sort by citation count descending
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].CitationCountLocal > gaps[j].CitationCountLocal
	})

	// Limit results
	totalGaps := len(gaps)
	if len(gaps) > s2GapsLimit {
		gaps = gaps[:s2GapsLimit]
	}

	result := S2GapsResult{
		Gaps:           gaps,
		Total:          totalGaps,
		AnalyzedPapers: totalPapers,
	}

	return outputGapsResult(result)
}

func outputGapsResult(result S2GapsResult) error {
	if humanOutput {
		outputGapsHuman(result)
	} else {
		outputJSON(result)
	}
	return nil
}

func outputGapsHuman(result S2GapsResult) {
	if len(result.Gaps) == 0 {
		fmt.Println("No literature gaps found.")
		fmt.Printf("Analyzed %d papers.\n", result.AnalyzedPapers)
		return
	}

	fmt.Printf("Literature gaps (cited by %d+ papers in your collection):\n\n", s2GapsMinCitations)

	for _, g := range result.Gaps {
		fmt.Printf("  %s (%d)\n", g.Title, g.Year)
		if g.DOI != "" {
			fmt.Printf("    DOI: %s\n", g.DOI)
		}
		if g.Venue != "" {
			fmt.Printf("    Venue: %s\n", g.Venue)
		}
		fmt.Printf("    Cited by %d papers in your collection:\n", g.CitationCountLocal)
		for _, localID := range g.CitedByLocal {
			fmt.Printf("      - %s\n", localID)
		}
		fmt.Println()
	}

	fmt.Printf("Found %d gaps after analyzing %d papers.\n", result.Total, result.AnalyzedPapers)
}

func outputGapsError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "api_error", context, err)
}
