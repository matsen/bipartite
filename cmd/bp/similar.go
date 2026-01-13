package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/reference"
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

// SimilarResult represents a paper in similar papers results.
type SimilarResult struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Authors    []reference.Author `json:"authors"`
	Year       int                `json:"year"`
	Similarity float32            `json:"similarity"`
}

// SimilarResponse is the response for the similar papers command.
type SimilarResponse struct {
	Source  SimilarSource   `json:"source"`
	Similar []SimilarResult `json:"similar"`
	Total   int             `json:"total"`
	Model   string          `json:"model"`
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
	idx, err := semantic.Load(repoRoot)
	if err != nil {
		if err == semantic.ErrIndexNotFound {
			exitWithError(ExitConfigError, "Semantic index not found\n\nRun 'bp index build' to create the index.")
		}
		exitWithError(ExitError, "loading index: %v", err)
	}

	// Open database
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Get source paper
	sourcePaper, err := db.GetByID(paperID)
	if err != nil {
		exitWithError(ExitError, "looking up paper: %v", err)
	}
	if sourcePaper == nil {
		exitWithError(ExitError, "Paper '%s' not found", paperID)
	}

	// Check if paper has abstract (needed for similarity)
	if sourcePaper.Abstract == "" {
		exitWithError(ExitNoAbstract, "Paper '%s' has no abstract\n\nSimilarity search requires papers with abstracts.", paperID)
	}

	// Check if paper is in index
	if !idx.HasPaper(paperID) {
		exitWithError(ExitNoAbstract, "Paper '%s' has no abstract\n\nSimilarity search requires papers with abstracts.", paperID)
	}

	// Find similar papers
	results, err := idx.FindSimilar(paperID, similarLimit)
	if err != nil {
		if err == semantic.ErrPaperNotIndexed {
			exitWithError(ExitNoAbstract, "Paper '%s' has no abstract\n\nSimilarity search requires papers with abstracts.", paperID)
		}
		exitWithError(ExitError, "finding similar papers: %v", err)
	}

	// Build response
	similarResults := make([]SimilarResult, 0, len(results))
	for _, r := range results {
		ref, err := db.GetByID(r.PaperID)
		if err != nil || ref == nil {
			continue // Skip if paper not found
		}
		similarResults = append(similarResults, SimilarResult{
			ID:         ref.ID,
			Title:      ref.Title,
			Authors:    ref.Authors,
			Year:       ref.Published.Year,
			Similarity: r.Similarity,
		})
	}

	// Output
	if humanOutput {
		fmt.Printf("Papers similar to: %s\n", paperID)
		fmt.Printf("\"%s\"\n\n", truncateString(sourcePaper.Title, DetailTitleMaxLen))

		for i, r := range similarResults {
			fmt.Printf("%d. [%.2f] %s\n", i+1, r.Similarity, r.ID)
			fmt.Printf("   %s\n", truncateString(r.Title, SearchTitleMaxLen))
			fmt.Printf("   %s (%d)\n\n", formatAuthorsShort(r.Authors, 3), r.Year)
		}
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
