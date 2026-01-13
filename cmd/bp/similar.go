package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/semantic"
	"github.com/spf13/cobra"
)

var (
	similarLimit int
)

func init() {
	rootCmd.AddCommand(similarCmd)

	similarCmd.Flags().IntVarP(&similarLimit, "limit", "l", 10, "Maximum number of results")
}

// SimilarSource is the source paper info for similar papers response.
type SimilarSource struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SimilarResponse is the response for the similar papers command.
type SimilarResponse struct {
	Source  SimilarSource       `json:"source"`
	Similar []PaperSearchResult `json:"similar"`
	Total   int                 `json:"total"`
	Model   string              `json:"model"`
}

var similarCmd = &cobra.Command{
	Use:   "similar <paper-id>",
	Short: "Find papers similar to a specific paper",
	Long: `Find papers that are semantically similar to a given paper.

This uses the paper's abstract to find other papers with related content.
The source paper is excluded from results.

Requires the semantic index to be built first with 'bp index build'.`,
	Args: cobra.ExactArgs(1),
	RunE: runSimilar,
}

func runSimilar(cmd *cobra.Command, args []string) error {
	paperID := args[0]
	repoRoot := mustFindRepository()

	// Load index
	idx := mustLoadSemanticIndex(repoRoot)

	// Open database
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Check if paper is in index first (more efficient than DB lookup)
	if !idx.HasPaper(paperID) {
		// Paper not in index - check if it exists at all to provide helpful error
		sourcePaper, _ := db.GetByID(paperID)
		if sourcePaper == nil {
			exitWithError(ExitError, "Paper '%s' not found in database", paperID)
		}
		// Paper exists but not indexed - likely no/short abstract
		exitWithError(ExitNoAbstract, "Paper '%s' is not in the semantic index\n\nThis paper may have no abstract or an abstract shorter than %d characters.\nRebuild the index with 'bp index build' if you recently added an abstract.", paperID, semantic.MinAbstractLength)
	}

	// Get source paper info for display
	sourcePaper, err := db.GetByID(paperID)
	if err != nil {
		exitWithError(ExitError, "looking up paper details: %v", err)
	}

	// Find similar papers
	results, err := idx.FindSimilar(paperID, similarLimit)
	if err != nil {
		exitWithError(ExitError, "finding similar papers: %v", err)
	}

	// Build response
	similarResults := buildSearchResults(results, db, false)

	// Output
	if humanOutput {
		fmt.Printf("Papers similar to: %s\n", paperID)
		fmt.Printf("\"%s\"\n\n", truncateString(sourcePaper.Title, DetailTitleMaxLen))
		printSearchResultsHuman(similarResults)
	} else {
		outputJSON(SimilarResponse{
			Source: SimilarSource{
				ID:    sourcePaper.ID,
				Title: sourcePaper.Title,
			},
			Similar: similarResults,
			Total:   len(similarResults),
			Model:   idx.ModelName,
		})
	}

	return nil
}
