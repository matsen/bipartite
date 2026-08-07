package main

import (
	"fmt"
	"sort"
	"unicode"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/importer"
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
	dedupeCmd.Flags().BoolVar(&dedupeMerge, "merge", false, "Merge duplicates (keep first, remove others)")
	rootCmd.AddCommand(dedupeCmd)
}

var dedupeCmd = &cobra.Command{
	Use:   "dedupe",
	Short: "Find and remove duplicate references",
	Long: `Find duplicate references by import source ID, and report references
that share a normalized title (a broader signal that includes source-ID
duplicates plus other likely matches for human review).

Only source-ID groups are actionable by --merge; title groups are reported
for review and never merged automatically.

Examples:
  bip dedupe --dry-run    # Show duplicates without making changes
  bip dedupe --merge      # Merge source-ID duplicates: keep first, remove others`,
	RunE: runDedupe,
}

// DuplicateGroup represents a set of duplicate or likely-duplicate references.
// MatchBasis is "source_id" (actionable by --merge) or "title" (report only).
type DuplicateGroup struct {
	MatchBasis string   `json:"match_basis"`
	SourceType string   `json:"source_type,omitempty"`
	SourceID   string   `json:"source_id,omitempty"`
	Primary    string   `json:"primary,omitempty"`    // ID of the entry to keep; set for match_basis=="source_id"
	Duplicates []string `json:"duplicates,omitempty"` // IDs of entries to remove; set for match_basis=="source_id"
	Members    []string `json:"members,omitempty"`    // all ids; set for match_basis=="title"
	Title      string   `json:"title,omitempty"`      // normalized title; set for match_basis=="title"
}

// DedupeResult represents the result of a dedupe operation.
type DedupeResult struct {
	DryRun     bool             `json:"dry_run"`
	Groups     []DuplicateGroup `json:"groups"`
	TotalDupes int              `json:"total_duplicates"`
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

	// TotalDupes counts only source-id groups; title groups leave
	// Duplicates empty since --merge never acts on them.
	totalDupes := 0
	for _, g := range groups {
		totalDupes += len(g.Duplicates)
	}

	if dedupeDryRun {
		if humanOutput {
			fmt.Printf("Found %d duplicate groups (%d total duplicates):\n\n", len(groups), totalDupes)
			for _, g := range groups {
				switch g.MatchBasis {
				case "source_id":
					fmt.Printf("Source: %s/%s\n", g.SourceType, g.SourceID)
					fmt.Printf("  Keep:   %s\n", g.Primary)
					fmt.Printf("  Remove: %v\n\n", g.Duplicates)
				case "title":
					fmt.Printf("Title match: %q\n", g.Title)
					fmt.Printf("  Members: %v\n\n", g.Members)
				}
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

	// Merge mode only acts on source-id groups; title groups are report-only.
	mergeGroups := make([]DuplicateGroup, 0, len(groups))
	for _, g := range groups {
		if g.MatchBasis == "source_id" {
			mergeGroups = append(mergeGroups, g)
		}
	}

	if err := performMerge(repoRoot, refs, mergeGroups); err != nil {
		exitWithError(ExitDataError, "performing merge: %v", err)
	}

	if humanOutput {
		fmt.Printf("Merged %d duplicate groups (%d duplicates removed)\n", len(mergeGroups), totalDupes)
	} else {
		outputJSON(DedupeResult{
			DryRun:     false,
			Groups:     groups,
			TotalDupes: totalDupes,
		})
	}

	return nil
}

// sourceKey is a composite key for grouping references by import source.
type sourceKey struct {
	Type string
	ID   string
}

// normalizeTitleAlnum normalizes a title for duplicate grouping: lowercase,
// keeping only Unicode letters and digits. This deliberately differs from
// normalizeTitleStrict, which preserves word boundaries for S2-lookup
// verification — here punctuation and spacing variants should collide.
func normalizeTitleAlnum(title string) string {
	var result []rune
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result = append(result, unicode.ToLower(r))
		}
	}
	return string(result)
}

// findDuplicateGroups finds references with the same source ID, then
// separately groups references sharing a normalized title. Source-ID groups
// come first, then title groups; both are sorted for stable output.
func findDuplicateGroups(refs []reference.Reference) []DuplicateGroup {
	sourceGroups := findSourceIDGroups(refs)
	titleGroups := findTitleGroups(refs)

	groups := make([]DuplicateGroup, 0, len(sourceGroups)+len(titleGroups))
	groups = append(groups, sourceGroups...)
	groups = append(groups, titleGroups...)
	return groups
}

// findSourceIDGroups finds references with the same import source ID.
func findSourceIDGroups(refs []reference.Reference) []DuplicateGroup {
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
			MatchBasis: "source_id",
			SourceType: key.Type,
			SourceID:   key.ID,
			Primary:    ids[0],  // Keep first occurrence
			Duplicates: ids[1:], // Remove rest
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].SourceType != groups[j].SourceType {
			return groups[i].SourceType < groups[j].SourceType
		}
		return groups[i].SourceID < groups[j].SourceID
	})

	return groups
}

// findTitleGroups finds references sharing a normalized title. Refs whose
// normalized title is empty, or whose raw title is the unknown-title
// sentinel, are skipped.
func findTitleGroups(refs []reference.Reference) []DuplicateGroup {
	titleMap := make(map[string][]string) // normalized title -> list of ref IDs

	for _, ref := range refs {
		if ref.Title == importer.UnknownTitle {
			continue
		}
		key := normalizeTitleAlnum(ref.Title)
		if key == "" {
			continue
		}
		titleMap[key] = append(titleMap[key], ref.ID)
	}

	var groups []DuplicateGroup
	for title, ids := range titleMap {
		if len(ids) < 2 {
			continue
		}
		groups = append(groups, DuplicateGroup{
			MatchBasis: "title",
			Members:    ids,
			Title:      title,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Title < groups[j].Title
	})

	return groups
}

// performMerge removes duplicate references.
func performMerge(repoRoot string, refs []reference.Reference, groups []DuplicateGroup) error {
	// Build the set of duplicate IDs to drop
	dupeSet := make(map[string]bool)
	for _, g := range groups {
		for _, dupeID := range g.Duplicates {
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
		return fmt.Errorf("writing refs: %w", err)
	}

	return nil
}
