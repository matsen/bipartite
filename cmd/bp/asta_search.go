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
  bp asta search "phylogenetic inference"
  bp asta search "SARS-CoV-2" --limit 10 --year 2020:2024
  bp asta search "machine learning" --venue "Nature" --human`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaSearch,
}

func init() {
	astaSearchCmd.Flags().IntVar(&astaSearchLimit, "limit", 50, "Maximum number of results")
	astaSearchCmd.Flags().StringVar(&astaSearchYear, "year", "", "Publication date range (e.g., 2020:2024)")
	astaSearchCmd.Flags().StringVar(&astaSearchVenue, "venue", "", "Filter by venue")
	astaCmd.AddCommand(astaSearchCmd)
}

func runAstaSearch(cmd *cobra.Command, args []string) {
	query := args[0]
	client := asta.NewClient()

	result, err := client.SearchPapers(context.Background(), query, astaSearchLimit, astaSearchYear, astaSearchVenue)
	if err != nil {
		os.Exit(astaOutputError(err, ""))
	}

	if astaHuman {
		if result.Total == 0 {
			fmt.Println("No papers found")
			return
		}
		fmt.Printf("Found %d papers\n\n", result.Total)
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
