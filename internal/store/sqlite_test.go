package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateDDL(t *testing.T) {
	schema := &Schema{
		Name: "test_table",
		Fields: map[string]*Field{
			"id":       {Type: FieldTypeString, Primary: true},
			"name":     {Type: FieldTypeString},
			"count":    {Type: FieldTypeInteger},
			"score":    {Type: FieldTypeFloat},
			"active":   {Type: FieldTypeBoolean},
			"date":     {Type: FieldTypeDate},
			"updated":  {Type: FieldTypeDatetime},
			"metadata": {Type: FieldTypeJSON},
		},
	}

	ddl := GenerateDDL(schema)

	// Check table name
	if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS test_table") {
		t.Errorf("DDL should contain table name: %s", ddl)
	}

	// Check column types
	expectations := []string{
		"id TEXT PRIMARY KEY",
		"name TEXT",
		"count INTEGER",
		"score REAL",
		"active INTEGER", // boolean stored as INTEGER
		"date TEXT",      // date stored as TEXT
		"updated TEXT",   // datetime stored as TEXT
		"metadata TEXT",  // json stored as TEXT
	}

	for _, expected := range expectations {
		if !strings.Contains(ddl, expected) {
			t.Errorf("DDL should contain %q: %s", expected, ddl)
		}
	}
}

func TestGenerateIndexDDL(t *testing.T) {
	ddl := GenerateIndexDDL("my_table", "my_field")

	expected := "CREATE INDEX IF NOT EXISTS idx_my_table_my_field ON my_table(my_field)"
	if ddl != expected {
		t.Errorf("GenerateIndexDDL = %q, want %q", ddl, expected)
	}
}

func TestGenerateFTS5DDL(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		wantFTS  bool
		contains []string
	}{
		{
			name: "no FTS fields",
			schema: &Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":   {Type: FieldTypeString, Primary: true},
					"name": {Type: FieldTypeString},
				},
			},
			wantFTS: false,
		},
		{
			name: "with FTS fields",
			schema: &Schema{
				Name: "test",
				Fields: map[string]*Field{
					"id":    {Type: FieldTypeString, Primary: true},
					"title": {Type: FieldTypeString, FTS: true},
					"body":  {Type: FieldTypeString, FTS: true},
				},
			},
			wantFTS:  true,
			contains: []string{"CREATE VIRTUAL TABLE", "test_fts", "USING fts5", "title", "body", "id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ddl := GenerateFTS5DDL(tt.schema)
			if tt.wantFTS {
				if ddl == "" {
					t.Error("expected FTS DDL, got empty string")
				}
				for _, s := range tt.contains {
					if !strings.Contains(ddl, s) {
						t.Errorf("FTS DDL should contain %q: %s", s, ddl)
					}
				}
			} else {
				if ddl != "" {
					t.Errorf("expected empty DDL for no FTS fields, got %q", ddl)
				}
			}
		})
	}
}

func TestGenerateMetaTableDDL(t *testing.T) {
	ddl := GenerateMetaTableDDL()

	if !strings.Contains(ddl, "CREATE TABLE IF NOT EXISTS _meta") {
		t.Errorf("meta DDL should create _meta table: %s", ddl)
	}
	if !strings.Contains(ddl, "key TEXT PRIMARY KEY") {
		t.Errorf("meta DDL should have key column: %s", ddl)
	}
	if !strings.Contains(ddl, "value TEXT") {
		t.Errorf("meta DDL should have value column: %s", ddl)
	}
}

func TestStoredHash(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := openStoreDB(dbPath)
	if err != nil {
		t.Fatalf("openStoreDB: %v", err)
	}
	defer db.Close()

	// Create meta table
	if _, err := db.Exec(GenerateMetaTableDDL()); err != nil {
		t.Fatalf("creating meta table: %v", err)
	}

	// No hash initially
	hash, err := GetStoredHash(db)
	if err != nil {
		t.Fatalf("GetStoredHash: %v", err)
	}
	if hash != "" {
		t.Errorf("expected empty hash initially, got %q", hash)
	}

	// Set hash
	testHash := "abc123def456"
	if err := SetStoredHash(db, testHash); err != nil {
		t.Fatalf("SetStoredHash: %v", err)
	}

	// Get hash back
	hash, err = GetStoredHash(db)
	if err != nil {
		t.Fatalf("GetStoredHash: %v", err)
	}
	if hash != testHash {
		t.Errorf("GetStoredHash = %q, want %q", hash, testHash)
	}

	// Update hash
	newHash := "new_hash_789"
	if err := SetStoredHash(db, newHash); err != nil {
		t.Fatalf("SetStoredHash (update): %v", err)
	}

	hash, err = GetStoredHash(db)
	if err != nil {
		t.Fatalf("GetStoredHash: %v", err)
	}
	if hash != newHash {
		t.Errorf("GetStoredHash after update = %q, want %q", hash, newHash)
	}
}

func TestLastSyncTime(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := openStoreDB(dbPath)
	if err != nil {
		t.Fatalf("openStoreDB: %v", err)
	}
	defer db.Close()

	// Create meta table
	if _, err := db.Exec(GenerateMetaTableDDL()); err != nil {
		t.Fatalf("creating meta table: %v", err)
	}

	// No time initially
	syncTime, err := GetLastSyncTime(db)
	if err != nil {
		t.Fatalf("GetLastSyncTime: %v", err)
	}
	if !syncTime.IsZero() {
		t.Errorf("expected zero time initially, got %v", syncTime)
	}

	// Set time
	now := time.Now().Truncate(time.Second) // Truncate for comparison
	if err := SetLastSyncTime(db, now); err != nil {
		t.Fatalf("SetLastSyncTime: %v", err)
	}

	// Get time back
	syncTime, err = GetLastSyncTime(db)
	if err != nil {
		t.Fatalf("GetLastSyncTime: %v", err)
	}
	// Compare with some tolerance
	if syncTime.Sub(now).Abs() > time.Second {
		t.Errorf("GetLastSyncTime = %v, want ~%v", syncTime, now)
	}
}

func TestPrepareFTSQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"  spaces  ", "spaces"},
		{"", ""},
		{"with:colon", `"with:colon"`},
		{"with*star", `"with*star"`},
		{`with"quote`, `"with""quote"`},
		{"normal query", "normal query"},
		{"query (with) parens", `"query (with) parens"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := PrepareFTSQuery(tt.input)
			if got != tt.want {
				t.Errorf("PrepareFTSQuery(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOpenStoreDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := openStoreDB(dbPath)
	if err != nil {
		t.Fatalf("openStoreDB: %v", err)
	}
	defer db.Close()

	// Verify we can execute SQL
	_, err = db.Exec("CREATE TABLE test (id TEXT)")
	if err != nil {
		t.Errorf("failed to create table: %v", err)
	}

	// Verify we can query
	_, err = db.Query("SELECT * FROM test")
	if err != nil {
		t.Errorf("failed to query: %v", err)
	}
}
