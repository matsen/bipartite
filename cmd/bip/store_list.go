package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

// StoreListItem represents a store in list output.
type StoreListItem struct {
	Name    string `json:"name"`
	Records int    `json:"records"`
	Path    string `json:"path"`
}

func init() {
	storeCmd.AddCommand(storeListCmd)
}

var storeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered stores",
	Long: `List all stores registered in .bipartite/stores.json.

Example:
  bip store list`,
	Args: cobra.NoArgs,
	RunE: runStoreList,
}

func runStoreList(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	stores, err := store.ListStores(repoRoot)
	if err != nil {
		exitWithError(ExitError, "listing stores: %v", err)
	}

	if len(stores) == 0 {
		if humanOutput {
			fmt.Println("No stores registered")
		} else {
			outputJSON([]StoreListItem{})
		}
		return nil
	}

	if humanOutput {
		// Calculate column widths
		nameWidth := 4    // "NAME"
		recordsWidth := 7 // "RECORDS"
		for _, s := range stores {
			if len(s.Name) > nameWidth {
				nameWidth = len(s.Name)
			}
			recStr := fmt.Sprintf("%d", s.Records)
			if len(recStr) > recordsWidth {
				recordsWidth = len(recStr)
			}
		}

		// Print header
		fmt.Printf("%s  %s  %s\n",
			padRight("NAME", nameWidth),
			padRight("RECORDS", recordsWidth),
			"PATH")

		// Print rows
		for _, s := range stores {
			recStr := fmt.Sprintf("%d", s.Records)
			fmt.Printf("%s  %s  %s\n",
				padRight(s.Name, nameWidth),
				padLeft(recStr, recordsWidth),
				s.JSONLPath)
		}
	} else {
		var items []StoreListItem
		for _, s := range stores {
			items = append(items, StoreListItem{
				Name:    s.Name,
				Records: s.Records,
				Path:    s.JSONLPath,
			})
		}
		outputJSON(items)
	}

	return nil
}

// padLeft pads a string with spaces on the left.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
