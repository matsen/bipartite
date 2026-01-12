package main

import (
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var listLimit int

func init() {
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Maximum results to return (0 = all)")
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all references",
	Long: `List all references in the repository.

Examples:
  bp list
  bp list --limit 100`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	refs, err := db.ListAll(listLimit)
	if err != nil {
		if humanOutput {
			fmt.Fprintf(os.Stderr, "error: listing references: %v\n", err)
		} else {
			outputJSON(ErrorResponse{Error: fmt.Sprintf("listing references: %v", err)})
		}
		os.Exit(ExitError)
	}

	// Get total count for human output
	total, _ := db.Count()

	if humanOutput {
		if len(refs) == 0 {
			fmt.Println("No references in repository")
		} else {
			if listLimit > 0 && listLimit < total {
				fmt.Printf("%d references (showing first %d):\n\n", total, len(refs))
			} else {
				fmt.Printf("%d references in repository:\n\n", len(refs))
			}
			for _, ref := range refs {
				title := truncateString(ref.Title, 50)
				fmt.Printf("  %-16s %s\n", ref.ID, title)
			}
		}
	} else {
		if refs == nil {
			refs = []storage.Reference{}
		}
		outputJSON(refs)
	}

	return nil
}
