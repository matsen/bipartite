package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var (
	astaCitationsLimit int
	astaCitationsYear  string
)

var astaCitationsCmd = &cobra.Command{
	Use:   "citations <paper-id>",
	Short: "Get papers that cite this paper",
	Long: `Get papers that cite the given paper.

Examples:
  bp asta citations DOI:10.1093/sysbio/syy032
  bp asta citations DOI:10.1038/nature12373 --limit 20 --human
  bp asta citations DOI:10.1038/nature12373 --year 2020:`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaCitations,
}

func init() {
	astaCitationsCmd.Flags().IntVar(&astaCitationsLimit, "limit", 100, "Maximum number of results")
	astaCitationsCmd.Flags().StringVar(&astaCitationsYear, "year", "", "Filter citing papers by publication date (e.g., 2020:2024)")
	astaCmd.AddCommand(astaCitationsCmd)
}

func runAstaCitations(cmd *cobra.Command, args []string) {
	paperID := args[0]
	client := asta.NewClient()

	result, err := client.GetCitations(context.Background(), paperID, astaCitationsLimit, astaCitationsYear)
	if err != nil {
		os.Exit(astaOutputError(err, paperID))
	}

	if astaHuman {
		if len(result.Citations) == 0 {
			fmt.Printf("No citations found for %s\n", paperID)
			return
		}
		fmt.Printf("Found %d citations for %s\n\n", result.CitationCount, paperID)
		for i, p := range result.Citations {
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
