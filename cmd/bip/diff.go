package main

import (
	"fmt"
	"sort"

	"github.com/matsen/bipartite/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show papers added or removed since last commit",
	Long: `Show papers added or removed in the working tree compared to the last commit.

Useful for reviewing uncommitted changes before committing.

Examples:
  bip diff
  bip diff --human`,
	RunE: runDiff,
}

func runDiff(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	gitRoot := mustFindGitRepo(repoRoot)
	mustCheckGitTracking(gitRoot)

	// Get diff between HEAD and working tree
	diff, err := git.DiffWorkingTree(gitRoot)
	if err != nil {
		exitWithError(ExitError, "getting diff: %v", err)
	}

	// Sort for deterministic output
	git.SortRefsAlphabetically(diff.Added)
	git.SortRefsAlphabetically(diff.Removed)

	// Convert to output format
	added := make([]DiffPaper, 0, len(diff.Added))
	for _, ref := range diff.Added {
		added = append(added, refToDiffPaper(ref))
	}

	removed := make([]DiffPaper, 0, len(diff.Removed))
	for _, ref := range diff.Removed {
		removed = append(removed, refToDiffPaper(ref))
	}

	// Sort output by ID for consistency
	sort.Slice(added, func(i, j int) bool { return added[i].ID < added[j].ID })
	sort.Slice(removed, func(i, j int) bool { return removed[i].ID < removed[j].ID })

	result := DiffResult{
		Added:   added,
		Removed: removed,
	}

	if humanOutput {
		printDiffHuman(result)
	} else {
		outputJSON(result)
	}

	return nil
}

func printDiffHuman(result DiffResult) {
	if len(result.Added) == 0 && len(result.Removed) == 0 {
		fmt.Println("No changes since last commit.")
		return
	}

	fmt.Println("Changes since last commit:")
	fmt.Println()

	if len(result.Added) > 0 {
		fmt.Printf("Added (%d):\n", len(result.Added))
		for _, p := range result.Added {
			title := truncateString(p.Title, 50)
			fmt.Printf("  + %s: %s (%s, %d)\n", p.ID, title, p.Authors, p.Year)
		}
		fmt.Println()
	}

	if len(result.Removed) > 0 {
		fmt.Printf("Removed (%d):\n", len(result.Removed))
		for _, p := range result.Removed {
			title := truncateString(p.Title, 50)
			fmt.Printf("  - %s: %s (%s, %d)\n", p.ID, title, p.Authors, p.Year)
		}
	}
}
