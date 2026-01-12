// Package main provides the bp CLI entry point.
package main

import (
	"os"

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

// getRepoRoot returns the repository root, or exits with an error if not in a repository.
func getRepoRoot() (string, int) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", outputError(ExitError, "getting current directory: %v", err)
	}

	// Check BP_ROOT environment variable first
	if root := os.Getenv("BP_ROOT"); root != "" {
		cwd = root
	}

	return cwd, 0
}
