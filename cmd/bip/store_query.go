package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

var storeQueryJSON bool
var storeQueryCSV bool
var storeQueryJSONL bool
var storeQueryCross bool

func init() {
	storeCmd.AddCommand(storeQueryCmd)
	storeQueryCmd.Flags().BoolVar(&storeQueryJSON, "json", false, "Output JSON array")
	storeQueryCmd.Flags().BoolVar(&storeQueryCSV, "csv", false, "Output CSV")
	storeQueryCmd.Flags().BoolVar(&storeQueryJSONL, "jsonl", false, "Output JSONL")
	storeQueryCmd.Flags().BoolVarP(&storeQueryCross, "cross", "x", false, "Enable cross-store query")
}

var storeQueryCmd = &cobra.Command{
	Use:   "query <name> <sql>",
	Short: "Query a store using SQL",
	Long: `Execute a SQL query against a store's SQLite index.

For cross-store queries (JOINs across multiple stores), use --cross flag.
The store name can be omitted with --cross since tables are referenced in SQL.

Examples:
  # Basic query
  bip store query gh_activity "SELECT * FROM gh_activity WHERE type = 'pr'"

  # Full-text search
  bip store query gh_activity "SELECT * FROM gh_activity WHERE id IN (SELECT id FROM gh_activity_fts WHERE gh_activity_fts MATCH 'store')"

  # Cross-store query
  bip store query --cross "SELECT r.title, g.author FROM refs r JOIN gh_activity g ON r.id = g.ref_id"

  # Output formats
  bip store query gh_activity "SELECT id, title FROM gh_activity" --json
  bip store query gh_activity "SELECT id, title FROM gh_activity" --csv
  bip store query gh_activity "SELECT id, title FROM gh_activity" --jsonl`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStoreQuery,
}

func runStoreQuery(cmd *cobra.Command, args []string) error {
	repoRoot := mustFindRepository()

	var storeName, sql string

	if storeQueryCross {
		// Cross-store: SQL is the only/first argument
		if len(args) == 1 {
			sql = args[0]
		} else {
			// Allow "store query --cross storename sql" for consistency
			sql = args[1]
		}
	} else {
		if len(args) < 2 {
			exitWithError(ExitError, "usage: bip store query <name> <sql>")
		}
		storeName = args[0]
		sql = args[1]
	}

	var records []store.Record
	var err error

	if storeQueryCross {
		records, err = store.QueryCross(repoRoot, sql)
		if err != nil {
			exitWithError(ExitError, "SQL error: %v", err)
		}
	} else {
		// Open store
		s, err := store.OpenStore(repoRoot, storeName)
		if err != nil {
			exitWithError(ExitError, "store %q not found", storeName)
		}

		// Check if synced
		needsSync, err := s.NeedsSync()
		if err != nil {
			exitWithError(ExitError, "checking sync status: %v", err)
		}
		if needsSync {
			exitWithError(ExitError, "store %q not synced, run 'bip store sync %s' first", storeName, storeName)
		}

		// Execute query
		records, err = s.Query(sql)
		if err != nil {
			exitWithError(ExitError, "SQL error: %v", err)
		}
	}

	// Output results
	if storeQueryJSON || (!humanOutput && !storeQueryCSV && !storeQueryJSONL) {
		outputJSON(records)
	} else if storeQueryCSV {
		outputCSV(records)
	} else if storeQueryJSONL {
		outputJSONL(records)
	} else {
		outputTable(records)
	}

	return nil
}

// outputCSV writes records as CSV.
func outputCSV(records []store.Record) {
	if len(records) == 0 {
		return
	}

	// Get column names from first record
	var cols []string
	for col := range records[0] {
		cols = append(cols, col)
	}

	w := csv.NewWriter(os.Stdout)

	// Write header
	w.Write(cols)

	// Write data
	for _, record := range records {
		var row []string
		for _, col := range cols {
			row = append(row, fmt.Sprintf("%v", record[col]))
		}
		w.Write(row)
	}

	w.Flush()
}

// outputJSONL writes records as JSONL.
func outputJSONL(records []store.Record) {
	for _, record := range records {
		data, _ := json.Marshal(record)
		fmt.Println(string(data))
	}
}

// outputTable writes records as a formatted table.
func outputTable(records []store.Record) {
	if len(records) == 0 {
		fmt.Println("(0 rows)")
		return
	}

	// Get column names from first record
	var cols []string
	for col := range records[0] {
		cols = append(cols, col)
	}

	// Calculate column widths
	widths := make(map[string]int)
	for _, col := range cols {
		widths[col] = len(col)
	}
	for _, record := range records {
		for _, col := range cols {
			valStr := fmt.Sprintf("%v", record[col])
			if len(valStr) > widths[col] {
				widths[col] = len(valStr)
			}
		}
	}

	// Cap column widths at 40 characters
	for col := range widths {
		if widths[col] > 40 {
			widths[col] = 40
		}
	}

	// Print header
	var header []string
	for _, col := range cols {
		header = append(header, padRight(strings.ToUpper(col), widths[col]))
	}
	fmt.Println(strings.Join(header, "  "))

	// Print data
	for _, record := range records {
		var row []string
		for _, col := range cols {
			valStr := fmt.Sprintf("%v", record[col])
			if len(valStr) > widths[col] {
				valStr = valStr[:widths[col]-3] + "..."
			}
			row = append(row, padRight(valStr, widths[col]))
		}
		fmt.Println(strings.Join(row, "  "))
	}

	fmt.Printf("(%d rows)\n", len(records))
}

// padRight pads a string with spaces on the right.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
