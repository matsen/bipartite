package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegistry_New(t *testing.T) {
	dir := t.TempDir()

	// Create .bipartite dir
	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	// Load non-existent registry (should create empty)
	registry, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	if registry == nil {
		t.Fatal("registry should not be nil")
	}
	if registry.Stores == nil {
		t.Error("Stores should not be nil")
	}
	if len(registry.Stores) != 0 {
		t.Errorf("expected 0 stores, got %d", len(registry.Stores))
	}
}

func TestSaveAndLoadRegistry(t *testing.T) {
	dir := t.TempDir()

	// Create .bipartite dir
	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	// Create and save registry
	registry := &StoreRegistry{
		Stores: map[string]*StoreConfig{
			"store1": {SchemaPath: "schemas/store1.json", Dir: ".bipartite"},
			"store2": {SchemaPath: "schemas/store2.json", Dir: "custom/dir"},
		},
	}

	if err := SaveRegistry(dir, registry); err != nil {
		t.Fatalf("SaveRegistry: %v", err)
	}

	// Verify file exists
	registryPath := filepath.Join(bipartiteDir, "stores.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Error("stores.json not created")
	}

	// Load it back
	loaded, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}

	if len(loaded.Stores) != 2 {
		t.Errorf("expected 2 stores, got %d", len(loaded.Stores))
	}

	if loaded.Stores["store1"] == nil {
		t.Error("store1 should exist")
	}
	if loaded.Stores["store1"].SchemaPath != "schemas/store1.json" {
		t.Errorf("store1.SchemaPath = %q, want %q", loaded.Stores["store1"].SchemaPath, "schemas/store1.json")
	}

	if loaded.Stores["store2"] == nil {
		t.Error("store2 should exist")
	}
	if loaded.Stores["store2"].Dir != "custom/dir" {
		t.Errorf("store2.Dir = %q, want %q", loaded.Stores["store2"].Dir, "custom/dir")
	}
}

func TestLoadRegistry_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	// Create .bipartite dir
	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	// Write invalid JSON
	registryPath := filepath.Join(bipartiteDir, "stores.json")
	if err := os.WriteFile(registryPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("writing invalid registry: %v", err)
	}

	_, err := LoadRegistry(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestRegistryFilename(t *testing.T) {
	// Verify the constant is set correctly
	if RegistryFilename != "stores.json" {
		t.Errorf("RegistryFilename = %q, want %q", RegistryFilename, "stores.json")
	}
}

func TestListStores(t *testing.T) {
	dir := t.TempDir()

	// Create .bipartite dir
	bipartiteDir := filepath.Join(dir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		t.Fatalf("creating .bipartite dir: %v", err)
	}

	// Create schema files
	schemasDir := filepath.Join(dir, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatalf("creating schemas dir: %v", err)
	}

	schema1 := `{"name": "store1", "fields": {"id": {"type": "string", "primary": true}}}`
	schema2 := `{"name": "store2", "fields": {"id": {"type": "string", "primary": true}}}`

	if err := os.WriteFile(filepath.Join(schemasDir, "store1.json"), []byte(schema1), 0644); err != nil {
		t.Fatalf("writing schema1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(schemasDir, "store2.json"), []byte(schema2), 0644); err != nil {
		t.Fatalf("writing schema2: %v", err)
	}

	// Initialize stores
	for _, name := range []string{"store1", "store2"} {
		schemaPath := filepath.Join(schemasDir, name+".json")
		schema, err := ParseSchema(schemaPath)
		if err != nil {
			t.Fatalf("ParseSchema: %v", err)
		}
		store := NewStore(name, schema, bipartiteDir, schemaPath)
		if err := store.Init(dir); err != nil {
			t.Fatalf("Init %s: %v", name, err)
		}
	}

	// List stores
	stores, err := ListStores(dir)
	if err != nil {
		t.Fatalf("ListStores: %v", err)
	}

	if len(stores) != 2 {
		t.Errorf("expected 2 stores, got %d", len(stores))
	}

	// Check store names are in the list
	names := make(map[string]bool)
	for _, s := range stores {
		names[s.Name] = true
	}
	if !names["store1"] || !names["store2"] {
		t.Errorf("expected store1 and store2, got %v", names)
	}
}
