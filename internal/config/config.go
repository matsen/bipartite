// Package config handles repository configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents repository configuration stored in .bipartite/config.json.
type Config struct {
	PDFRoot    string `json:"pdf_root"`              // Absolute path to PDF folder
	PDFReader  string `json:"pdf_reader"`            // Reader preference: system, skim, zathura, etc.
	PapersRepo string `json:"papers_repo,omitempty"` // Path to bip-papers repository
}

const (
	BipartiteDir = ".bipartite"
	ConfigFile   = "config.json"
	RefsFile     = "refs.jsonl"
	EdgesFile    = "edges.jsonl"
	ConceptsFile = "concepts.jsonl"
	CacheDir     = "cache"
	DBFile       = "refs.db"
)

// ValidReaders lists the supported PDF reader values.
var ValidReaders = []string{"system", "skim", "zathura", "evince", "okular"}

// BipartitePath returns the path to the .bipartite directory from a root path.
func BipartitePath(root string) string {
	return filepath.Join(root, BipartiteDir)
}

// ConfigPath returns the path to config.json from a root path.
func ConfigPath(root string) string {
	return filepath.Join(root, BipartiteDir, ConfigFile)
}

// RefsPath returns the path to refs.jsonl from a root path.
func RefsPath(root string) string {
	return filepath.Join(root, BipartiteDir, RefsFile)
}

// EdgesPath returns the path to edges.jsonl from a root path.
func EdgesPath(root string) string {
	return filepath.Join(root, BipartiteDir, EdgesFile)
}

// ConceptsPath returns the path to concepts.jsonl from a root path.
func ConceptsPath(root string) string {
	return filepath.Join(root, BipartiteDir, ConceptsFile)
}

// CachePath returns the path to the cache directory from a root path.
func CachePath(root string) string {
	return filepath.Join(root, BipartiteDir, CacheDir)
}

// DBPath returns the path to refs.db from a root path.
func DBPath(root string) string {
	return filepath.Join(root, BipartiteDir, CacheDir, DBFile)
}

// IsRepository checks if the given path contains a bipartite repository.
func IsRepository(root string) bool {
	info, err := os.Stat(BipartitePath(root))
	return err == nil && info.IsDir()
}

// FindRepository walks up from the given path to find a bipartite repository.
// Returns the repository root path or an error if not found.
func FindRepository(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	for {
		if IsRepository(abs) {
			return abs, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("not in a bipartite repository (no .bipartite directory found)")
		}
		abs = parent
	}
}

// Load reads configuration from the repository at the given root.
func Load(root string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(root))
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Save writes configuration to the repository at the given root.
func (c *Config) Save(root string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(root), data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// ValidatePDFRoot checks that the PDF root path exists and is a directory.
func ValidatePDFRoot(path string) error {
	if path == "" {
		return nil // Empty is allowed (not yet configured)
	}

	// Expand ~ to home directory
	expandedPath := ExpandPath(path)

	info, err := os.Stat(expandedPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", expandedPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", expandedPath)
	}

	return nil
}

// ValidatePDFReader checks that the reader value is valid.
func ValidatePDFReader(reader string) error {
	if reader == "" {
		return nil // Empty defaults to "system"
	}

	for _, valid := range ValidReaders {
		if reader == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid pdf_reader: %s (valid: %v)", reader, ValidReaders)
}

// ExpandPath expands ~ to the user's home directory.
// Returns the original path unchanged if it doesn't start with ~.
func ExpandPath(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path // Return original if we can't get home directory
	}

	return filepath.Join(home, path[1:])
}

// ValidatePapersRepo checks that the papers repo path exists and is a bipartite repository.
func ValidatePapersRepo(path string) error {
	if path == "" {
		return nil // Empty is allowed (not yet configured)
	}

	// Expand ~ to home directory
	expandedPath := ExpandPath(path)

	if !IsRepository(expandedPath) {
		return fmt.Errorf("not a bipartite repository: %s (no .bipartite directory)", expandedPath)
	}

	return nil
}
