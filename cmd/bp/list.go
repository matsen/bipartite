package main

import (
	"fmt"

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
	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	refs, err := db.ListAll(listLimit)
	if err != nil {
		exitWithError(ExitError, "listing references: %v", err)
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
				title := truncateString(ref.Title, ListTitleTruncateLen)
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
