// Package main provides the bip CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/embedding"
	"github.com/matsen/bipartite/internal/semantic"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

// humanOutput controls whether to use human-readable output
var humanOutput bool

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Print the error since we have SilenceErrors: true
		// This ensures Cobra errors (like missing required flags) are visible
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(ExitError)
	}
}

var rootCmd = &cobra.Command{
	Use:   "bip",
	Short: "Agent-first research workflow CLI",
	Long: `bip is an agent-first CLI for research workflows.

Core features:
  - Academic references with knowledge graph (papers, concepts, edges)
  - Semantic search via embeddings
  - GitHub project tracking (issues, PRs, boards, activity digests)
  - Slack integration for team updates
  - Remote server availability checking

Data is stored in git-versionable JSONL with ephemeral SQLite for queries.
All commands output JSON by default for AI agent integration.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&humanOutput, "human", false, "Use human-readable output instead of JSON")
	rootCmd.Version = Version
}

// getStartingDirectory returns the directory to start searching for a repository.
// Checks global config nexus_path first, then current working directory.
func getStartingDirectory() (string, int) {
	if root := config.GetNexusPath(); root != "" {
		return root, 0
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", outputError(ExitError, "getting current directory: %v", err)
	}
	return cwd, 0
}

// mustFindRepository finds and validates the repository, exits on error.
// Returns the repository root path.
func mustFindRepository() string {
	start, exitCode := getStartingDirectory()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	repoRoot, err := config.FindRepository(start)
	if err != nil {
		// Show helpful message if no global config exists
		fmt.Fprintln(os.Stderr, config.HelpfulConfigMessage())
		os.Exit(ExitConfigError)
	}
	return repoRoot
}

// mustOpenDatabase opens the SQLite database, exits on error.
// The caller is responsible for calling Close() on the returned DB.
func mustOpenDatabase(repoRoot string) *storage.DB {
	dbPath := config.DBPath(repoRoot)
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		exitWithError(ExitError, "opening database: %v", err)
	}
	return db
}

// mustLoadConfig loads configuration, exits on error.
func mustLoadConfig(repoRoot string) *config.Config {
	cfg, err := config.Load(repoRoot)
	if err != nil {
		exitWithError(ExitConfigError, "loading config: %v", err)
	}
	return cfg
}

// mustLoadSemanticIndex loads the semantic index, exits on error.
func mustLoadSemanticIndex(repoRoot string) *semantic.SemanticIndex {
	idx, err := semantic.Load(repoRoot)
	if err != nil {
		if err == semantic.ErrIndexNotFound {
			exitWithError(ExitConfigError, "Semantic index not found\n\nRun 'bip index build' to create the index.")
		}
		exitWithError(ExitError, "loading index: %v", err)
	}
	return idx
}

// mustValidateOllama checks that Ollama is running and optionally validates the model.
// If checkModel is true, also verifies the required embedding model is available.
func mustValidateOllama(ctx context.Context, provider *embedding.OllamaProvider, checkModel bool) {
	if err := provider.IsAvailable(ctx); err != nil {
		exitWithError(ExitDataError, "Ollama is not running\n\nStart Ollama with 'ollama serve' or install from https://ollama.ai")
	}

	if checkModel {
		hasModel, err := provider.HasModel(ctx)
		if err != nil {
			exitWithError(ExitError, "checking model availability: %v", err)
		}
		if !hasModel {
			exitWithError(ExitModelNotFound, "embedding model %q not found\n\nRun 'ollama pull %s' to download it.", provider.ModelName(), provider.ModelName())
		}
	}
}
