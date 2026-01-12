// Package main provides the bp CLI entry point.
package main

import (
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

// humanOutput controls whether to use human-readable output
var humanOutput bool

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitError)
	}
}

var rootCmd = &cobra.Command{
	Use:   "bp",
	Short: "Agent-first academic reference manager",
	Long: `bp is an agent-first CLI for managing academic references.

It stores references in git-versionable JSONL format with an ephemeral
SQLite database for fast queries. All commands output JSON by default
for easy integration with AI agents and other tools.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&humanOutput, "human", false, "Use human-readable output instead of JSON")
	rootCmd.Version = Version
}

// getStartingDirectory returns the directory to start searching for a repository.
// It checks the BP_ROOT environment variable first, then falls back to the current working directory.
func getStartingDirectory() (string, int) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", outputError(ExitError, "getting current directory: %v", err)
	}

	// Check BP_ROOT environment variable first
	if root := os.Getenv("BP_ROOT"); root != "" {
		return root, 0
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
		exitWithError(ExitConfigError, "%v", err)
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
