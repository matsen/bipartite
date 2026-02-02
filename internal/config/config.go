// Package config handles repository configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents repository configuration stored in .bipartite/config.yml.
type Config struct {
	PDFRoot    string `yaml:"pdf_root"`              // Absolute path to PDF folder
	PDFReader  string `yaml:"pdf_reader"`            // Reader preference: system, skim, zathura, etc.
	PapersRepo string `yaml:"papers_repo,omitempty"` // Path to bip-papers repository
}

const (
	BipartiteDir = ".bipartite"
	ConfigFile   = "config.yml"
	RefsFile     = "refs.jsonl"
	EdgesFile    = "edges.jsonl"
	ConceptsFile = "concepts.jsonl"
	ProjectsFile = "projects.jsonl"
	ReposFile    = "repos.jsonl"
	CacheDir     = "cache"
	DBFile       = "refs.db"
)

// ValidReaders lists the supported PDF reader values.
var ValidReaders = []string{"system", "skim", "zathura", "evince", "okular"}

// BipartitePath returns the path to the .bipartite directory from a root path.
func BipartitePath(root string) string {
	return filepath.Join(root, BipartiteDir)
}

// ConfigPath returns the path to config.yml from a root path.
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

// ProjectsPath returns the path to projects.jsonl from a root path.
func ProjectsPath(root string) string {
	return filepath.Join(root, BipartiteDir, ProjectsFile)
}

// ReposPath returns the path to repos.jsonl from a root path.
func ReposPath(root string) string {
	return filepath.Join(root, BipartiteDir, ReposFile)
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
// Returns an empty config (not an error) if the file doesn't exist.
func Load(root string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Save writes configuration to the repository at the given root.
func (c *Config) Save(root string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(root), data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// validateDirectoryPath validates that a path exists and is a directory.
// Returns the expanded path and any validation error.
func validateDirectoryPath(path string) (string, error) {
	if path == "" {
		return "", nil // Empty is allowed (not yet configured)
	}

	expandedPath := ExpandTilde(path)

	info, err := os.Stat(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", expandedPath)
		}
		return "", fmt.Errorf("cannot access path: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", expandedPath)
	}

	return expandedPath, nil
}

// ValidatePDFRoot checks that the PDF root path exists and is a directory.
func ValidatePDFRoot(path string) error {
	_, err := validateDirectoryPath(path)
	return err
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

// ExpandTilde expands ~ to the user's home directory.
// Returns the original path unchanged if it doesn't start with ~.
func ExpandTilde(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path // Return original if we can't get home directory
	}

	return filepath.Join(home, path[1:])
}

// ExpandPath is an alias for ExpandTilde for backward compatibility.
// Deprecated: Use ExpandTilde instead.
func ExpandPath(path string) string {
	return ExpandTilde(path)
}

// ValidatePapersRepo checks that the papers repo path exists and is a bipartite repository.
func ValidatePapersRepo(path string) error {
	expandedPath, err := validateDirectoryPath(path)
	if err != nil {
		return err
	}

	if expandedPath != "" && !IsRepository(expandedPath) {
		return fmt.Errorf("not a bipartite repository: %s (no .bipartite directory)", expandedPath)
	}

	return nil
}
