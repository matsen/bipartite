package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var astaPaperFields string

var astaPaperCmd = &cobra.Command{
	Use:   "paper <paper-id>",
	Short: "Get paper details by ID",
	Long: `Get detailed information about a paper by its identifier.

Supported ID formats:
  DOI:10.1093/sysbio/syy032
  ARXIV:2106.15928
  PMID:19872477
  PMCID:2323736
  CorpusId:215416146
  <S2 paper ID>

Examples:
  bip asta paper DOI:10.1093/sysbio/syy032
  bip asta paper ARXIV:2106.15928 --human
  bip asta paper DOI:10.1038/nature12373 --fields title,authors,citationCount`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaPaper,
}

func init() {
	astaPaperCmd.Flags().StringVar(&astaPaperFields, "fields", "", "Comma-separated fields to return")
	astaCmd.AddCommand(astaPaperCmd)
}

func runAstaPaper(cmd *cobra.Command, args []string) {
	paperID, err := validatePaperID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.GetPaper(ctx, paperID, astaPaperFields)
		},
		formatPaperDetailsHuman,
		paperID,
	)
}
