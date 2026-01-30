package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/storage"
	"github.com/spf13/cobra"
)

var (
	searchLimit   int
	searchAuthors []string
	searchYear    string
	searchTitle   string
	searchVenue   string
	searchDOI     string
)

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", DefaultSearchLimit, "Maximum results to return")
	searchCmd.Flags().StringArrayVarP(&searchAuthors, "author", "a", nil, "Search by author name (can be repeated, uses AND logic)")
	searchCmd.Flags().StringVar(&searchYear, "year", "", "Filter by year: exact (2024), range (2020:2024), or open (2020: or :2024)")
	searchCmd.Flags().StringVarP(&searchTitle, "title", "t", "", "Search in title only")
	searchCmd.Flags().StringVar(&searchVenue, "venue", "", "Filter by venue/journal (partial match)")
	searchCmd.Flags().StringVar(&searchDOI, "doi", "", "Lookup by exact DOI")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search references by keyword, author, or year",
	Long: `Search references with flexible filtering options.

Query Syntax (positional argument):
  Plain text     - Searches title, abstract, and authors
  author:name    - Search author names only (legacy syntax)
  title:text     - Search title only

Flags:
  --author, -a   - Search by author (repeatable, AND logic, fuzzy prefix)
  --title, -t    - Search in title only
  --year         - Filter by year (exact, range, or open-ended)
  --venue        - Filter by venue/journal (partial match)
  --doi          - Lookup by exact DOI

Author matching supports fuzzy prefix matching, so "Tim" matches "Timothy".
When multiple authors are specified, all must match (AND logic).

Year syntax:
  --year 2024         - Exact year
  --year 2020:2024    - Range (inclusive)
  --year 2020:        - 2020 and later
  --year :2020        - 2020 and earlier

Examples:
  bip search "phylogenetics"
  bip search "deep mutational scanning" -a "Bloom" --year 2023:
  bip search -a "Yu" -a "Bloom" --year 2022:
  bip search --title "SARS-CoV-2" --venue Nature
  bip search --doi "10.1126/science.abf4063"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	var refs []reference.Reference
	var err error

	// Check if using flag-based search
	useFilters := len(searchAuthors) > 0 || searchYear != "" || searchTitle != "" || searchVenue != "" || searchDOI != ""

	if useFilters {
		filters := storage.SearchFilters{
			Authors: searchAuthors,
			Title:   searchTitle,
			Venue:   searchVenue,
			DOI:     searchDOI,
		}

		if len(args) > 0 {
			filters.Keyword = args[0]
		}

		if searchYear != "" {
			from, to, err := parseYearRange(searchYear)
			if err != nil {
				exitWithError(ExitError, "invalid year format: %v", err)
			}
			filters.YearFrom = from
			filters.YearTo = to
		}

		refs, err = db.SearchWithFilters(filters, searchLimit)
	} else if len(args) > 0 {
		// Legacy behavior: positional query argument
		query := args[0]

		// Check for field-specific searches (legacy syntax)
		if strings.HasPrefix(query, "author:") {
			value := strings.TrimPrefix(query, "author:")
			refs, err = db.SearchField("author", value, searchLimit)
		} else if strings.HasPrefix(query, "title:") {
			value := strings.TrimPrefix(query, "title:")
			refs, err = db.SearchField("title", value, searchLimit)
		} else {
			refs, err = db.Search(query, searchLimit)
		}
	} else {
		exitWithError(ExitError, "must specify a query or at least one filter (--author, --year)")
	}

	if err != nil {
		exitWithError(ExitError, "searching: %v", err)
	}

	// Empty result is not an error
	if refs == nil {
		refs = []reference.Reference{}
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

// parseYearRange parses a year specification into from/to values.
// Supported formats: "2024", "2020:2024", "2020:", ":2024"
func parseYearRange(spec string) (from, to int, err error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, 0, nil
	}

	// Check for range syntax
	if strings.Contains(spec, ":") {
		parts := strings.SplitN(spec, ":", 2)

		if parts[0] != "" {
			from, err = strconv.Atoi(parts[0])
			if err != nil {
				return 0, 0, fmt.Errorf("invalid start year %q", parts[0])
			}
		}

		if parts[1] != "" {
			to, err = strconv.Atoi(parts[1])
			if err != nil {
				return 0, 0, fmt.Errorf("invalid end year %q", parts[1])
			}
		}

		return from, to, nil
	}

	// Single year - exact match
	year, err := strconv.Atoi(spec)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year %q", spec)
	}

	return year, year, nil
}

func printRefSummary(num int, ref reference.Reference) {
	fmt.Printf("[%d] %s\n", num, ref.ID)
	fmt.Printf("    %s\n", truncateString(ref.Title, SearchTitleMaxLen))

	// Format authors (max 3, then "et al.")
	if len(ref.Authors) > 0 {
		fmt.Printf("    %s\n", formatAuthorsShort(ref.Authors, 3))
	}

	// Format venue and year
	if ref.Venue != "" {
		fmt.Printf("    %s (%d)\n", ref.Venue, ref.Published.Year)
	} else {
		fmt.Printf("    (%d)\n", ref.Published.Year)
	}
	fmt.Println()
}
