package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/embedding"
	"github.com/spf13/cobra"
)

var (
	semanticLimit     int
	semanticThreshold float32
)

func init() {
	rootCmd.AddCommand(semanticCmd)

	semanticCmd.Flags().IntVarP(&semanticLimit, "limit", "l", 10, "Maximum number of results")
	semanticCmd.Flags().Float32VarP(&semanticThreshold, "threshold", "t", 0.5, "Minimum similarity threshold (0.0-1.0)")
}

// SemanticResponse is the response for the semantic search command.
type SemanticResponse struct {
	Query     string              `json:"query"`
	Results   []PaperSearchResult `json:"results"`
	Total     int                 `json:"total"`
	Threshold float32             `json:"threshold"`
	Model     string              `json:"model"`
}

var semanticCmd = &cobra.Command{
	Use:   "semantic <query>",
	Short: "Search papers by semantic similarity",
	Long: `Search papers using semantic similarity to find conceptually related papers.

Unlike keyword search, semantic search understands the meaning of your query
and finds papers with related concepts, even without exact word matches.

Requires the semantic index to be built first with 'bip index build'.`,
	Args: cobra.ExactArgs(1),
	RunE: runSemantic,
}

func runSemantic(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	query := strings.TrimSpace(args[0])

	// Validate query
	if query == "" {
		exitWithError(ExitError, "Search query cannot be empty")
	}

	repoRoot := mustFindRepository()

	// Load index
	idx := mustLoadSemanticIndex(repoRoot)

	// Check Ollama availability (no model check needed for query-only operations)
	provider := embedding.NewOllamaProvider()
	mustValidateOllama(ctx, provider, false)

	// Generate query embedding
	queryEmb, err := provider.Embed(ctx, query)
	if err != nil {
		exitWithError(ExitError, "generating query embedding: %v", err)
	}

	// Search
	results, err := idx.Search(queryEmb.Vector, semanticLimit, semanticThreshold)
	if err != nil {
		exitWithError(ExitError, "searching index: %v", err)
	}

	// Open database to get full paper info
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Build response
	semanticResults := buildSearchResults(results, db, true)

	// Output
	if humanOutput {
		fmt.Printf("Search: \"%s\"\n", query)
		fmt.Printf("Found %d papers (threshold: %.1f)\n\n", len(semanticResults), semanticThreshold)
		printSearchResultsHuman(semanticResults)
	} else {
		outputJSON(SemanticResponse{
			Query:     query,
			Results:   semanticResults,
			Total:     len(semanticResults),
			Threshold: semanticThreshold,
			Model:     provider.ModelName(),
		})
	}

	return nil
}
