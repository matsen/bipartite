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
	astaReferencesCmd.Flags().IntVar(&astaReferencesLimit, "limit", 100, "Maximum number of results")
	astaCmd.AddCommand(astaReferencesCmd)
}

func runAstaReferences(cmd *cobra.Command, args []string) {
	paperID := args[0]
	client := asta.NewClient()

	result, err := client.GetReferences(context.Background(), paperID, astaReferencesLimit)
	if err != nil {
		os.Exit(astaOutputError(err, paperID))
	}

	if astaHuman {
		if len(result.References) == 0 {
			fmt.Printf("No references found for %s\n", paperID)
			return
		}
		fmt.Printf("Found %d references for %s\n\n", result.ReferenceCount, paperID)
		for i, p := range result.References {
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
