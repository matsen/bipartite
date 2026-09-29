// Package main provides the bip CLI entry point.
package main

import (
	"fmt"
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
	rejectUnknownSubcommands(rootCmd)
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
  - Academic reference library (papers, projects, repos)
  - GitHub project tracking (issues, PRs, boards, activity digests)
  - Slack integration for team updates
  - Remote server availability checking

Data is stored in git-versionable JSONL with ephemeral SQLite for queries.
All commands output JSON by default for AI agent integration.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// rejectUnknownSubcommands makes every command group below root fail on an
// unknown subcommand. Cobra only does this at root; a group like `bip epic`
// otherwise prints its help and exits 0, so a script calling a removed
// subcommand reads success. Run it after every init has registered commands.
func rejectUnknownSubcommands(cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		if c.HasSubCommands() && !c.Runnable() {
			c.Args = cobra.NoArgs
			c.RunE = func(cmd *cobra.Command, args []string) error { return cmd.Help() }
		}
		rejectUnknownSubcommands(c)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&humanOutput, "human", false, "Use human-readable output instead of JSON")
	rootCmd.Version = Version
}

// getStartingDirectory returns the directory to start searching for a repository.
// Prefers nexus_path from global config, falls back to current working directory.
func getStartingDirectory() (string, int) {
	// Try global config first
	if root := config.GetNexusPath(); root != "" {
		return root, 0
	}

	// Fall back to current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		return "", ExitError
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
		// Show helpful message with tip about global config
		fmt.Fprintln(os.Stderr, config.HelpfulConfigMessage())
		os.Exit(ExitConfigError)
	}
	return repoRoot
}

// mustOpenDatabase opens the SQLite database, exits on error.
// The caller is responsible for calling Close() on the returned DB.
// A database older than its JSONL is rebuilt first, so a query never
// silently misses refs that a pull brought in.
func mustOpenDatabase(repoRoot string) *storage.DB {
	stale := dbStale(repoRoot)
	dbPath := config.DBPath(repoRoot)
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		exitWithError(ExitError, "opening database: %v", err)
	}
	// Silent: callers parse stdout and stderr together, and commands that write
	// a JSONL and then open the database land here on every edit.
	if stale {
		if _, _, _, err := rebuildAll(db, repoRoot); err != nil {
			exitWithError(ExitDataError, "%v", err)
		}
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
