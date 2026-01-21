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

Use "bip asta author <name>" to find an author's ID first.

Examples:
  bip asta author-papers 1234567
  bip asta author-papers 1234567 --limit 50 --human
  bip asta author-papers 1234567 --year 2020:2024`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaAuthorPapers,
}

func init() {
	astaAuthorPapersCmd.Flags().IntVar(&astaAuthorPapersLimit, "limit", asta.DefaultAuthorPapersLimit, "Maximum number of results")
	astaAuthorPapersCmd.Flags().StringVar(&astaAuthorPapersYear, "year", "", "Filter by publication date (e.g., 2020:2024)")
	astaCmd.AddCommand(astaAuthorPapersCmd)
}

func runAstaAuthorPapers(cmd *cobra.Command, args []string) {
	authorID, err := validateAuthorID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.GetAuthorPapers(ctx, authorID, astaAuthorPapersLimit, astaAuthorPapersYear)
		},
		formatAuthorPapersHuman(authorID),
		"",
	)
}
