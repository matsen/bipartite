package main

import (
	"fmt"
	"strings"

	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var searchLimit int

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", DefaultSearchLimit, "Maximum results to return")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search references by keyword",
	Long: `Search references by keyword.

Query Syntax:
  Plain text     - Searches title, abstract, and authors
  author:name    - Search author names only
  title:text     - Search title only

Examples:
  bp search "phylogenetics"
  bp search "author:Matsen"
  bp search "title:influenza"`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	query := args[0]
	var refs []storage.Reference
	var err error

	// Check for field-specific searches
	if strings.HasPrefix(query, "author:") {
		value := strings.TrimPrefix(query, "author:")
		refs, err = db.SearchField("author", value, searchLimit)
	} else if strings.HasPrefix(query, "title:") {
		value := strings.TrimPrefix(query, "title:")
		refs, err = db.SearchField("title", value, searchLimit)
	} else {
		refs, err = db.Search(query, searchLimit)
	}

	if err != nil {
		exitWithError(ExitError, "searching: %v", err)
	}

	// Empty result is not an error
	if refs == nil {
		refs = []storage.Reference{}
	}

	if humanOutput {
		if len(refs) == 0 {
			fmt.Println("No references found")
		} else {
			fmt.Printf("Found %d references:\n\n", len(refs))
			for i, ref := range refs {
				printRefSummary(i+1, ref)
			}
		}
	} else {
		outputJSON(refs)
	}

	return nil
}

func printRefSummary(num int, ref storage.Reference) {
	fmt.Printf("[%d] %s\n", num, ref.ID)
	fmt.Printf("    %s\n", truncateString(ref.Title, SummaryTitleLen))

	// Format authors
	if len(ref.Authors) > 0 {
		var authorNames []string
		for i, a := range ref.Authors {
			if i >= 3 {
				authorNames = append(authorNames, "et al.")
				break
			}
			if a.First != "" {
				authorNames = append(authorNames, a.Last+" "+string(a.First[0]))
			} else {
				authorNames = append(authorNames, a.Last)
			}
		}
		fmt.Printf("    %s\n", strings.Join(authorNames, ", "))
	}

	// Format venue and year
	if ref.Venue != "" {
		fmt.Printf("    %s (%d)\n", ref.Venue, ref.Published.Year)
	} else {
		fmt.Printf("    (%d)\n", ref.Published.Year)
	}
	fmt.Println()
}
