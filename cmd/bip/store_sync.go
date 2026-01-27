package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

var storeSyncAll bool

// StoreSyncResult is the response for store sync command.
type StoreSyncResult struct {
	Store   string `json:"store"`
	Records int    `json:"records"`
	Action  string `json:"action"` // "rebuilt" or "skipped"
}

// StoreSyncAllResult is the response for store sync --all command.
type StoreSyncAllResult struct {
	Results []StoreSyncResult `json:"results"`
}

func init() {
	storeCmd.AddCommand(storeSyncCmd)
	storeSyncCmd.Flags().BoolVarP(&storeSyncAll, "all", "a", false, "Sync all registered stores")
}

var storeSyncCmd = &cobra.Command{
	Use:   "sync [name]",
	Short: "Sync JSONL to SQLite index",
	Long: `Rebuild the SQLite query index from the JSONL source of truth.

Use this after manually editing JSONL files or after pulling changes from git.

Example:
  bip store sync gh_activity    # Sync single store
  bip store sync --all          # Sync all stores`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStoreSync,
}

func runStoreSync(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	if storeSyncAll {
		return runStoreSyncAll(repoRoot)
	}

	if len(args) == 0 {
		exitWithError(ExitError, "store name required (or use --all)")
	}

	storeName := args[0]

	// Open store
	s, err := store.OpenStore(repoRoot, storeName)
	if err != nil {
		exitWithError(ExitError, "store %q not found", storeName)
	}

	// Check if sync is needed
	needsSync, err := s.NeedsSync()
	if err != nil {
		exitWithError(ExitError, "checking sync status: %v", err)
	}

	var result StoreSyncResult
	result.Store = storeName

	if !needsSync {
		// Get record count even when skipped
		count, _ := s.Count()
		result.Records = count
		result.Action = "skipped"
	} else {
		// Perform sync
		count, err := s.Sync()
		if err != nil {
			exitWithError(ExitDataError, "syncing store: %v", err)
		}
		result.Records = count
		result.Action = "rebuilt"
	}

	if humanOutput {
		if result.Action == "skipped" {
			fmt.Printf("'%s' already in sync (skipped)\n", storeName)
		} else {
			fmt.Printf("Synced '%s': %d records (rebuilt)\n", storeName, result.Records)
		}
	} else {
		outputJSON(result)
	}

	return nil
}

func runStoreSyncAll(repoRoot string) error {
	registry, err := store.LoadRegistry(repoRoot)
	if err != nil {
		exitWithError(ExitError, "loading registry: %v", err)
	}

	if len(registry.Stores) == 0 {
		if humanOutput {
			fmt.Println("No stores registered")
		} else {
			outputJSON(StoreSyncAllResult{Results: []StoreSyncResult{}})
		}
		return nil
	}

	var results []StoreSyncResult

	for name := range registry.Stores {
		s, err := store.OpenStore(repoRoot, name)
		if err != nil {
			if humanOutput {
				fmt.Printf("Error opening '%s': %v\n", name, err)
			}
			continue
		}

		needsSync, err := s.NeedsSync()
		if err != nil {
			if humanOutput {
				fmt.Printf("Error checking '%s': %v\n", name, err)
			}
			continue
		}

		var result StoreSyncResult
		result.Store = name

		if !needsSync {
			count, _ := s.Count()
			result.Records = count
			result.Action = "skipped"
		} else {
			count, err := s.Sync()
			if err != nil {
				if humanOutput {
					fmt.Printf("Error syncing '%s': %v\n", name, err)
				}
				continue
			}
			result.Records = count
			result.Action = "rebuilt"
		}

		results = append(results, result)

		if humanOutput {
			if result.Action == "skipped" {
				fmt.Printf("'%s' already in sync (skipped)\n", name)
			} else {
				fmt.Printf("Synced '%s': %d records (rebuilt)\n", name, result.Records)
			}
		}
	}

	if !humanOutput {
		outputJSON(StoreSyncAllResult{Results: results})
	}

	return nil
}
