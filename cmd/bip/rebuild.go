package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
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
	Edges      int    `json:"edges"`
	Concepts   int    `json:"concepts"`
}

func runRebuild(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	// Ensure cache directory exists
	cacheDir := config.CachePath(repoRoot)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		exitWithError(ExitError, "creating cache directory: %v", err)
	}

	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	// Rebuild refs from JSONL
	refsPath := config.RefsPath(repoRoot)
	refsCount, err := db.RebuildFromJSONL(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "rebuilding refs database: %v", err)
	}

	// Rebuild edges from JSONL
	edgesPath := config.EdgesPath(repoRoot)
	edgesCount, err := db.RebuildEdgesFromJSONL(edgesPath)
	if err != nil {
		exitWithError(ExitDataError, "rebuilding edges database: %v", err)
	}

	// Rebuild concepts from JSONL
	conceptsPath := config.ConceptsPath(repoRoot)
	conceptsCount, err := db.RebuildConceptsFromJSONL(conceptsPath)
	if err != nil {
		exitWithError(ExitDataError, "rebuilding concepts database: %v", err)
	}

	// Output results
	if humanOutput {
		fmt.Printf("Rebuilt query database with %d references, %d edges, and %d concepts\n", refsCount, edgesCount, conceptsCount)
	} else {
		outputJSON(RebuildResult{
			Status:     "rebuilt",
			References: refsCount,
			Edges:      edgesCount,
			Concepts:   conceptsCount,
		})
	}

	return nil
}
