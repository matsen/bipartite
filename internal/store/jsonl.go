package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaxJSONLLineCapacity is the maximum buffer size for reading JSONL lines (1MB per line).
const MaxJSONLLineCapacity = 1024 * 1024

// ComputeJSONLHash computes a SHA256 hash of a JSONL file's contents.
func ComputeJSONLHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Empty file hash
			h := sha256.Sum256([]byte{})
			return hex.EncodeToString(h[:]), nil
		}
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ReadAllRecords reads all records from a JSONL file.
func ReadAllRecords(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Empty file returns empty slice
		}
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)

	// Increase buffer size for long lines
	buf := make([]byte, MaxJSONLLineCapacity)
	scanner.Buffer(buf, MaxJSONLLineCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var record Record
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return records, nil
}

// CheckDuplicatePrimaryKey checks if a primary key value already exists in the JSONL file.
func CheckDuplicatePrimaryKey(path string, pkField string, pkValue any) (bool, error) {
	records, err := ReadAllRecords(path)
	if err != nil {
		return false, err
	}

	pkStr := fmt.Sprintf("%v", pkValue)
	for _, record := range records {
		if fmt.Sprintf("%v", record[pkField]) == pkStr {
			return true, nil
		}
	}

	return false, nil
}

// AppendRecord appends a single record to a JSONL file.
func AppendRecord(path string, record Record) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file for append: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encoding record: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("writing record: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// WriteAllRecords writes all records to a JSONL file atomically.
// Uses temp file + rename for atomic operation.
func WriteAllRecords(path string, records []Record) error {
	// Create temp file in same directory for atomic rename
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*.jsonl")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Write all records
	for i, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("encoding record %d: %w", i, err)
		}

		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			return fmt.Errorf("writing record %d: %w", i, err)
		}
		if _, err := tmpFile.WriteString("\n"); err != nil {
			tmpFile.Close()
			return fmt.Errorf("writing newline: %w", err)
		}
	}

	// Close and sync
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	success = true
	return nil
}
