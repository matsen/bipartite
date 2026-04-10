package main

import (
	"context"
	"fmt"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/matsen/bipartite/internal/zotero"
	"github.com/spf13/cobra"
)

var zoteroSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync references from Zotero library into bip",
	Long: `Pull all items from your Zotero library and import them into bip.

New items are added, existing items (matched by Zotero key, DOI, or ID)
are updated with the latest metadata.

Examples:
  bip zotero sync
  bip zotero sync --human
  bip zotero sync --dry-run --human`,
	Args: cobra.NoArgs,
	RunE: runZoteroSync,
}

var zoteroSyncDryRun bool

func init() {
	zoteroCmd.AddCommand(zoteroSyncCmd)
	zoteroSyncCmd.Flags().BoolVar(&zoteroSyncDryRun, "dry-run", false, "Show what would be synced without writing")
}

// ZoteroSyncResult is the JSON output for the sync command.
type ZoteroSyncResult struct {
	Action  string `json:"action"` // synced, dry_run
	Fetched int    `json:"fetched"`
	New     int    `json:"new"`
	Updated int    `json:"updated"`
	Skipped int    `json:"skipped"`
	Errors  int    `json:"errors"`
}

func runZoteroSync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Zotero client
	client, err := zotero.NewClient()
	if err != nil {
		return outputZoteroError(ExitZoteroNotConfigured, "Zotero not configured", err)
	}

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	existingRefs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputZoteroError(ExitZoteroAPIError, "reading refs", err)
	}

	// Fetch items from Zotero
	if humanOutput {
		fmt.Println("Fetching items from Zotero...")
	}

	items, err := client.GetItems(ctx)
	if err != nil {
		return outputZoteroError(ExitZoteroAPIError, "fetching items from Zotero", err)
	}

	if humanOutput {
		fmt.Printf("Fetched %d items from Zotero\n", len(items))
	}

	// Convert Zotero items to references
	var convertedRefs []reference.Reference
	var convertErrors int

	for _, item := range items {
		ref, err := zotero.MapZoteroToReference(item)
		if err != nil {
			convertErrors++
			continue
		}
		convertedRefs = append(convertedRefs, ref)
	}

	// Classify using the same processImports pipeline as file import
	stats, _, newRefs := processImports(convertedRefs, existingRefs)

	result := ZoteroSyncResult{
		Fetched: len(items),
		New:     stats.newCount,
		Updated: stats.updated,
		Skipped: stats.skipped,
		Errors:  convertErrors,
	}

	if zoteroSyncDryRun {
		result.Action = "dry_run"
		if humanOutput {
			fmt.Printf("\nDry run - would sync from Zotero:\n")
			fmt.Printf("  Would add:    %d new references\n", stats.newCount)
			fmt.Printf("  Would update: %d existing references\n", stats.updated)
			fmt.Printf("  Skipped:      %d (no changes)\n", stats.skipped)
			if convertErrors > 0 {
				fmt.Printf("  Errors:       %d (missing required fields)\n", convertErrors)
			}
		} else {
			outputJSON(result)
		}
		return nil
	}

	// Actually persist
	if err := persistImports(refsPath, existingRefs, newRefs); err != nil {
		return outputZoteroError(ExitZoteroAPIError, "writing refs", err)
	}

	result.Action = "synced"
	if humanOutput {
		fmt.Printf("\nSynced from Zotero:\n")
		fmt.Printf("  Added:   %d new references\n", stats.newCount)
		fmt.Printf("  Updated: %d existing references\n", stats.updated)
		fmt.Printf("  Skipped: %d (no changes)\n", stats.skipped)
		if convertErrors > 0 {
			fmt.Printf("  Errors:  %d (missing required fields)\n", convertErrors)
		}
		if stats.newCount > 0 || stats.updated > 0 {
			fmt.Println("\nRun 'bip rebuild' to update the search index.")
		}
	} else {
		outputJSON(result)
	}

	return nil
}

func outputZoteroError(exitCode int, context string, err error) error {
	return outputGenericError(exitCode, "zotero_error", context, err)
}
