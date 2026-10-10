package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
	"github.com/matsen/bipartite/internal/s2"
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
	searchTag     string
	searchSort    string
)

// SearchResult is the --json shape for `bip search`. Total may exceed
// len(References) when the result was truncated by --limit.
type SearchResult struct {
	References []reference.Reference `json:"references"`
	Total      int                   `json:"total"`
}

// hasAnyFilterFlags returns true if any field-specific search flags were provided.
// This determines whether to use the new flag-based search or legacy positional query.
func hasAnyFilterFlags() bool {
	return len(searchAuthors) > 0 ||
		searchYear != "" ||
		searchTitle != "" ||
		searchVenue != "" ||
		searchDOI != "" ||
		searchTag != ""
}

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", DefaultSearchLimit, "Maximum results to return (0 or -1 for unlimited)")
	searchCmd.Flags().StringArrayVarP(&searchAuthors, "author", "a", nil, "Search by author name (can be repeated, uses AND logic)")
	searchCmd.Flags().StringVar(&searchYear, "year", "", "Filter by year: exact (2024), range (2020:2024), or open (2020: or :2024)")
	searchCmd.Flags().StringVarP(&searchTitle, "title", "t", "", "Search in title only")
	searchCmd.Flags().StringVar(&searchVenue, "venue", "", "Filter by venue/journal (partial match)")
	searchCmd.Flags().StringVar(&searchDOI, "doi", "", "Lookup by exact DOI")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Filter by tag/label (partial match)")
	searchCmd.Flags().StringVar(&searchSort, "sort", "relevance", "Result order: relevance or date (newest first)")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search references by keyword, author, or year",
	Long: `Search references with flexible filtering options.

Query Syntax (positional argument):
  Plain text     - Searches title, abstract, authors, notes, and tags
  author:name    - Search author names only (legacy syntax)
  title:text     - Search title only

Plain-text matching is exact-token (AND logic) and unstemmed: "germinal
center" will not match a title containing only "germinal centers". Try
--author or --doi for a definitive check before concluding a paper is
absent from the library.

Results are ranked by relevance (best match first), not alphabetically.
Use --sort date for newest first; it sorts every match before --limit applies.

Flags:
  --author, -a   - Search by author (repeatable, AND logic, exact last name)
  --title, -t    - Search in title only
  --year         - Filter by year (exact, range, or open-ended)
  --venue        - Filter by venue/journal (partial match)
  --doi          - Lookup by exact DOI
  --tag          - Filter by tag/label (partial match)
  --limit        - Maximum results to return (default 50; 0 or -1 for unlimited)
  --sort         - relevance (default) or date (newest first)

Author matching uses exact last name matching to avoid false positives:
  -a "Yu"           - Matches last name "Yu" exactly (not "Yujia")
  -a "Timothy Yu"   - Last name "Yu" + first name starts with "Timothy"
  -a "Yu, Timothy"  - Same as above (Last, First format)

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
  bip search -a "Bloom" --sort date --limit 2
  bip search --title "SARS-CoV-2" --venue Nature
  bip search --doi "10.1126/science.abf4063"
  bip search --tag "antibody"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func runSearch(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()
	db := mustOpenDatabase(repoRoot)
	defer db.Close()

	if searchSort != "relevance" && searchSort != "date" {
		exitWithError(ExitError, "invalid --sort %q: use relevance or date", searchSort)
	}
	// Sorting by date must see every match, so the limit applies afterwards.
	fetchLimit := searchLimit
	if searchSort == "date" {
		fetchLimit = 0
	}

	var refs []reference.Reference
	var total int
	var err error

	// Check if using flag-based search
	hasFilterFlags := hasAnyFilterFlags()

	if hasFilterFlags {
		filters := storage.SearchFilters{
			Authors: searchAuthors,
			Title:   searchTitle,
			Venue:   searchVenue,
			DOI:     s2.NormalizeDOI(searchDOI),
			Tag:     searchTag,
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

		refs, total, err = db.SearchWithFilters(filters, fetchLimit)
	} else if len(args) > 0 {
		// Legacy behavior: positional query argument
		query := args[0]

		// Check for field-specific searches (legacy syntax)
		if strings.HasPrefix(query, "author:") {
			value := strings.TrimPrefix(query, "author:")
			refs, total, err = db.SearchField("author", value, fetchLimit)
		} else if strings.HasPrefix(query, "title:") {
			value := strings.TrimPrefix(query, "title:")
			refs, total, err = db.SearchField("title", value, fetchLimit)
		} else {
			refs, total, err = db.Search(query, fetchLimit)
		}
	} else {
		exitWithError(ExitError, "must specify a query or at least one filter (--author, --year)")
	}

	if err != nil {
		exitWithError(ExitError, "searching: %v", err)
	}

	if searchSort == "date" {
		sortByDateDesc(refs)
		if searchLimit > 0 && len(refs) > searchLimit {
			refs = refs[:searchLimit]
		}
	}

	// Empty result is not an error
	if refs == nil {
		refs = []reference.Reference{}
	}

	if humanOutput {
		if len(refs) == 0 {
			fmt.Println("No references found")
		} else {
			if len(refs) < total {
				fmt.Printf("Found %d references (showing %d; use --limit to see more):\n\n", total, len(refs))
			} else {
				fmt.Printf("Found %d references:\n\n", len(refs))
			}
			for i, ref := range refs {
				printRefSummary(i+1, ref)
			}
		}
	} else {
		outputJSON(SearchResult{References: refs, Total: total})
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

// sortByDateDesc orders refs newest first by publication year, month, and
// day; an unknown month or day (0) sorts after known ones in the same year.
func sortByDateDesc(refs []reference.Reference) {
	sort.SliceStable(refs, func(i, j int) bool {
		a, b := refs[i].Published, refs[j].Published
		if a.Year != b.Year {
			return a.Year > b.Year
		}
		if a.Month != b.Month {
			return a.Month > b.Month
		}
		return a.Day > b.Day
	})
}
