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
		return outputZoteroError(ExitConfigError, "Zotero not configured", err)
	}

	// Find repository
	repoRoot := mustFindRepository()
	refsPath := config.RefsPath(repoRoot)

	// Load existing refs
	existingRefs, err := storage.ReadAll(refsPath)
	if err != nil {
		return outputZoteroError(ExitError, "reading refs", err)
	}

	// Fetch items from Zotero
	if humanOutput {
		fmt.Println("Fetching items from Zotero...")
	}

	items, err := client.GetItems(ctx)
	if err != nil {
		return outputZoteroError(ExitError, "fetching items from Zotero", err)
	}

	if humanOutput {
		fmt.Printf("Fetched %d items from Zotero\n", len(items))
	}

	// Convert Zotero items to references and classify
	var newRefs []storage.RefWithAction
	var stats importStats
	var convertErrors int

	// Build working set for deduplication (grows as we add new refs)
	workingSet := make([]reference.Reference, len(existingRefs))
	copy(workingSet, existingRefs)

	for _, item := range items {
		ref, err := zotero.MapZoteroToReference(item)
		if err != nil {
			convertErrors++
			continue
		}

		action := classifyImport(workingSet, ref)

		switch action.action {
		case "new":
			ref.ID = storage.GenerateUniqueID(workingSet, ref.ID)
			newRefs = append(newRefs, storage.RefWithAction{Ref: ref, Action: "new"})
			workingSet = append(workingSet, ref)
			stats.newCount++
		case "update":
			if action.existingIdx < len(existingRefs) {
				// Check if anything actually changed before writing
				existing := existingRefs[action.existingIdx]
				if refsMetadataEqual(existing, ref) {
					stats.skipped++
				} else {
					// Preserve the existing bip ID
					ref.ID = existing.ID
					newRefs = append(newRefs, storage.RefWithAction{Ref: ref, Action: "update", ExistingIdx: action.existingIdx})
					stats.updated++
				}
			} else {
				// Within-batch duplicate
				stats.skipped++
			}
		case "skip":
			stats.skipped++
		}
	}

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
		return outputZoteroError(ExitError, "writing refs", err)
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

// refsMetadataEqual checks if two references have the same metadata.
// Used to skip unnecessary updates during sync.
func refsMetadataEqual(a, b reference.Reference) bool {
	if a.Title != b.Title || a.DOI != b.DOI || a.Abstract != b.Abstract {
		return false
	}
	if a.Venue != b.Venue || a.Note != b.Note {
		return false
	}
	if a.Published != b.Published {
		return false
	}
	if a.PMID != b.PMID || a.PMCID != b.PMCID || a.ArXivID != b.ArXivID {
		return false
	}
	if len(a.Authors) != len(b.Authors) {
		return false
	}
	for i := range a.Authors {
		if a.Authors[i].First != b.Authors[i].First || a.Authors[i].Last != b.Authors[i].Last {
			return false
		}
	}
	return true
}
