package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rebuildCmd)
}

var rebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild the query layer from source data",
	Long: `Rebuild the SQLite query database from the JSONL source file.

Use this after pulling changes from git or if the database becomes corrupted.`,
	RunE: runRebuild,
}

// RebuildResult is the response for the rebuild command.
type RebuildResult struct {
	Status     string `json:"status"`
	References int    `json:"references"`
}

func runRebuild(cmd *cobra.Command, args []string) error {
	root, exitCode := getRepoRoot()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	// Find repository
	repoRoot, err := config.FindRepository(root)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: err.Error()})
		}
		os.Exit(ExitConfigError)
	}

	// Ensure cache directory exists
	cacheDir := config.CachePath(repoRoot)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: creating cache directory: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("creating cache directory: %v", err)})
		}
		os.Exit(ExitError)
	}

	// Open database
	dbPath := config.DBPath(repoRoot)
	db, err := storage.OpenDB(dbPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: opening database: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("opening database: %v", err)})
		}
		os.Exit(ExitError)
	}
	defer db.Close()

	// Rebuild from JSONL
	refsPath := config.RefsPath(repoRoot)
	count, err := db.RebuildFromJSONL(refsPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: rebuilding database: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("rebuilding database: %v", err)})
		}
		os.Exit(ExitDataError)
	}

	// Output results
	if humanOutput {
		fmt.Printf("Rebuilt query database with %d references\n", count)
	} else {
		outputJSON(RebuildResult{
			Status:     "rebuilt",
			References: count,
		})
	}

	return nil
}
