package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

var storeAppendFile string
var storeAppendStdin bool

// StoreAppendResult is the response for store append command.
type StoreAppendResult struct {
	Store    string `json:"store"`
	Appended int    `json:"appended"`
}

func init() {
	storeCmd.AddCommand(storeAppendCmd)
	storeAppendCmd.Flags().StringVarP(&storeAppendFile, "file", "f", "", "Path to JSON/JSONL file")
	storeAppendCmd.Flags().BoolVar(&storeAppendStdin, "stdin", false, "Read JSONL from stdin")
}

var storeAppendCmd = &cobra.Command{
	Use:   "append <name> [json]",
	Short: "Append records to a store",
	Long: `Append one or more records to a store's JSONL file.

Records are validated against the store's schema before appending.
Primary key uniqueness is enforced.

Examples:
  # Single record from argument
  bip store append gh_activity '{"id":"pr-123","type":"pr",...}'

  # From a file
  bip store append gh_activity --file records.json

  # From stdin (JSONL)
  cat records.jsonl | bip store append gh_activity --stdin`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStoreAppend,
}

func runStoreAppend(cmd *cobra.Command, args []string) error {
	storeName := args[0]
	repoRoot := mustFindRepository()

	// Open store
	s, err := store.OpenStore(repoRoot, storeName)
	if err != nil {
		exitWithError(ExitError, "store %q not found", storeName)
	}

	var records []store.Record

	// Determine input source
	if storeAppendStdin {
		// Read from stdin
		records, err = readRecordsFromStdin()
		if err != nil {
			exitWithError(ExitError, "reading from stdin: %v", err)
		}
	} else if storeAppendFile != "" {
		// Read from file
		records, err = readRecordsFromFile(storeAppendFile)
		if err != nil {
			exitWithError(ExitError, "reading file: %v", err)
		}
	} else if len(args) == 2 {
		// Parse inline JSON
		var record store.Record
		if err := json.Unmarshal([]byte(args[1]), &record); err != nil {
			exitWithError(ExitError, "invalid JSON: %v", err)
		}
		records = []store.Record{record}
	} else {
		exitWithError(ExitError, "no input provided: use inline JSON, --file, or --stdin")
	}

	// Append records
	appended := 0
	for _, record := range records {
		if err := s.Append(record); err != nil {
			exitWithError(ExitError, "%v", err)
		}
		appended++
	}

	result := StoreAppendResult{
		Store:    storeName,
		Appended: appended,
	}

	if humanOutput {
		if appended == 1 {
			fmt.Printf("Appended 1 record to '%s'\n", storeName)
		} else {
			fmt.Printf("Appended %d records to '%s'\n", appended, storeName)
		}
	} else {
		outputJSON(result)
	}

	return nil
}

// readRecordsFromStdin reads JSONL records from stdin.
func readRecordsFromStdin() ([]store.Record, error) {
	var records []store.Record
	scanner := bufio.NewScanner(os.Stdin)

	buf := make([]byte, store.MaxJSONLLineCapacity)
	scanner.Buffer(buf, store.MaxJSONLLineCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record store.Record
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		records = append(records, record)
	}

	return records, scanner.Err()
}

// readRecordsFromFile reads records from a JSON or JSONL file.
func readRecordsFromFile(path string) ([]store.Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try parsing as JSON array first
	var records []store.Record
	if err := json.Unmarshal(data, &records); err == nil {
		return records, nil
	}

	// Try parsing as single JSON object
	var record store.Record
	if err := json.Unmarshal(data, &record); err == nil {
		return []store.Record{record}, nil
	}

	// Fall back to JSONL
	return store.ReadAllRecords(path)
}
