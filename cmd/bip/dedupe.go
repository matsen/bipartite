package main

import (
	"fmt"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	dedupeDryRun bool
	dedupeMerge  bool
)

func init() {
	dedupeCmd.Flags().BoolVar(&dedupeDryRun, "dry-run", false, "Show duplicates without making changes")
	dedupeCmd.Flags().BoolVar(&dedupeMerge, "merge", false, "Merge duplicates (keep first, update edges)")
	rootCmd.AddCommand(dedupeCmd)
}

var dedupeCmd = &cobra.Command{
	Use:   "dedupe",
	Short: "Find and remove duplicate references",
	Long: `Find and remove duplicate references by their import source ID.

Examples:
  bip dedupe --dry-run    # Show duplicates without making changes
  bip dedupe --merge      # Merge duplicates: keep first, remove others, update edges`,
	RunE: runDedupe,
}

// DuplicateGroup represents a set of duplicate references.
type DuplicateGroup struct {
	SourceType string   `json:"source_type"`
	SourceID   string   `json:"source_id"`
	Primary    string   `json:"primary"`    // ID of the entry to keep
	Duplicates []string `json:"duplicates"` // IDs of entries to remove
}

// DedupeResult represents the result of a dedupe operation.
type DedupeResult struct {
	DryRun        bool             `json:"dry_run"`
	Groups        []DuplicateGroup `json:"groups"`
	TotalDupes    int              `json:"total_duplicates"`
	EdgesModified int              `json:"edges_modified,omitempty"`
}

func runDedupe(cmd *cobra.Command, args []string) error {
	if !dedupeDryRun && !dedupeMerge {
		return fmt.Errorf("must specify either --dry-run or --merge")
	}
	if dedupeDryRun && dedupeMerge {
		return fmt.Errorf("cannot specify both --dry-run and --merge")
	}

	repoRoot := mustFindRepository()

	// Load all references
	refsPath := config.RefsPath(repoRoot)
	refs, err := storage.ReadAll(refsPath)
	if err != nil {
		exitWithError(ExitDataError, "reading refs: %v", err)
	}

	// Find duplicates by source ID
	groups := findDuplicateGroups(refs)

	if len(groups) == 0 {
		if humanOutput {
			fmt.Println("No duplicates found.")
		} else {
			outputJSON(DedupeResult{DryRun: dedupeDryRun})
		}
		return nil
	}

	totalDupes := 0
	for _, g := range groups {
		totalDupes += len(g.Duplicates)
	}

	if dedupeDryRun {
		if humanOutput {
			fmt.Printf("Found %d duplicate groups (%d total duplicates):\n\n", len(groups), totalDupes)
			for _, g := range groups {
				fmt.Printf("Source: %s/%s\n", g.SourceType, g.SourceID)
				fmt.Printf("  Keep:   %s\n", g.Primary)
				fmt.Printf("  Remove: %v\n\n", g.Duplicates)
			}
		} else {
			outputJSON(DedupeResult{
				DryRun:     true,
				Groups:     groups,
				TotalDupes: totalDupes,
			})
		}
		return nil
	}

	// Merge mode: actually remove duplicates and update edges
	edgesModified, err := performMerge(repoRoot, refs, groups)
	if err != nil {
		exitWithError(ExitDataError, "performing merge: %v", err)
	}

	if humanOutput {
		fmt.Printf("Merged %d duplicate groups (%d duplicates removed)\n", len(groups), totalDupes)
		if edgesModified > 0 {
			fmt.Printf("Modified %d edges\n", edgesModified)
		}
	} else {
		outputJSON(DedupeResult{
			DryRun:        false,
			Groups:        groups,
			TotalDupes:    totalDupes,
			EdgesModified: edgesModified,
		})
	}

	return nil
}

// sourceKey is a composite key for grouping references by import source.
type sourceKey struct {
	Type string
	ID   string
}

// findDuplicateGroups finds references with the same source ID.
func findDuplicateGroups(refs []reference.Reference) []DuplicateGroup {
	// Map source key -> list of ref IDs
	sourceMap := make(map[sourceKey][]string)

	for _, ref := range refs {
		if ref.Source.ID == "" {
			continue // Skip refs without source ID
		}
		key := sourceKey{Type: ref.Source.Type, ID: ref.Source.ID}
		sourceMap[key] = append(sourceMap[key], ref.ID)
	}

	// Build duplicate groups (only where there are 2+ entries)
	var groups []DuplicateGroup
	for key, ids := range sourceMap {
		if len(ids) < 2 {
			continue
		}

		groups = append(groups, DuplicateGroup{
			SourceType: key.Type,
			SourceID:   key.ID,
			Primary:    ids[0],  // Keep first occurrence
			Duplicates: ids[1:], // Remove rest
		})
	}

	return groups
}

// performMerge removes duplicates and updates edge references.
func performMerge(repoRoot string, refs []reference.Reference, groups []DuplicateGroup) (int, error) {
	// Build redirect map: duplicate ID -> primary ID
	redirectMap := make(map[string]string)
	dupeSet := make(map[string]bool)
	for _, g := range groups {
		for _, dupeID := range g.Duplicates {
			redirectMap[dupeID] = g.Primary
			dupeSet[dupeID] = true
		}
	}

	// Filter out duplicates from refs
	var cleanRefs []reference.Reference
	for _, ref := range refs {
		if !dupeSet[ref.ID] {
			cleanRefs = append(cleanRefs, ref)
		}
	}

	// Write cleaned refs
	refsPath := config.RefsPath(repoRoot)
	if err := storage.WriteAll(refsPath, cleanRefs); err != nil {
		return 0, fmt.Errorf("writing refs: %w", err)
	}

	// Update edges that reference duplicates
	edgesPath := config.EdgesPath(repoRoot)
	edges, err := storage.ReadAllEdges(edgesPath)
	if err != nil {
		return 0, fmt.Errorf("reading edges: %w", err)
	}

	edgesModified := 0
	for i := range edges {
		modified := false
		if newID, ok := redirectMap[edges[i].SourceID]; ok {
			edges[i].SourceID = newID
			modified = true
		}
		if newID, ok := redirectMap[edges[i].TargetID]; ok {
			edges[i].TargetID = newID
			modified = true
		}
		if modified {
			edgesModified++
		}
	}

	if edgesModified > 0 {
		if err := storage.WriteAllEdges(edgesPath, edges); err != nil {
			return 0, fmt.Errorf("writing edges: %w", err)
		}
	}

	return edgesModified, nil
}
