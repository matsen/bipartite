package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathFunctions(t *testing.T) {
	root := "/test/repo"

	tests := []struct {
		name string
		fn   func(string) string
		want string
	}{
		{"BipartitePath", BipartitePath, "/test/repo/.bipartite"},
		{"ConfigPath", ConfigPath, "/test/repo/.bipartite/config.json"},
		{"RefsPath", RefsPath, "/test/repo/.bipartite/refs.jsonl"},
		{"CachePath", CachePath, "/test/repo/.bipartite/cache"},
		{"DBPath", DBPath, "/test/repo/.bipartite/cache/refs.db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn(root)
			if got != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, root, got, tt.want)
			}
		})
	}
}

func TestIsRepository(t *testing.T) {
	tmpDir := t.TempDir()

	// Not a repository initially
	if IsRepository(tmpDir) {
		t.Error("IsRepository() = true for non-repo directory")
	}

	// Create .bipartite directory
	bpDir := filepath.Join(tmpDir, BipartiteDir)
	if err := os.Mkdir(bpDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite: %v", err)
	}

	// Now it should be a repository
	if !IsRepository(tmpDir) {
		t.Error("IsRepository() = false for repo directory")
	}
}

func TestIsRepository_FileNotDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .bipartite as a file, not directory
	bpPath := filepath.Join(tmpDir, BipartiteDir)
	if err := os.WriteFile(bpPath, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("Failed to create .bipartite file: %v", err)
	}

	// Should not be considered a repository
	if IsRepository(tmpDir) {
		t.Error("IsRepository() = true when .bipartite is a file")
	}
}

func TestFindRepository(t *testing.T) {
	// Create nested structure: /tmp/xxx/repo/.bipartite
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	nestedDir := filepath.Join(repoDir, "src", "pkg")

	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}
	if err := os.Mkdir(filepath.Join(repoDir, BipartiteDir), 0755); err != nil {
		t.Fatalf("Failed to create .bipartite: %v", err)
	}

	// Find from nested dir should return repo root
	found, err := FindRepository(nestedDir)
	if err != nil {
		t.Fatalf("FindRepository() error = %v", err)
	}
	if found != repoDir {
		t.Errorf("FindRepository() = %q, want %q", found, repoDir)
	}

	// Find from repo root
	found, err = FindRepository(repoDir)
	if err != nil {
		t.Fatalf("FindRepository() error = %v", err)
	}
	if found != repoDir {
		t.Errorf("FindRepository() = %q, want %q", found, repoDir)
	}
}

func TestFindRepository_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindRepository(tmpDir)
	if err == nil {
		t.Error("FindRepository() should return error when no repo found")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .bipartite directory
	bpDir := filepath.Join(tmpDir, BipartiteDir)
	if err := os.Mkdir(bpDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite: %v", err)
	}

	// Save config
	cfg := &Config{
		PDFRoot:   "/path/to/pdfs",
		PDFReader: "skim",
	}
	if err := cfg.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load config
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.PDFRoot != cfg.PDFRoot {
		t.Errorf("PDFRoot = %q, want %q", loaded.PDFRoot, cfg.PDFRoot)
	}
	if loaded.PDFReader != cfg.PDFReader {
		t.Errorf("PDFReader = %q, want %q", loaded.PDFReader, cfg.PDFReader)
	}
}

func TestLoad_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .bipartite directory but no config
	bpDir := filepath.Join(tmpDir, BipartiteDir)
	if err := os.Mkdir(bpDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite: %v", err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Load() should return error when config not found")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .bipartite directory
	bpDir := filepath.Join(tmpDir, BipartiteDir)
	if err := os.Mkdir(bpDir, 0755); err != nil {
		t.Fatalf("Failed to create .bipartite: %v", err)
	}

	// Write invalid JSON
	if err := os.WriteFile(ConfigPath(tmpDir), []byte("not json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestValidatePDFRoot(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty path", "", false}, // Empty is allowed
		{"valid directory", tmpDir, false},
		{"non-existent path", "/nonexistent/path", true},
		{"file not directory", tmpFile, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePDFRoot(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePDFRoot(%q) error = %v, wantErr = %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePDFRoot_HomeExpansion(t *testing.T) {
	// Test that ~ is expanded
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// ~ should expand to home and be valid (home always exists)
	err = ValidatePDFRoot("~")
	if err != nil {
		t.Errorf("ValidatePDFRoot(~) error = %v", err)
	}

	// ~/nonexistent should fail
	err = ValidatePDFRoot("~/nonexistent_dir_xyz_123")
	if err == nil {
		t.Error("ValidatePDFRoot(~/nonexistent) should return error")
	}

	_ = home // Use home variable
}

func TestValidatePDFReader(t *testing.T) {
	tests := []struct {
		reader  string
		wantErr bool
	}{
		{"", false},        // Empty defaults to system
		{"system", false},
		{"skim", false},
		{"zathura", false},
		{"evince", false},
		{"okular", false},
		{"invalid", true},
		{"adobe", true},
	}

	for _, tt := range tests {
		t.Run(tt.reader, func(t *testing.T) {
			err := ValidatePDFReader(tt.reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePDFReader(%q) error = %v, wantErr = %v", tt.reader, err, tt.wantErr)
			}
		})
	}
}

func TestValidReaders(t *testing.T) {
	// Ensure ValidReaders contains expected values
	expected := []string{"system", "skim", "zathura", "evince", "okular"}

	if len(ValidReaders) != len(expected) {
		t.Errorf("ValidReaders has %d entries, want %d", len(ValidReaders), len(expected))
	}

	for _, e := range expected {
		found := false
		for _, v := range ValidReaders {
			if v == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidReaders missing %q", e)
		}
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have expected values
	if BipartiteDir != ".bipartite" {
		t.Errorf("BipartiteDir = %q, want .bipartite", BipartiteDir)
	}
	if ConfigFile != "config.json" {
		t.Errorf("ConfigFile = %q, want config.json", ConfigFile)
	}
	if RefsFile != "refs.jsonl" {
		t.Errorf("RefsFile = %q, want refs.jsonl", RefsFile)
	}
	if CacheDir != "cache" {
		t.Errorf("CacheDir = %q, want cache", CacheDir)
	}
	if DBFile != "refs.db" {
		t.Errorf("DBFile = %q, want refs.db", DBFile)
	}
}
