package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// openStoreDB opens a SQLite database for a store.
func openStoreDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite doesn't support concurrent writes
	db.SetMaxOpenConns(1)

	return db, nil
}

// GenerateDDL generates a CREATE TABLE statement from a schema.
func GenerateDDL(schema *Schema) string {
	var cols []string

	for name, field := range schema.Fields {
		col := fmt.Sprintf("%s %s", name, sqliteType(field.Type))
		if field.Primary {
			col += " PRIMARY KEY"
		}
		cols = append(cols, col)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		schema.Name,
		strings.Join(cols, ",\n  "))
}

// GenerateIndexDDL generates a CREATE INDEX statement for a field.
func GenerateIndexDDL(tableName, fieldName string) string {
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s)",
		tableName, fieldName, tableName, fieldName)
}

// GenerateFTS5DDL generates a CREATE VIRTUAL TABLE statement for FTS5.
// Returns empty string if no FTS fields are defined.
func GenerateFTS5DDL(schema *Schema) string {
	pkField := schema.PrimaryKeyField()

	var ftsFields []string
	ftsFields = append(ftsFields, pkField) // Always include primary key

	for name, field := range schema.Fields {
		if field.FTS {
			ftsFields = append(ftsFields, name)
		}
	}

	// If only primary key, no FTS needed
	if len(ftsFields) == 1 {
		return ""
	}

	return fmt.Sprintf("CREATE VIRTUAL TABLE IF NOT EXISTS %s_fts USING fts5(\n  %s\n)",
		schema.Name,
		strings.Join(ftsFields, ",\n  "))
}

// GenerateMetaTableDDL generates the _meta table DDL.
func GenerateMetaTableDDL() string {
	return `CREATE TABLE IF NOT EXISTS _meta (
  key TEXT PRIMARY KEY,
  value TEXT
)`
}

// sqliteType maps FieldType to SQLite type.
func sqliteType(ft FieldType) string {
	switch ft {
	case FieldTypeString, FieldTypeDate, FieldTypeDatetime, FieldTypeJSON:
		return "TEXT"
	case FieldTypeInteger, FieldTypeBoolean:
		return "INTEGER"
	case FieldTypeFloat:
		return "REAL"
	default:
		return "TEXT"
	}
}

// GetStoredHash retrieves the JSONL hash from the _meta table.
func GetStoredHash(db *sql.DB) (string, error) {
	var hash sql.NullString
	err := db.QueryRow("SELECT value FROM _meta WHERE key = 'jsonl_hash'").Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return hash.String, nil
}

// SetStoredHash stores the JSONL hash in the _meta table.
func SetStoredHash(db *sql.DB, hash string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO _meta (key, value) VALUES ('jsonl_hash', ?)`, hash)
	return err
}

// GetLastSyncTime retrieves the last sync time from the _meta table.
func GetLastSyncTime(db *sql.DB) (time.Time, error) {
	var timeStr sql.NullString
	err := db.QueryRow("SELECT value FROM _meta WHERE key = 'last_sync'").Scan(&timeStr)
	if err == sql.ErrNoRows || !timeStr.Valid {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, timeStr.String)
}

// SetLastSyncTime stores the last sync time in the _meta table.
func SetLastSyncTime(db *sql.DB, t time.Time) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO _meta (key, value) VALUES ('last_sync', ?)`,
		t.Format(time.RFC3339))
	return err
}

// PrepareFTSQuery escapes special characters for FTS5 queries.
func PrepareFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return query
	}

	// If query contains special chars, quote it
	if strings.ContainsAny(query, "\"*+-:(){}[]^~") {
		// Escape internal quotes and wrap in quotes
		query = strings.ReplaceAll(query, "\"", "\"\"")
		return "\"" + query + "\""
	}

	return query
}

// AttachAllStores attaches all store databases to the given connection.
// Returns a cleanup function that detaches all stores.
func AttachAllStores(db *sql.DB, repoRoot string) (func(), error) {
	registry, err := LoadRegistry(repoRoot)
	if err != nil {
		return nil, err
	}

	var attached []string
	for name := range registry.Stores {
		store, err := OpenStore(repoRoot, name)
		if err != nil {
			continue // Skip stores that can't be opened
		}

		alias := name
		_, err = db.Exec(fmt.Sprintf("ATTACH DATABASE '%s' AS %s", store.DBPath(), alias))
		if err != nil {
			continue
		}
		attached = append(attached, alias)
	}

	cleanup := func() {
		for _, alias := range attached {
			db.Exec(fmt.Sprintf("DETACH DATABASE %s", alias))
		}
	}

	return cleanup, nil
}

// QueryCross executes a SQL query across multiple stores.
func QueryCross(repoRoot, sql string) ([]Record, error) {
	// Create a temporary in-memory database
	db, err := openStoreDB(":memory:")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Attach all stores
	cleanup, err := AttachAllStores(db, repoRoot)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Execute query
	rows, err := db.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	return scanRecords(rows)
}
