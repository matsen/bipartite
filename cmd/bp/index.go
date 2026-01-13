package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/matsen/bipartite/internal/embedding"
	"github.com/matsen/bipartite/internal/semantic"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	noProgress bool
)

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.AddCommand(indexBuildCmd)
	indexCmd.AddCommand(indexCheckCmd)

	indexBuildCmd.Flags().BoolVar(&noProgress, "no-progress", false, "Suppress progress output")
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Manage the semantic search index",
	Long:  `Commands for building and checking the semantic search index.`,
}

// IndexBuildResult is the response for index build command.
type IndexBuildResult struct {
	Status          string  `json:"status"`
	PapersIndexed   int     `json:"papers_indexed"`
	PapersSkipped   int     `json:"papers_skipped"`
	SkippedReason   string  `json:"skipped_reason"`
	DurationSeconds float64 `json:"duration_seconds"`
	Model           string  `json:"model"`
	IndexSizeBytes  int64   `json:"index_size_bytes"`
}

var indexBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build or rebuild the semantic index",
	Long: `Build or rebuild the semantic index from paper abstracts.

Requires Ollama to be running with the embedding model available.
Run 'ollama pull all-minilm:l6-v2' to download the model.`,
	RunE: runIndexBuild,
}

// validateOllamaSetup checks that Ollama is running and has the required model.
func validateOllamaSetup(ctx context.Context, provider *embedding.OllamaProvider) {
	if err := provider.IsAvailable(ctx); err != nil {
		exitWithError(ExitDataError, "Ollama is not running\n\nStart Ollama with 'ollama serve' or install from https://ollama.ai")
	}

	hasModel, err := provider.HasModel(ctx)
	if err != nil {
		exitWithError(ExitError, "checking model availability: %v", err)
	}
	if !hasModel {
		exitWithError(ExitModelNotFound, "Embedding model '%s' not found\n\nRun 'ollama pull %s' to download it.", provider.ModelName(), provider.ModelName())
	}
}

// outputBuildResults outputs the build statistics in the appropriate format.
func outputBuildResults(provider *embedding.OllamaProvider, stats *semantic.BuildStats) {
	if humanOutput {
		fmt.Printf("\nBuild complete:\n")
		fmt.Printf("  Papers indexed: %d\n", stats.PapersIndexed)
		fmt.Printf("  Papers skipped: %d (no abstract)\n", stats.PapersSkipped)
		fmt.Printf("  Time elapsed: %s\n", formatDuration(stats.Duration))
		fmt.Printf("  Index size: %s\n", formatBytes(stats.IndexSizeBytes))
		fmt.Printf("  Model: %s\n", provider.ModelName())
	} else {
		outputJSON(IndexBuildResult{
			Status:          "complete",
			PapersIndexed:   stats.PapersIndexed,
			PapersSkipped:   stats.PapersSkipped,
			SkippedReason:   stats.SkippedReason,
			DurationSeconds: stats.Duration.Seconds(),
			Model:           provider.ModelName(),
			IndexSizeBytes:  stats.IndexSizeBytes,
		})
	}
}

func runIndexBuild(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	repoRoot := mustFindRepository()

	// Validate Ollama setup
	provider := embedding.NewOllamaProvider()
	validateOllamaSetup(ctx, provider)

	// Open database and get references
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	refs, err := db.ListAll(0)
	if err != nil {
		exitWithError(ExitError, "listing references: %v", err)
	}

	// Build index with progress reporting
	builder := semantic.NewBuilder(provider, db)
	if !noProgress && humanOutput {
		builder.SetProgressReporter(semantic.ProgressFunc(printProgress))
		fmt.Fprintf(os.Stderr, "Building semantic index...\n")
	}

	idx, stats, err := builder.Build(ctx, refs)
	if err != nil {
		exitWithError(ExitError, "building index: %v", err)
	}

	// Save index
	if err := idx.Save(repoRoot); err != nil {
		exitWithError(ExitError, "saving index: %v", err)
	}

	// Get index size (non-fatal if it fails)
	if indexSize, err := semantic.IndexSize(repoRoot); err == nil {
		stats.IndexSizeBytes = indexSize
	} else if humanOutput {
		fmt.Fprintf(os.Stderr, "Warning: could not determine index size: %v\n", err)
	}

	// Clear progress line if we were showing progress
	if humanOutput && !noProgress {
		fmt.Fprintf(os.Stderr, "\r%s\r", "                                                  ")
	}

	outputBuildResults(provider, stats)
	return nil
}

// IndexCheckResult is the response for index check command.
type IndexCheckResult struct {
	Status             string   `json:"status"`
	PapersTotal        int      `json:"papers_total"`
	PapersWithAbstract int      `json:"papers_with_abstract"`
	PapersIndexed      int      `json:"papers_indexed"`
	PapersMissing      int      `json:"papers_missing"`
	MissingIDs         []string `json:"missing_ids,omitempty"`
	Model              string   `json:"model"`
	IndexCreated       string   `json:"index_created"`
	IndexSizeBytes     int64    `json:"index_size_bytes"`
	Recommendation     string   `json:"recommendation,omitempty"`
}

var indexCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check semantic index health",
	Long:  `Check the health and status of the semantic index.`,
	RunE:  runIndexCheck,
}

// findMissingPapers returns IDs of papers with abstracts that are not in the index.
func findMissingPapers(db *storage.DB, idx *semantic.SemanticIndex) ([]string, error) {
	paperIDs, err := db.ListPaperIDsWithAbstract(semantic.MinAbstractLength)
	if err != nil {
		return nil, fmt.Errorf("listing paper IDs: %w", err)
	}

	var missingIDs []string
	for _, id := range paperIDs {
		if !idx.HasPaper(id) {
			missingIDs = append(missingIDs, id)
		}
	}
	return missingIDs, nil
}

// outputCheckResults outputs the index check results in the appropriate format.
func outputCheckResults(result IndexCheckResult, exitCode int) {
	if humanOutput {
		fmt.Printf("Semantic Index Status: %s\n\n", result.Status)
		fmt.Printf("Papers:\n")
		fmt.Printf("  Total in database: %d\n", result.PapersTotal)
		fmt.Printf("  With abstracts: %d\n", result.PapersWithAbstract)
		fmt.Printf("  In semantic index: %d\n", result.PapersIndexed)
		fmt.Printf("  Missing from index: %d\n", result.PapersMissing)
		fmt.Printf("\nIndex Info:\n")
		fmt.Printf("  Model: %s\n", result.Model)
		fmt.Printf("  Created: %s\n", result.IndexCreated)
		fmt.Printf("  Size: %s\n", formatBytes(result.IndexSizeBytes))
		if result.Recommendation != "" {
			fmt.Printf("\n%s\n", result.Recommendation)
		}
	} else {
		outputJSON(result)
	}

	if exitCode != ExitSuccess {
		os.Exit(exitCode)
	}
}

func runIndexCheck(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	// Load index
	idx := mustLoadSemanticIndex(repoRoot)

	// Open database
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Get counts
	totalCount, err := db.Count()
	if err != nil {
		exitWithError(ExitError, "counting references: %v", err)
	}

	abstractCount, err := db.CountPapersWithAbstract(semantic.MinAbstractLength)
	if err != nil {
		exitWithError(ExitError, "counting abstracts: %v", err)
	}

	// Find missing papers
	missingIDs, err := findMissingPapers(db, idx)
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	// Get index size (non-fatal if it fails)
	var indexSize int64
	if size, err := semantic.IndexSize(repoRoot); err == nil {
		indexSize = size
	} else if humanOutput {
		fmt.Fprintf(os.Stderr, "Warning: could not determine index size: %v\n", err)
	}

	// Determine status and exit code
	status := "healthy"
	var recommendation string
	exitCode := ExitSuccess

	if len(missingIDs) > 0 {
		status = "stale"
		recommendation = "Run 'bp index build' to update the index"
		exitCode = ExitIndexStale
	}

	result := IndexCheckResult{
		Status:             status,
		PapersTotal:        totalCount,
		PapersWithAbstract: abstractCount,
		PapersIndexed:      idx.PaperCount,
		PapersMissing:      len(missingIDs),
		Model:              idx.ModelName,
		IndexCreated:       idx.CreatedAt.Format(time.RFC3339),
		IndexSizeBytes:     indexSize,
		Recommendation:     recommendation,
	}

	if len(missingIDs) > 0 && len(missingIDs) <= 10 {
		result.MissingIDs = missingIDs
	}

	outputCheckResults(result, exitCode)
	return nil
}

// progressBarWidth is the width in characters for terminal progress display.
const progressBarWidth = 30

// printProgress prints a progress bar to stderr.
func printProgress(current, total int) {
	if total == 0 {
		return
	}
	pct := float64(current) / float64(total) * 100
	filled := (progressBarWidth * current) / total

	var bar string
	if filled >= progressBarWidth {
		// Complete - all filled
		for i := 0; i < progressBarWidth; i++ {
			bar += "="
		}
	} else {
		// In progress - filled portion, arrow, empty portion
		for i := 0; i < filled; i++ {
			bar += "="
		}
		bar += ">"
		for i := filled + 1; i < progressBarWidth; i++ {
			bar += " "
		}
	}
	fmt.Fprintf(os.Stderr, "\r[%s] %d/%d (%.0f%%)", bar, current, total, pct)
}
