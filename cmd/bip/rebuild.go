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
	Projects   int    `json:"projects"`
	Repos      int    `json:"repos"`
}

func runRebuild(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	// Ensure cache directory exists
	cacheDir := config.CachePath(repoRoot)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		exitWithError(ExitError, "creating cache directory: %v", err)
	}

	db, err := storage.OpenDB(config.DBPath(repoRoot))
	if err != nil {
		exitWithError(ExitError, "opening database: %v", err)
	}
	defer db.Close()

	refsCount, projectsCount, reposCount, err := rebuildAll(db, repoRoot)
	if err != nil {
		exitWithError(ExitDataError, "%v", err)
	}

	// Output results
	if humanOutput {
		fmt.Printf("Rebuilt query database with %d references, %d projects, and %d repos\n", refsCount, projectsCount, reposCount)
	} else {
		outputJSON(RebuildResult{
			Status:     "rebuilt",
			References: refsCount,
			Projects:   projectsCount,
			Repos:      reposCount,
		})
	}

	return nil
}

// rebuildAll rebuilds the refs, projects and repos tables from their JSONL files.
func rebuildAll(db *storage.DB, repoRoot string) (refs, projects, repos int, err error) {
	if refs, err = db.RebuildFromJSONL(config.RefsPath(repoRoot)); err != nil {
		return 0, 0, 0, fmt.Errorf("rebuilding refs database: %w", err)
	}
	if projects, err = db.RebuildProjectsFromJSONL(config.ProjectsPath(repoRoot)); err != nil {
		return 0, 0, 0, fmt.Errorf("rebuilding projects database: %w", err)
	}
	if repos, err = db.RebuildReposFromJSONL(config.ReposPath(repoRoot)); err != nil {
		return 0, 0, 0, fmt.Errorf("rebuilding repos database: %w", err)
	}
	return refs, projects, repos, nil
}

// dbStale reports whether a JSONL source is newer than the query database,
// as after a git pull that nobody followed with bip rebuild.
func dbStale(repoRoot string) bool {
	db, err := os.Stat(config.DBPath(repoRoot))
	if err != nil {
		return false // OpenDB creates it; nothing to compare yet
	}
	for _, p := range []string{config.RefsPath(repoRoot), config.ProjectsPath(repoRoot), config.ReposPath(repoRoot)} {
		if src, err := os.Stat(p); err == nil && src.ModTime().After(db.ModTime()) {
			return true
		}
	}
	return false
}
