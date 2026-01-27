package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

var storeInitSchemaPath string
var storeInitDir string

// StoreInitResult is the response for store init command.
type StoreInitResult struct {
	Name       string `json:"name"`
	JSONLPath  string `json:"jsonl_path"`
	DBPath     string `json:"db_path"`
	SchemaPath string `json:"schema_path"`
}

func init() {
	storeCmd.AddCommand(storeInitCmd)
	storeInitCmd.Flags().StringVarP(&storeInitSchemaPath, "schema", "s", "", "Path to JSON schema file (required)")
	storeInitCmd.Flags().StringVarP(&storeInitDir, "dir", "d", "", "Directory for store files (default: .bipartite/)")
	storeInitCmd.MarkFlagRequired("schema")
}

var storeInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Initialize a new store",
	Long: `Initialize a new store with a JSON schema.

Creates an empty JSONL file, SQLite database with the schema, and registers
the store in .bipartite/stores.json.

Example:
  bip store init gh_activity --schema .bipartite/schemas/gh_activity.json`,
	Args: cobra.ExactArgs(1),
	RunE: runStoreInit,
}

// validStoreName matches valid store names (alphanumeric + underscore, must start with letter or underscore).
var validStoreName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func runStoreInit(cmd *cobra.Command, args []string) error {
	storeName := args[0]

	// Validate store name
	if !validStoreName.MatchString(storeName) {
		exitWithError(ExitError, "invalid store name %q: must be alphanumeric with underscores, starting with letter or underscore", storeName)
	}

	// Find repository
	repoRoot := mustFindRepository()

	// Check if store already exists
	registry, err := store.LoadRegistry(repoRoot)
	if err != nil {
		exitWithError(ExitError, "loading registry: %v", err)
	}

	if _, exists := registry.Stores[storeName]; exists {
		exitWithError(ExitError, "store %q already exists", storeName)
	}

	// Parse and validate schema
	schemaPath := storeInitSchemaPath
	if !filepath.IsAbs(schemaPath) {
		schemaPath = filepath.Join(repoRoot, schemaPath)
	}

	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		exitWithError(ExitError, "schema file not found: %s", storeInitSchemaPath)
	}

	schema, err := store.ParseSchema(schemaPath)
	if err != nil {
		exitWithError(ExitError, "invalid schema: %v", err)
	}

	if err := schema.Validate(); err != nil {
		exitWithError(ExitError, "invalid schema: %v", err)
	}

	// Override schema name to match store name
	schema.Name = storeName

	// Determine store directory
	dir := storeInitDir
	if dir == "" {
		dir = filepath.Join(repoRoot, ".bipartite")
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoRoot, dir)
	}

	// Create store
	s := store.NewStore(storeName, schema, dir, schemaPath)
	if err := s.Init(repoRoot); err != nil {
		exitWithError(ExitError, "initializing store: %v", err)
	}

	// Output result
	result := StoreInitResult{
		Name:       storeName,
		JSONLPath:  s.JSONLPath(),
		DBPath:     s.DBPath(),
		SchemaPath: storeInitSchemaPath,
	}

	if humanOutput {
		fmt.Printf("Created store '%s':\n", storeName)
		fmt.Printf("  JSONL:  %s\n", result.JSONLPath)
		fmt.Printf("  DB:     %s\n", result.DBPath)
		fmt.Printf("  Schema: %s\n", result.SchemaPath)
	} else {
		outputJSON(result)
	}

	return nil
}
