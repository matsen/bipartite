package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var (
	astaSearchLimit int
	astaSearchYear  string
	astaSearchVenue string
)

var astaSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search papers by keyword",
	Long: `Search for papers by keyword relevance using ASTA.

Examples:
  bip asta search "phylogenetic inference"
  bip asta search "SARS-CoV-2" --limit 10 --year 2020:2024
  bip asta search "machine learning" --venue "Nature" --human`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaSearch,
}

func init() {
	astaSearchCmd.Flags().IntVar(&astaSearchLimit, "limit", asta.DefaultSearchLimit, "Maximum number of results")
	astaSearchCmd.Flags().StringVar(&astaSearchYear, "year", "", "Publication date range (e.g., 2020:2024)")
	astaSearchCmd.Flags().StringVar(&astaSearchVenue, "venue", "", "Filter by venue")
	astaCmd.AddCommand(astaSearchCmd)
}

func runAstaSearch(cmd *cobra.Command, args []string) {
	query, err := validateQuery(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.SearchPapers(ctx, query, astaSearchLimit, astaSearchYear, astaSearchVenue)
		},
		formatSearchResultsHuman,
		"",
	)
}
