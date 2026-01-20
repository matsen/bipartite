package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/asta"
	"github.com/spf13/cobra"
)

var astaAuthorLimit int

var astaAuthorCmd = &cobra.Command{
	Use:   "author <name>",
	Short: "Search for authors by name",
	Long: `Search for authors by name and get their publication metrics.

Examples:
  bp asta author "Frederick Matsen"
  bp asta author "Smith" --limit 20 --human`,
	Args: cobra.ExactArgs(1),
	Run:  runAstaAuthor,
}

func init() {
	astaAuthorCmd.Flags().IntVar(&astaAuthorLimit, "limit", asta.DefaultAuthorSearchLimit, "Maximum number of results")
	astaCmd.AddCommand(astaAuthorCmd)
}

func runAstaAuthor(cmd *cobra.Command, args []string) {
	name, err := validateQuery(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	astaExecute(
		func(ctx context.Context, client *asta.Client) (any, error) {
			return client.SearchAuthors(ctx, name, astaAuthorLimit)
		},
		formatAuthorsHuman(name),
		"",
	)
}
