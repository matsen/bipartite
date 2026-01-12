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
	root, exitCode := getRepoRoot()
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	// Check if already initialized
	if config.IsRepository(root) {
		if humanOutput {
			fmt.Fprintln(os.Stderr, "error: directory already contains a bipartite repository")
		} else {
			outputJSON(ErrorResponse{Error: "directory already contains a bipartite repository"})
		}
		os.Exit(ExitError)
	}

	// Create directory structure
	bpDir := config.BipartitePath(root)
	if err := os.MkdirAll(bpDir, 0755); err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: creating .bipartite directory: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("creating .bipartite directory: %v", err)})
		}
		os.Exit(ExitError)
	}

	// Create cache directory
	cacheDir := config.CachePath(root)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: creating cache directory: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("creating cache directory: %v", err)})
		}
		os.Exit(ExitError)
	}

	// Create empty refs.jsonl
	refsPath := config.RefsPath(root)
	refsFile, err := os.Create(refsPath)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: creating refs.jsonl: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("creating refs.jsonl: %v", err)})
		}
		os.Exit(ExitError)
	}
	refsFile.Close()

	// Create default config
	cfg := &config.Config{
		PDFRoot:   "",
		PDFReader: "system",
	}
	if err := cfg.Save(root); err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: creating config.json: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("creating config.json: %v", err)})
		}
		os.Exit(ExitError)
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
