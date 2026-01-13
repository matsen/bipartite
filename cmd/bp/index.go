package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/matsen/bipartite/internal/embedding"
	"github.com/matsen/bipartite/internal/semantic"
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

func runIndexBuild(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	repoRoot := mustFindRepository()

	// Check Ollama availability
	provider := embedding.NewOllamaProvider()
	if err := provider.IsAvailable(ctx); err != nil {
		exitWithError(ExitDataError, "Ollama is not running\n\nStart Ollama with 'ollama serve' or install from https://ollama.ai")
	}

	// Check model availability
	hasModel, err := provider.HasModel(ctx)
	if err != nil {
		exitWithError(ExitError, "checking model availability: %v", err)
	}
	if !hasModel {
		exitWithError(ExitModelNotFound, "Embedding model '%s' not found\n\nRun 'ollama pull %s' to download it.", provider.ModelName(), provider.ModelName())
	}

	// Open database
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Get all references
	refs, err := db.ListAll(0)
	if err != nil {
		exitWithError(ExitError, "listing references: %v", err)
	}

	// Build index
	builder := semantic.NewBuilder(provider, db)

	// Set progress reporter unless suppressed
	if !noProgress && humanOutput {
		builder.SetProgressReporter(semantic.ProgressFunc(func(current, total int) {
			printProgress(current, total)
		}))
	}

	if humanOutput && !noProgress {
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

	// Get index size
	indexSize, err := semantic.IndexSize(repoRoot)
	if err != nil {
		indexSize = 0 // Non-fatal
	}
	stats.IndexSizeBytes = indexSize

	// Clear progress line if we were showing progress
	if humanOutput && !noProgress {
		fmt.Fprintf(os.Stderr, "\r%s\r", "                                                  ")
	}

	// Output results
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

func runIndexCheck(cmd *cobra.Command, args []string) error {
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
	paperIDs, err := db.ListPaperIDsWithAbstract(semantic.MinAbstractLength)
	if err != nil {
		exitWithError(ExitError, "listing paper IDs: %v", err)
	}

	var missingIDs []string
	for _, id := range paperIDs {
		if !idx.HasPaper(id) {
			missingIDs = append(missingIDs, id)
		}
	}

	// Get index size
	indexSize, _ := semantic.IndexSize(repoRoot)

	// Determine status
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

	// Output results
	if humanOutput {
		fmt.Printf("Semantic Index Status: %s\n\n", status)
		fmt.Printf("Papers:\n")
		fmt.Printf("  Total in database: %d\n", totalCount)
		fmt.Printf("  With abstracts: %d\n", abstractCount)
		fmt.Printf("  In semantic index: %d\n", idx.PaperCount)
		fmt.Printf("  Missing from index: %d\n", len(missingIDs))
		fmt.Printf("\nIndex Info:\n")
		fmt.Printf("  Model: %s\n", idx.ModelName)
		fmt.Printf("  Created: %s\n", idx.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Size: %s\n", formatBytes(indexSize))
		if recommendation != "" {
			fmt.Printf("\n%s\n", recommendation)
		}
	} else {
		outputJSON(result)
	}

	if exitCode != ExitSuccess {
		os.Exit(exitCode)
	}
	return nil
}

// printProgress prints a progress bar to stderr.
func printProgress(current, total int) {
	if total == 0 {
		return
	}
	pct := float64(current) / float64(total) * 100
	barWidth := 30
	filled := int(float64(barWidth) * float64(current) / float64(total))
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "="
		} else if i == filled {
			bar += ">"
		} else {
			bar += " "
		}
	}
	fmt.Fprintf(os.Stderr, "\r[%s] %d/%d (%.0f%%)", bar, current, total, pct)
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// formatBytes formats bytes in a human-readable way.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
