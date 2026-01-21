package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var (
	astaSnippetLimit  int
	astaSnippetVenue  string
	astaSnippetPapers string
)

var astaSnippetCmd = &cobra.Command{
	Use:   "snippet <query>",
	Short: "Search text snippets within papers",
	Long: `Search for text snippets within papers using ASTA's unique snippet search.

This finds exact text passages matching your query, returning the snippet
with context about the paper it came from.

Examples:
  bip asta snippet "variational inference phylogenetics"
  bip asta snippet "mutation rate" --papers "DOI:10.1093/sysbio/syy032,ARXIV:2106.15928"
  bip asta snippet "deep learning" --venue "Nature" --human`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaSnippet,
}

func init() {
	astaSnippetCmd.Flags().IntVar(&astaSnippetLimit, "limit", asta.DefaultSnippetLimit, "Maximum number of snippets")
	astaSnippetCmd.Flags().StringVar(&astaSnippetVenue, "venue", "", "Filter by venue")
	astaSnippetCmd.Flags().StringVar(&astaSnippetPapers, "papers", "", "Comma-separated paper IDs to search within")
	astaCmd.AddCommand(astaSnippetCmd)
}

func runAstaSnippet(cmd *cobra.Command, args []string) {
	query, err := validateQuery(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.SnippetSearch(ctx, query, astaSnippetLimit, astaSnippetVenue, astaSnippetPapers)
		},
		formatSnippetResultsHuman,
		"",
	)
}
