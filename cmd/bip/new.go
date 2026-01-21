package main

import (
	"fmt"
	"sort"

	"github.com/matsen/bipartite/internal/git"
	"github.com/spf13/cobra"
)

var (
	newSince string
	newDays  int
)

func init() {
	newCmd.Flags().StringVar(&newSince, "since", "", "List papers added after this git commit")
	newCmd.Flags().IntVar(&newDays, "days", 0, "List papers added within last N days (UTC)")
	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "List papers added since a commit or within N days",
	Long: `List papers added to the repository since a specific commit or within the last N days.

Useful for tracking new additions from collaborators after pulling updates.

Examples:
  bip new --since abc123f
  bip new --since HEAD~3
  bip new --days 7
  bip new --days 7 --human`,
	RunE: runNew,
}

func runNew(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	gitRoot := mustFindGitRepo(repoRoot)

	// Validate mutual exclusivity
	hasSince := newSince != ""
	hasDays := newDays > 0

	if hasSince && hasDays {
		exitWithError(ExitError, "--since and --days are mutually exclusive")
	}

	if !hasSince && !hasDays {
		exitWithError(ExitError, "--since or --days flag required\n  Hint: Use 'bip new --since <commit>' or 'bip new --days N'")
	}

	var papers []NewPaper
	var sinceRef string

	if hasSince {
		// Validate commit
		sha := mustValidateCommit(gitRoot, newSince)
		sinceRef = sha[:8] // Short SHA

		// Get papers added since commit
		recentPapers, err := git.GetPapersAddedSince(gitRoot, newSince)
		if err != nil {
			exitWithError(ExitError, "getting papers since %s: %v", newSince, err)
		}

		for _, rp := range recentPapers {
			papers = append(papers, refToNewPaper(rp.Reference, rp.CommitSHA))
		}
	} else if hasDays {
		sinceRef = fmt.Sprintf("%d days ago", newDays)

		// Get papers added in last N days
		recentPapers, err := git.GetPapersAddedInDays(gitRoot, newDays)
		if err != nil {
			exitWithError(ExitError, "getting papers from last %d days: %v", newDays, err)
		}

		for _, rp := range recentPapers {
			papers = append(papers, refToNewPaper(rp.Reference, rp.CommitSHA))
		}
	}

	// Sort by ID for deterministic output
	sort.Slice(papers, func(i, j int) bool { return papers[i].ID < papers[j].ID })

	result := NewPapersResult{
		Papers:     papers,
		SinceRef:   sinceRef,
		TotalCount: len(papers),
	}

	if humanOutput {
		printNewPapersHuman(result)
	} else {
		outputJSON(result)
	}

	return nil
}

func printNewPapersHuman(result NewPapersResult) {
	if len(result.Papers) == 0 {
		fmt.Printf("No papers added since %s.\n", result.SinceRef)
		return
	}

	fmt.Printf("Papers added since %s:\n\n", result.SinceRef)

	for _, p := range result.Papers {
		fmt.Printf("  %s: %s\n", p.ID, truncateString(p.Title, 50))
		fmt.Printf("    %s (%d)\n", p.Authors, p.Year)
		if p.CommitSHA != "" {
			fmt.Printf("    Added in commit %s\n", p.CommitSHA)
		}
		fmt.Println()
	}

	fmt.Printf("%d paper(s) added\n", result.TotalCount)
}
