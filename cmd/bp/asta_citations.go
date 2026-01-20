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
	astaCitationsCmd.Flags().IntVar(&astaCitationsLimit, "limit", asta.DefaultCitationsLimit, "Maximum number of results")
	astaCitationsCmd.Flags().StringVar(&astaCitationsYear, "year", "", "Filter citing papers by publication date (e.g., 2020:2024)")
	astaCmd.AddCommand(astaCitationsCmd)
}

func runAstaCitations(cmd *cobra.Command, args []string) {
	paperID, err := validatePaperID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.GetCitations(ctx, paperID, astaCitationsLimit, astaCitationsYear)
		},
		formatCitationsHuman(paperID),
		paperID,
	)
}
