package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var (
	astaAuthorPapersLimit int
	astaAuthorPapersYear  string
)

var astaAuthorPapersCmd = &cobra.Command{
	Use:   "author-papers <author-id>",
	Short: "Get papers by an author",
	Long: `Get papers by an author using their Semantic Scholar author ID.

Use "bp asta author <name>" to find an author's ID first.

Examples:
  bp asta author-papers 1234567
  bp asta author-papers 1234567 --limit 50 --human
  bp asta author-papers 1234567 --year 2020:2024`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaAuthorPapers,
}

func init() {
	astaAuthorPapersCmd.Flags().IntVar(&astaAuthorPapersLimit, "limit", 100, "Maximum number of results")
	astaAuthorPapersCmd.Flags().StringVar(&astaAuthorPapersYear, "year", "", "Filter by publication date (e.g., 2020:2024)")
	astaCmd.AddCommand(astaAuthorPapersCmd)
}

func runAstaAuthorPapers(cmd *cobra.Command, args []string) {
	authorID := args[0]
	client := asta.NewClient()

	result, err := client.GetAuthorPapers(context.Background(), authorID, astaAuthorPapersLimit, astaAuthorPapersYear)
	if err != nil {
		os.Exit(astaOutputError(err, ""))
	}

	if astaHuman {
		if len(result.Papers) == 0 {
			fmt.Printf("No papers found for author %s\n", authorID)
			return
		}
		fmt.Printf("Found %d papers by author %s\n\n", len(result.Papers), authorID)
		for i, p := range result.Papers {
			fmt.Print(formatPaperHuman(p, i+1))
			fmt.Println()
		}
	} else {
		if err := astaOutputJSON(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(ExitError)
		}
	}
}
