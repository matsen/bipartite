package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store represents a registered data store.
type Store struct {
	Name       string
	Schema     *Schema
	Dir        string // Directory containing JSONL and DB files
	SchemaPath string // Path to the schema file
	jsonlPath  string // Derived: Dir/<name>.jsonl
	dbPath     string // Derived: Dir/<name>.db
	db         *sql.DB
}

// StoreInfo contains detailed information about a store.
type StoreInfo struct {
	Name       string    `json:"name"`
	JSONLPath  string    `json:"jsonl_path"`
	DBPath     string    `json:"db_path"`
	SchemaPath string    `json:"schema_path"`
	Records    int       `json:"records"`
	JSONLSize  int64     `json:"jsonl_size"`
	DBSize     int64     `json:"db_size"`
	LastSync   time.Time `json:"last_sync,omitempty"`
	InSync     bool      `json:"in_sync"`
	Error      string    `json:"error,omitempty"`
	Schema     *Schema   `json:"schema,omitempty"`
}

// NewStore creates a new Store instance with derived paths.
func NewStore(name string, schema *Schema, dir string, schemaPath string) *Store {
	return &Store{
		Name:       name,
		Schema:     schema,
		Dir:        dir,
		SchemaPath: schemaPath,
		jsonlPath:  filepath.Join(dir, name+".jsonl"),
		dbPath:     filepath.Join(dir, name+".db"),
	}
}

// OpenStore opens an existing store by name.
func OpenStore(repoRoot, name string) (*Store, error) {
	registry, err := LoadRegistry(repoRoot)
	if err != nil {
		return nil, err
	}

	config, ok := registry.Stores[name]
	if !ok {
		return nil, fmt.Errorf("store %q not found", name)
	}

	// Load schema
	schemaPath := filepath.Join(repoRoot, config.SchemaPath)
	schema, err := ParseSchema(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	// Determine store directory
	var dir string
	if config.Dir != "" {
		if filepath.IsAbs(config.Dir) {
			dir = config.Dir
		} else {
			dir = filepath.Join(repoRoot, config.Dir)
		}
	} else {
		// Default to .bipartite/
		dir = filepath.Join(repoRoot, ".bipartite")
	}

	return NewStore(name, schema, dir, config.SchemaPath), nil
}

// Init initializes a new store, creating empty JSONL and SQLite files.
func (s *Store) Init(repoRoot string) error {
	// Create store directory if needed
	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return fmt.Errorf("creating store directory: %w", err)
	}

	// Create empty JSONL file
	f, err := os.Create(s.jsonlPath)
	if err != nil {
		return fmt.Errorf("creating JSONL file: %w", err)
	}
	f.Close()

	// Create SQLite database with schema
	db, err := openStoreDB(s.dbPath)
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}
	defer db.Close()

	// Generate and execute DDL
	if err := s.createTables(db); err != nil {
		return fmt.Errorf("creating tables: %w", err)
	}

	// Register store
	registry, err := LoadRegistry(repoRoot)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	// Make schema path relative to repo root
	relSchemaPath, err := filepath.Rel(repoRoot, s.SchemaPath)
	if err != nil {
		relSchemaPath = s.SchemaPath
	}

	// Make directory relative to repo root (omit if default .bipartite/)
	var relDir string
	defaultDir := filepath.Join(repoRoot, ".bipartite")
	if s.Dir != defaultDir {
		relDir, err = filepath.Rel(repoRoot, s.Dir)
		if err != nil {
			relDir = s.Dir
		}
	}

	registry.Stores[s.Name] = &StoreConfig{
		SchemaPath: relSchemaPath,
		Dir:        relDir,
	}

	if err := SaveRegistry(repoRoot, registry); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	return nil
}

// createTables creates the SQLite tables for this store.
func (s *Store) createTables(db *sql.DB) error {
	// Main table
	ddl := GenerateDDL(s.Schema)
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("creating main table: %w", err)
	}

	// Indexes
	for name, field := range s.Schema.Fields {
		if field.Index && !field.Primary {
			indexDDL := GenerateIndexDDL(s.Schema.Name, name)
			if _, err := db.Exec(indexDDL); err != nil {
				return fmt.Errorf("creating index for %s: %w", name, err)
			}
		}
	}

	// FTS table (if any FTS fields)
	ftsDDL := GenerateFTS5DDL(s.Schema)
	if ftsDDL != "" {
		if _, err := db.Exec(ftsDDL); err != nil {
			return fmt.Errorf("creating FTS table: %w", err)
		}
	}

	// Meta table
	metaDDL := GenerateMetaTableDDL()
	if _, err := db.Exec(metaDDL); err != nil {
		return fmt.Errorf("creating meta table: %w", err)
	}

	return nil
}

// JSONLPath returns the path to the JSONL file.
func (s *Store) JSONLPath() string {
	return s.jsonlPath
}

// DBPath returns the path to the SQLite database.
func (s *Store) DBPath() string {
	return s.dbPath
}

// Count returns the number of records in the JSONL file.
func (s *Store) Count() (int, error) {
	records, err := ReadAllRecords(s.jsonlPath)
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// Info returns detailed information about the store.
func (s *Store) Info() (*StoreInfo, error) {
	info := &StoreInfo{
		Name:       s.Name,
		JSONLPath:  s.jsonlPath,
		DBPath:     s.dbPath,
		SchemaPath: s.SchemaPath,
		Schema:     s.Schema,
	}

	// Get record count
	count, err := s.Count()
	if err != nil {
		return nil, fmt.Errorf("counting records: %w", err)
	}
	info.Records = count

	// Get file sizes
	if stat, err := os.Stat(s.jsonlPath); err == nil {
		info.JSONLSize = stat.Size()
	}
	if stat, err := os.Stat(s.dbPath); err == nil {
		info.DBSize = stat.Size()
	}

	// Check sync status
	needsSync, err := s.NeedsSync()
	if err == nil {
		info.InSync = !needsSync
	}

	// Get last sync time
	lastSync, err := s.getLastSyncTime()
	if err == nil && !lastSync.IsZero() {
		info.LastSync = lastSync
	}

	return info, nil
}

// NeedsSync returns true if the SQLite database needs to be rebuilt.
func (s *Store) NeedsSync() (bool, error) {
	// Compute current JSONL hash
	currentHash, err := ComputeJSONLHash(s.jsonlPath)
	if err != nil {
		return true, err
	}

	// Get stored hash
	db, err := openStoreDB(s.dbPath)
	if err != nil {
		return true, err
	}
	defer db.Close()

	storedHash, err := GetStoredHash(db)
	if err != nil {
		return true, err
	}

	return currentHash != storedHash, nil
}

// Sync rebuilds the SQLite database from the JSONL source.
func (s *Store) Sync() (int, error) {
	// Read all records
	records, err := ReadAllRecords(s.jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading records: %w", err)
	}

	// Compute hash
	hash, err := ComputeJSONLHash(s.jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("computing hash: %w", err)
	}

	// Open database
	db, err := openStoreDB(s.dbPath)
	if err != nil {
		return 0, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Rebuild tables
	if err := s.rebuildTables(db, records); err != nil {
		return 0, fmt.Errorf("rebuilding tables: %w", err)
	}

	// Update metadata
	if err := SetStoredHash(db, hash); err != nil {
		return 0, fmt.Errorf("updating hash: %w", err)
	}

	if err := SetLastSyncTime(db, time.Now()); err != nil {
		return 0, fmt.Errorf("updating sync time: %w", err)
	}

	return len(records), nil
}

// rebuildTables clears and rebuilds all tables from records.
func (s *Store) rebuildTables(db *sql.DB, records []Record) error {
	// Clear main table
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", s.Schema.Name)); err != nil {
		return fmt.Errorf("clearing main table: %w", err)
	}

	// Clear FTS table if it exists
	ftsTable := s.Schema.Name + "_fts"
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", ftsTable)); err != nil {
		// FTS table might not exist, ignore error
	}

	// Insert records
	for i, record := range records {
		if err := s.insertRecord(db, record); err != nil {
			return fmt.Errorf("inserting record %d: %w", i+1, err)
		}
	}

	return nil
}

// insertRecord inserts a single record into the database.
func (s *Store) insertRecord(db *sql.DB, record Record) error {
	// Build column list and values
	var cols []string
	var placeholders []string
	var values []any

	for name, field := range s.Schema.Fields {
		cols = append(cols, name)
		placeholders = append(placeholders, "?")
		values = append(values, convertValueForSQLite(record[name], field.Type))
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		s.Schema.Name,
		joinStrings(cols, ", "),
		joinStrings(placeholders, ", "))

	if _, err := db.Exec(sql, values...); err != nil {
		return err
	}

	// Insert into FTS table if applicable
	if err := s.insertFTSRecord(db, record); err != nil {
		return err
	}

	return nil
}

// insertFTSRecord inserts a record into the FTS table.
func (s *Store) insertFTSRecord(db *sql.DB, record Record) error {
	// Find FTS fields
	var ftsFields []string
	pkField := s.Schema.PrimaryKeyField()

	for name, field := range s.Schema.Fields {
		if field.FTS {
			ftsFields = append(ftsFields, name)
		}
	}

	if len(ftsFields) == 0 {
		return nil // No FTS table
	}

	// Build INSERT for FTS
	cols := []string{pkField}
	cols = append(cols, ftsFields...)

	var placeholders []string
	var values []any

	for _, col := range cols {
		placeholders = append(placeholders, "?")
		if v, ok := record[col]; ok {
			values = append(values, v)
		} else {
			values = append(values, nil)
		}
	}

	sql := fmt.Sprintf("INSERT INTO %s_fts (%s) VALUES (%s)",
		s.Schema.Name,
		joinStrings(cols, ", "),
		joinStrings(placeholders, ", "))

	_, err := db.Exec(sql, values...)
	return err
}

// Append adds a record to the store.
func (s *Store) Append(record Record) error {
	// Validate record against schema
	if err := s.Schema.ValidateRecord(record); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Check for duplicate primary key
	pkField := s.Schema.PrimaryKeyField()
	pkValue := record[pkField]

	isDupe, err := CheckDuplicatePrimaryKey(s.jsonlPath, pkField, pkValue)
	if err != nil {
		return fmt.Errorf("checking duplicates: %w", err)
	}
	if isDupe {
		return fmt.Errorf("duplicate primary key: %q already exists", pkValue)
	}

	// Append to JSONL
	if err := AppendRecord(s.jsonlPath, record); err != nil {
		return fmt.Errorf("appending record: %w", err)
	}

	return nil
}

// Query executes a SQL query against the store's database.
func (s *Store) Query(sql string) ([]Record, error) {
	db, err := openStoreDB(s.dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	return scanRecords(rows)
}

// scanRecords converts SQL rows to records.
func scanRecords(rows *sql.Rows) ([]Record, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var records []Record
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		record := make(Record)
		for i, col := range cols {
			record[col] = values[i]
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// DeleteByID deletes a record by its primary key.
func (s *Store) DeleteByID(id any) error {
	pkField := s.Schema.PrimaryKeyField()

	// Read all records
	records, err := ReadAllRecords(s.jsonlPath)
	if err != nil {
		return fmt.Errorf("reading records: %w", err)
	}

	// Find and remove the record
	found := false
	var newRecords []Record
	for _, record := range records {
		if fmt.Sprintf("%v", record[pkField]) == fmt.Sprintf("%v", id) {
			found = true
			continue
		}
		newRecords = append(newRecords, record)
	}

	if !found {
		return fmt.Errorf("record %q not found", id)
	}

	// Write back
	if err := WriteAllRecords(s.jsonlPath, newRecords); err != nil {
		return fmt.Errorf("writing records: %w", err)
	}

	return nil
}

// DeleteWhere deletes records matching a SQL WHERE clause.
// Returns the number of records deleted.
func (s *Store) DeleteWhere(whereClause string) (int, error) {
	// First, find matching IDs using SQL
	db, err := openStoreDB(s.dbPath)
	if err != nil {
		return 0, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	pkField := s.Schema.PrimaryKeyField()
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s", pkField, s.Schema.Name, whereClause)

	rows, err := db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("finding matching records: %w", err)
	}
	defer rows.Close()

	// Collect IDs to delete
	var idsToDelete []any
	for rows.Next() {
		var id any
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		idsToDelete = append(idsToDelete, id)
	}

	if len(idsToDelete) == 0 {
		return 0, nil
	}

	// Read all records
	records, err := ReadAllRecords(s.jsonlPath)
	if err != nil {
		return 0, fmt.Errorf("reading records: %w", err)
	}

	// Build set of IDs to delete
	deleteSet := make(map[string]bool)
	for _, id := range idsToDelete {
		deleteSet[fmt.Sprintf("%v", id)] = true
	}

	// Filter out deleted records
	var newRecords []Record
	for _, record := range records {
		idStr := fmt.Sprintf("%v", record[pkField])
		if !deleteSet[idStr] {
			newRecords = append(newRecords, record)
		}
	}

	// Write back
	if err := WriteAllRecords(s.jsonlPath, newRecords); err != nil {
		return 0, fmt.Errorf("writing records: %w", err)
	}

	return len(idsToDelete), nil
}

// getLastSyncTime returns the last sync time from the database.
func (s *Store) getLastSyncTime() (time.Time, error) {
	db, err := openStoreDB(s.dbPath)
	if err != nil {
		return time.Time{}, err
	}
	defer db.Close()

	return GetLastSyncTime(db)
}

// convertValueForSQLite converts a Go value to a SQLite-compatible value.
func convertValueForSQLite(value any, fieldType FieldType) any {
	if value == nil {
		return nil
	}

	switch fieldType {
	case FieldTypeBoolean:
		if b, ok := value.(bool); ok {
			if b {
				return 1
			}
			return 0
		}
	case FieldTypeJSON:
		// Store as JSON string
		if _, ok := value.(string); ok {
			return value
		}
		// Marshal to JSON
		data, _ := json.Marshal(value)
		return string(data)
	}

	return value
}

// joinStrings joins strings with a separator (avoiding strings import).
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
