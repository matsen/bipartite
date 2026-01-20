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
	astaAuthorCmd.Flags().IntVar(&astaAuthorLimit, "limit", 10, "Maximum number of results")
	astaCmd.AddCommand(astaAuthorCmd)
}

func runAstaAuthor(cmd *cobra.Command, args []string) {
	name := args[0]
	client := asta.NewClient()

	result, err := client.SearchAuthors(context.Background(), name, astaAuthorLimit)
	if err != nil {
		os.Exit(astaOutputError(err, ""))
	}

	if astaHuman {
		if len(result.Authors) == 0 {
			fmt.Printf("No authors found for \"%s\"\n", name)
			return
		}
		fmt.Printf("Found %d authors matching \"%s\"\n\n", len(result.Authors), name)
		for i, a := range result.Authors {
			fmt.Print(formatAuthorHuman(a, i+1))
			fmt.Println()
		}
	} else {
		if err := astaOutputJSON(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(ExitError)
		}
	}
}
