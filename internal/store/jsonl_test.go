package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeJSONLHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Empty file hash
	hash1, err := ComputeJSONLHash(path)
	if err != nil {
		t.Fatalf("ComputeJSONLHash (nonexistent): %v", err)
	}
	if hash1 == "" {
		t.Error("hash should not be empty for nonexistent file")
	}

	// Create file with content
	content := `{"id":"1","name":"test"}
{"id":"2","name":"test2"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	hash2, err := ComputeJSONLHash(path)
	if err != nil {
		t.Fatalf("ComputeJSONLHash: %v", err)
	}
	if hash2 == "" {
		t.Error("hash should not be empty")
	}
	if hash2 == hash1 {
		t.Error("hash should differ from empty file hash")
	}

	// Same content should produce same hash
	hash3, err := ComputeJSONLHash(path)
	if err != nil {
		t.Fatalf("ComputeJSONLHash: %v", err)
	}
	if hash3 != hash2 {
		t.Errorf("hash should be deterministic: %q != %q", hash3, hash2)
	}

	// Different content should produce different hash
	if err := os.WriteFile(path, []byte(`{"id":"different"}`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	hash4, err := ComputeJSONLHash(path)
	if err != nil {
		t.Fatalf("ComputeJSONLHash: %v", err)
	}
	if hash4 == hash2 {
		t.Error("hash should differ for different content")
	}
}

func TestReadAllRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Nonexistent file returns empty slice
	records, err := ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords (nonexistent): %v", err)
	}
	if records != nil {
		t.Errorf("expected nil for nonexistent file, got %v", records)
	}

	// Empty file
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	records, err = ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords (empty): %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}

	// File with records
	content := `{"id":"1","name":"first"}
{"id":"2","name":"second"}
{"id":"3","name":"third"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	records, err = ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
	if records[0]["id"] != "1" {
		t.Errorf("records[0][id] = %v, want %q", records[0]["id"], "1")
	}
	if records[2]["name"] != "third" {
		t.Errorf("records[2][name] = %v, want %q", records[2]["name"], "third")
	}
}

func TestReadAllRecords_EmptyLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// File with empty lines should skip them
	content := `{"id":"1"}

{"id":"2"}

`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	records, err := ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records (skipping empty lines), got %d", len(records))
	}
}

func TestReadAllRecords_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"id":"1"}
not valid json
{"id":"3"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ReadAllRecords(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !contains(err.Error(), "line 2") {
		t.Errorf("error should mention line number: %v", err)
	}
}

func TestCheckDuplicatePrimaryKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"id":"1","name":"first"}
{"id":"2","name":"second"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Existing key
	isDupe, err := CheckDuplicatePrimaryKey(path, "id", "1")
	if err != nil {
		t.Fatalf("CheckDuplicatePrimaryKey: %v", err)
	}
	if !isDupe {
		t.Error("expected duplicate for id=1")
	}

	// Non-existing key
	isDupe, err = CheckDuplicatePrimaryKey(path, "id", "999")
	if err != nil {
		t.Fatalf("CheckDuplicatePrimaryKey: %v", err)
	}
	if isDupe {
		t.Error("expected no duplicate for id=999")
	}

	// Nonexistent file
	isDupe, err = CheckDuplicatePrimaryKey(filepath.Join(dir, "nonexistent.jsonl"), "id", "1")
	if err != nil {
		t.Fatalf("CheckDuplicatePrimaryKey (nonexistent): %v", err)
	}
	if isDupe {
		t.Error("expected no duplicate for nonexistent file")
	}
}

func TestAppendRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Append to new file
	record1 := Record{"id": "1", "name": "first"}
	if err := AppendRecord(path, record1); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}

	records, err := ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}

	// Append another record
	record2 := Record{"id": "2", "name": "second"}
	if err := AppendRecord(path, record2); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}

	records, err = ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestWriteAllRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	records := []Record{
		{"id": "1", "name": "first"},
		{"id": "2", "name": "second"},
		{"id": "3", "name": "third"},
	}

	if err := WriteAllRecords(path, records); err != nil {
		t.Fatalf("WriteAllRecords: %v", err)
	}

	readBack, err := ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(readBack) != 3 {
		t.Errorf("expected 3 records, got %d", len(readBack))
	}

	// Verify content
	for i, r := range readBack {
		expectedID := records[i]["id"]
		if r["id"] != expectedID {
			t.Errorf("record %d: id = %v, want %v", i, r["id"], expectedID)
		}
	}
}

func TestWriteAllRecords_Atomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Write initial content
	initial := []Record{{"id": "1"}}
	if err := WriteAllRecords(path, initial); err != nil {
		t.Fatalf("WriteAllRecords: %v", err)
	}

	// Overwrite with new content
	updated := []Record{{"id": "2"}, {"id": "3"}}
	if err := WriteAllRecords(path, updated); err != nil {
		t.Fatalf("WriteAllRecords: %v", err)
	}

	readBack, err := ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(readBack) != 2 {
		t.Errorf("expected 2 records, got %d", len(readBack))
	}
	if readBack[0]["id"] != "2" {
		t.Errorf("record 0 id = %v, want %q", readBack[0]["id"], "2")
	}
}

func TestWriteAllRecords_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Write initial content
	initial := []Record{{"id": "1"}}
	if err := WriteAllRecords(path, initial); err != nil {
		t.Fatalf("WriteAllRecords: %v", err)
	}

	// Write empty slice (delete all)
	if err := WriteAllRecords(path, []Record{}); err != nil {
		t.Fatalf("WriteAllRecords (empty): %v", err)
	}

	readBack, err := ReadAllRecords(path)
	if err != nil {
		t.Fatalf("ReadAllRecords: %v", err)
	}
	if len(readBack) != 0 {
		t.Errorf("expected 0 records, got %d", len(readBack))
	}
}
