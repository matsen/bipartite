package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var astaReferencesLimit int

var astaReferencesCmd = &cobra.Command{
	Use:   "references <paper-id>",
	Short: "Get papers referenced by this paper",
	Long: `Get papers referenced by the given paper (its bibliography).

Examples:
  bp asta references DOI:10.1093/sysbio/syy032
  bp asta references ARXIV:2106.15928 --limit 50 --human`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaReferences,
}

func init() {
	astaReferencesCmd.Flags().IntVar(&astaReferencesLimit, "limit", asta.DefaultReferencesLimit, "Maximum number of results")
	astaCmd.AddCommand(astaReferencesCmd)
}

func runAstaReferences(cmd *cobra.Command, args []string) {
	paperID, err := validatePaperID(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.GetReferences(ctx, paperID, astaReferencesLimit)
		},
		formatReferencesHuman(paperID),
		paperID,
	)
}
