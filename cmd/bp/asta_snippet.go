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
  bp asta snippet "variational inference phylogenetics"
  bp asta snippet "mutation rate" --papers "DOI:10.1093/sysbio/syy032,ARXIV:2106.15928"
  bp asta snippet "deep learning" --venue "Nature" --human`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaSnippet,
}

func init() {
	astaSnippetCmd.Flags().IntVar(&astaSnippetLimit, "limit", 20, "Maximum number of snippets")
	astaSnippetCmd.Flags().StringVar(&astaSnippetVenue, "venue", "", "Filter by venue")
	astaSnippetCmd.Flags().StringVar(&astaSnippetPapers, "papers", "", "Comma-separated paper IDs to search within")
	astaCmd.AddCommand(astaSnippetCmd)
}

func runAstaSnippet(cmd *cobra.Command, args []string) {
	query := args[0]
	client := asta.NewClient()

	result, err := client.SnippetSearch(context.Background(), query, astaSnippetLimit, astaSnippetVenue, astaSnippetPapers)
	if err != nil {
		os.Exit(astaOutputError(err, ""))
	}

	if astaHuman {
		if len(result.Snippets) == 0 {
			fmt.Println("No snippets found")
			return
		}
		fmt.Printf("Found %d snippets\n\n", len(result.Snippets))
		for i, s := range result.Snippets {
			fmt.Print(formatSnippetHuman(s, i+1))
			fmt.Println()
		}
	} else {
		if err := astaOutputJSON(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(ExitError)
		}
	}
}
