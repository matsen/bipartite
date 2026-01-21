package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new bipartite repository",
	Long: `Initialize a new bipartite repository in the current directory.

Creates:
  .bipartite/
  ├── refs.jsonl      # Empty file
  ├── config.json     # Default config
  └── cache/          # Empty directory (gitignored)`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Use getStartingDirectory (not mustFindRepository) since the repo doesn't exist yet
	root, exitCode := getStartingDirectory()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	// Check if already initialized
	if config.IsRepository(root) {
		exitWithError(ExitError, "directory already contains a bipartite repository")
	}

	// Create directory structure
	bpDir := config.BipartitePath(root)
	if err := os.MkdirAll(bpDir, 0755); err != nil {
		exitWithError(ExitError, "creating .bipartite directory: %v", err)
	}

	// Create cache directory
	cacheDir := config.CachePath(root)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		exitWithError(ExitError, "creating cache directory: %v", err)
	}

	// Create empty refs.jsonl
	refsPath := config.RefsPath(root)
	refsFile, err := os.Create(refsPath)
	if err != nil {
		exitWithError(ExitError, "creating refs.jsonl: %v", err)
	}
	refsFile.Close()

	// Create default config
	cfg := &config.Config{
		PDFRoot:   "",
		PDFReader: "system",
	}
	if err := cfg.Save(root); err != nil {
		exitWithError(ExitError, "creating config.json: %v", err)
	}

	// Output success
	if humanOutput {
		fmt.Printf("Initialized bipartite repository in %s\n", root)
	} else {
		outputJSON(StatusResponse{
			Status: "initialized",
			Path:   root,
		})
	}

	return nil
}
