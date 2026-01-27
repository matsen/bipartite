package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

var storeDeleteWhere string

// StoreDeleteResult is the response for store delete command.
type StoreDeleteResult struct {
	Store   string `json:"store"`
	Deleted int    `json:"deleted"`
}

func init() {
	storeCmd.AddCommand(storeDeleteCmd)
	storeDeleteCmd.Flags().StringVarP(&storeDeleteWhere, "where", "w", "", "SQL WHERE clause for batch delete")
}

var storeDeleteCmd = &cobra.Command{
	Use:   "delete <name> [id]",
	Short: "Delete records from a store",
	Long: `Delete one or more records from a store.

Records are removed from the JSONL file. Run 'bip store sync' after deleting
to update the SQLite index.

Examples:
  # Delete by primary key
  bip store delete gh_activity pr-123

  # Delete by condition
  bip store delete gh_activity --where "date < '2025-01-01'"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStoreDelete,
}

func runStoreDelete(cmd *cobra.Command, args []string) error {
	storeName := args[0]
	repoRoot := mustFindRepository()

	// Open store
	s, err := store.OpenStore(repoRoot, storeName)
	if err != nil {
		exitWithError(ExitError, "store %q not found", storeName)
	}

	var deleted int

	if storeDeleteWhere != "" {
		// Delete by WHERE clause
		deleted, err = s.DeleteWhere(storeDeleteWhere)
		if err != nil {
			exitWithError(ExitError, "invalid WHERE clause: %v", err)
		}
	} else if len(args) == 2 {
		// Delete by ID
		id := args[1]
		err = s.DeleteByID(id)
		if err != nil {
			exitWithError(ExitError, "record %q not found", id)
		}
		deleted = 1
	} else {
		exitWithError(ExitError, "provide record ID or use --where for batch delete")
	}

	result := StoreDeleteResult{
		Store:   storeName,
		Deleted: deleted,
	}

	if humanOutput {
		if deleted == 1 {
			fmt.Printf("Deleted 1 record from '%s'\n", storeName)
		} else {
			fmt.Printf("Deleted %d records from '%s'\n", deleted, storeName)
		}
	} else {
		outputJSON(result)
	}

	return nil
}
