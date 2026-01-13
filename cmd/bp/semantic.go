package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/embedding"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/semantic"
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

// SemanticResult represents a paper in semantic search results.
type SemanticResult struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Authors    []reference.Author `json:"authors"`
	Year       int                `json:"year"`
	Similarity float32            `json:"similarity"`
	Abstract   string             `json:"abstract,omitempty"`
}

// SemanticResponse is the response for the semantic search command.
type SemanticResponse struct {
	Query     string           `json:"query"`
	Results   []SemanticResult `json:"results"`
	Total     int              `json:"total"`
	Threshold float32          `json:"threshold"`
	Model     string           `json:"model"`
}

var semanticCmd = &cobra.Command{
	Use:   "semantic <query>",
	Short: "Search papers by semantic similarity",
	Long: `Search papers using semantic similarity to find conceptually related papers.

Unlike keyword search, semantic search understands the meaning of your query
and finds papers with related concepts, even without exact word matches.

Requires the semantic index to be built first with 'bp index build'.`,
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
	idx, err := semantic.Load(repoRoot)
	if err != nil {
		if err == semantic.ErrIndexNotFound {
			exitWithError(ExitConfigError, "Semantic index not found\n\nRun 'bp index build' to create the index.")
		}
		exitWithError(ExitError, "loading index: %v", err)
	}

	// Check Ollama availability
	provider := embedding.NewOllamaProvider()
	if err := provider.IsAvailable(ctx); err != nil {
		exitWithError(ExitDataError, "Ollama is not running\n\nStart Ollama with 'ollama serve' or install from https://ollama.ai")
	}

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
	semanticResults := make([]SemanticResult, 0, len(results))
	for _, r := range results {
		ref, err := db.GetByID(r.PaperID)
		if err != nil || ref == nil {
			continue // Skip if paper not found
		}
		semanticResults = append(semanticResults, SemanticResult{
			ID:         ref.ID,
			Title:      ref.Title,
			Authors:    ref.Authors,
			Year:       ref.Published.Year,
			Similarity: r.Similarity,
			Abstract:   ref.Abstract,
		})
	}

	// Output
	if humanOutput {
		fmt.Printf("Search: \"%s\"\n", query)
		fmt.Printf("Found %d papers (threshold: %.1f)\n\n", len(semanticResults), semanticThreshold)

		for i, r := range semanticResults {
			fmt.Printf("%d. [%.2f] %s\n", i+1, r.Similarity, r.ID)
			fmt.Printf("   %s\n", truncateString(r.Title, SearchTitleMaxLen))
			fmt.Printf("   %s (%d)\n\n", formatAuthorsShort(r.Authors, 3), r.Year)
		}
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
