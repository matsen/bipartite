package store

import (
	"os"
	"path/filepath"
	"testing"
)

// testSchema returns a schema for testing.
func testSchema() *Schema {
	return &Schema{
		Name: "test_store",
		Fields: map[string]*Field{
			"id":     {Type: FieldTypeString, Primary: true},
			"name":   {Type: FieldTypeString, FTS: true},
			"count":  {Type: FieldTypeInteger, Index: true},
			"active": {Type: FieldTypeBoolean},
			"status": {Type: FieldTypeString, Enum: []string{"pending", "active", "done"}},
		},
	}
}

// setupTestStore creates a test store in a temp directory.
func setupTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()

	// Create schema file
	schemaPath := filepath.Join(dir, "schema.json")
	schemaJSON := `{
		"name": "test_store",
		"fields": {
			"id": {"type": "string", "primary": true},
			"name": {"type": "string", "fts": true},
			"count": {"type": "integer", "index": true},
			"active": {"type": "boolean"},
			"status": {"type": "string", "enum": ["pending", "active", "done"]}
		}
	}`
	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0644); err != nil {
		t.Fatalf("writing schema: %v", err)
	}

	schema, err := ParseSchema(schemaPath)
	if err != nil {
		t.Fatalf("ParseSchema: %v", err)
	}

	store := NewStore("test_store", schema, dir, schemaPath)
	return store, dir
}

func TestNewStore(t *testing.T) {
	schema := testSchema()
	store := NewStore("my_store", schema, "/tmp/test", "/tmp/test/schema.json")

	if store.Name != "my_store" {
		t.Errorf("Name = %q, want %q", store.Name, "my_store")
	}
	if store.JSONLPath() != "/tmp/test/my_store.jsonl" {
		t.Errorf("JSONLPath = %q, want %q", store.JSONLPath(), "/tmp/test/my_store.jsonl")
	}
	if store.DBPath() != "/tmp/test/my_store.db" {
		t.Errorf("DBPath = %q, want %q", store.DBPath(), "/tmp/test/my_store.db")
	}
}

func TestStoreInit(t *testing.T) {
	store, dir := setupTestStore(t)

	// Create .bipartite dir for registry
	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Check JSONL file created
	if _, err := os.Stat(store.JSONLPath()); os.IsNotExist(err) {
		t.Error("JSONL file not created")
	}

	// Check DB file created
	if _, err := os.Stat(store.DBPath()); os.IsNotExist(err) {
		t.Error("DB file not created")
	}

	// Check registry updated
	registry, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if _, ok := registry.Stores["test_store"]; !ok {
		t.Error("store not registered")
	}
}

func TestStoreAppendAndSync(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Append records
	records := []Record{
		{"id": "1", "name": "first", "count": float64(10), "active": true, "status": "pending"},
		{"id": "2", "name": "second", "count": float64(20), "active": false, "status": "active"},
		{"id": "3", "name": "third", "count": float64(30), "active": true, "status": "done"},
	}

	for _, r := range records {
		if err := store.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Check count
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}

	// Should need sync after append
	needsSync, err := store.NeedsSync()
	if err != nil {
		t.Fatalf("NeedsSync: %v", err)
	}
	if !needsSync {
		t.Error("should need sync after append")
	}

	// Sync
	synced, err := store.Sync()
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if synced != 3 {
		t.Errorf("Sync returned %d, want 3", synced)
	}

	// Should not need sync after sync
	needsSync, err = store.NeedsSync()
	if err != nil {
		t.Fatalf("NeedsSync: %v", err)
	}
	if needsSync {
		t.Error("should not need sync after sync")
	}
}

func TestStoreQuery(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Append and sync
	records := []Record{
		{"id": "1", "name": "first", "count": float64(10), "active": true, "status": "pending"},
		{"id": "2", "name": "second", "count": float64(20), "active": false, "status": "active"},
		{"id": "3", "name": "third", "count": float64(30), "active": true, "status": "done"},
	}

	for _, r := range records {
		if err := store.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	if _, err := store.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Query all
	results, err := store.Query("SELECT * FROM test_store")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Query all: got %d results, want 3", len(results))
	}

	// Query with WHERE
	results, err = store.Query("SELECT * FROM test_store WHERE status = 'pending'")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Query WHERE: got %d results, want 1", len(results))
	}

	// Query with COUNT
	results, err = store.Query("SELECT COUNT(*) as cnt FROM test_store WHERE active = 1")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Query COUNT: got %d results, want 1", len(results))
	}
	// SQLite returns int64 for COUNT
	cnt, ok := results[0]["cnt"].(int64)
	if !ok {
		t.Fatalf("COUNT result type: %T", results[0]["cnt"])
	}
	if cnt != 2 {
		t.Errorf("COUNT = %d, want 2", cnt)
	}
}

func TestStoreAppend_Validation(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	tests := []struct {
		name    string
		record  Record
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid record",
			record:  Record{"id": "1", "name": "test", "count": float64(10), "active": true, "status": "pending"},
			wantErr: false,
		},
		{
			name:    "missing primary key",
			record:  Record{"name": "test"},
			wantErr: true,
			errMsg:  "missing primary key",
		},
		{
			name:    "invalid enum",
			record:  Record{"id": "2", "status": "invalid"},
			wantErr: true,
			errMsg:  "not in enum",
		},
		{
			name:    "wrong type",
			record:  Record{"id": "3", "count": "not a number"},
			wantErr: true,
			errMsg:  "expected integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Append(tt.record)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestStoreAppend_DuplicateKey(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// First append succeeds
	if err := store.Append(Record{"id": "1", "name": "first"}); err != nil {
		t.Fatalf("first Append: %v", err)
	}

	// Duplicate key fails
	err := store.Append(Record{"id": "1", "name": "duplicate"})
	if err == nil {
		t.Error("expected error for duplicate key")
	}
	if !contains(err.Error(), "duplicate primary key") {
		t.Errorf("error should mention duplicate: %v", err)
	}
}

func TestStoreDeleteByID(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Append records
	for i := 1; i <= 3; i++ {
		r := Record{"id": string(rune('0' + i)), "name": "test"}
		if err := store.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Delete one
	if err := store.DeleteByID("2"); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	// Check count
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Errorf("Count after delete = %d, want 2", count)
	}

	// Verify deleted
	records, err := ReadAllRecords(store.JSONLPath())
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	for _, r := range records {
		if r["id"] == "2" {
			t.Error("record with id=2 should be deleted")
		}
	}

	// Delete nonexistent fails
	err = store.DeleteByID("999")
	if err == nil {
		t.Error("expected error for nonexistent id")
	}
}

func TestStoreDeleteWhere(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Append records
	records := []Record{
		{"id": "1", "name": "a", "count": float64(10), "status": "pending"},
		{"id": "2", "name": "b", "count": float64(20), "status": "active"},
		{"id": "3", "name": "c", "count": float64(30), "status": "pending"},
	}
	for _, r := range records {
		if err := store.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Sync before delete (needed to query)
	if _, err := store.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Delete by condition
	deleted, err := store.DeleteWhere("status = 'pending'")
	if err != nil {
		t.Fatalf("DeleteWhere: %v", err)
	}
	if deleted != 2 {
		t.Errorf("DeleteWhere returned %d, want 2", deleted)
	}

	// Check count
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count after delete = %d, want 1", count)
	}

	// Delete with no matches
	if _, err := store.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	deleted, err = store.DeleteWhere("count > 100")
	if err != nil {
		t.Fatalf("DeleteWhere (no match): %v", err)
	}
	if deleted != 0 {
		t.Errorf("DeleteWhere (no match) returned %d, want 0", deleted)
	}
}

func TestStoreInfo(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Append some records
	for i := 1; i <= 5; i++ {
		r := Record{"id": string(rune('0' + i)), "name": "test"}
		if err := store.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	info, err := store.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Name != "test_store" {
		t.Errorf("Info.Name = %q, want %q", info.Name, "test_store")
	}
	if info.Records != 5 {
		t.Errorf("Info.Records = %d, want 5", info.Records)
	}
	if info.JSONLPath != store.JSONLPath() {
		t.Errorf("Info.JSONLPath = %q, want %q", info.JSONLPath, store.JSONLPath())
	}
	if info.DBPath != store.DBPath() {
		t.Errorf("Info.DBPath = %q, want %q", info.DBPath, store.DBPath())
	}
	if info.InSync {
		t.Error("Info.InSync should be false before sync")
	}

	// Sync and check again
	if _, err := store.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	info, err = store.Info()
	if err != nil {
		t.Fatalf("Info after sync: %v", err)
	}
	if !info.InSync {
		t.Error("Info.InSync should be true after sync")
	}
	if info.LastSync.IsZero() {
		t.Error("Info.LastSync should not be zero after sync")
	}
}

func TestStoreFTS(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Append records with searchable text
	records := []Record{
		{"id": "1", "name": "hello world"},
		{"id": "2", "name": "goodbye world"},
		{"id": "3", "name": "hello there"},
	}
	for _, r := range records {
		if err := store.Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	if _, err := store.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// FTS query
	results, err := store.Query("SELECT id FROM test_store WHERE id IN (SELECT id FROM test_store_fts WHERE test_store_fts MATCH 'hello')")
	if err != nil {
		t.Fatalf("FTS Query: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("FTS query: got %d results, want 2", len(results))
	}

	// Verify the right records matched
	ids := make(map[string]bool)
	for _, r := range results {
		ids[r["id"].(string)] = true
	}
	if !ids["1"] || !ids["3"] {
		t.Errorf("FTS should match ids 1 and 3, got %v", ids)
	}
}

func TestOpenStore(t *testing.T) {
	store, dir := setupTestStore(t)

	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	if err := store.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Open the store
	opened, err := OpenStore(dir, "test_store")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	if opened.Name != "test_store" {
		t.Errorf("opened.Name = %q, want %q", opened.Name, "test_store")
	}
	if opened.Schema == nil {
		t.Error("opened.Schema should not be nil")
	}

	// Open nonexistent store
	_, err = OpenStore(dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent store")
	}
}
