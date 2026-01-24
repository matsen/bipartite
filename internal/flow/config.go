package flow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// File paths relative to the nexus directory.
const (
	SourcesFile = "sources.json"
	BeadsDir    = ".beads"
	BeadsFile   = "issues.jsonl"
	StateFile   = ".last-checkin.json"
	CacheFile   = ".flow-cache.json"
	ConfigFile  = "config.json"
)

// Default paths when config.json doesn't exist.
const (
	DefaultCodePath    = "~/re"
	DefaultWritingPath = "~/writing"
)

// Errors.
var (
	ErrNotNexusDir = errors.New("sources.json not found in current directory; run from nexus directory")
	ErrNoRepos     = errors.New("no repos found in sources.json")
)

// ValidateNexusDirectory checks that sources.json exists in the current directory.
func ValidateNexusDirectory() error {
	if _, err := os.Stat(SourcesFile); os.IsNotExist(err) {
		return ErrNotNexusDir
	}
	return nil
}

// LoadSources loads and parses sources.json from the current directory.
func LoadSources() (*Sources, error) {
	data, err := os.ReadFile(SourcesFile)
	if err != nil {
		return nil, fmt.Errorf("reading sources.json: %w", err)
	}

	// First, unmarshal into a raw map to handle mixed types
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing sources.json: %w", err)
	}

	sources := &Sources{
		Boards:  make(map[string]string),
		Context: make(map[string]string),
	}

	// Parse boards if present
	if boardsRaw, ok := raw["boards"]; ok {
		if err := json.Unmarshal(boardsRaw, &sources.Boards); err != nil {
			return nil, fmt.Errorf("parsing boards: %w", err)
		}
	}

	// Parse context if present
	if contextRaw, ok := raw["context"]; ok {
		if err := json.Unmarshal(contextRaw, &sources.Context); err != nil {
			return nil, fmt.Errorf("parsing context: %w", err)
		}
	}

	// Parse code repos
	if codeRaw, ok := raw["code"]; ok {
		sources.Code, err = parseRepoEntries(codeRaw)
		if err != nil {
			return nil, fmt.Errorf("parsing code repos: %w", err)
		}
	}

	// Parse writing repos
	if writingRaw, ok := raw["writing"]; ok {
		sources.Writing, err = parseRepoEntries(writingRaw)
		if err != nil {
			return nil, fmt.Errorf("parsing writing repos: %w", err)
		}
	}

	return sources, nil
}

// parseRepoEntries parses a JSON array that can contain strings or objects.
func parseRepoEntries(data json.RawMessage) ([]RepoEntry, error) {
	var entries []RepoEntry

	// Try to unmarshal as array of interfaces first
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	for _, item := range raw {
		// Try string first
		var str string
		if err := json.Unmarshal(item, &str); err == nil {
			entries = append(entries, RepoEntry{Repo: str})
			continue
		}

		// Try object
		var entry RepoEntry
		if err := json.Unmarshal(item, &entry); err != nil {
			return nil, fmt.Errorf("invalid repo entry: %s", string(item))
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// LoadAllRepos returns all repos from sources.json.
func LoadAllRepos() ([]string, error) {
	sources, err := LoadSources()
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, entry := range sources.Code {
		repos = append(repos, entry.Repo)
	}
	for _, entry := range sources.Writing {
		repos = append(repos, entry.Repo)
	}

	if len(repos) == 0 {
		return nil, ErrNoRepos
	}

	return repos, nil
}

// LoadReposByCategory returns repos for a specific category.
func LoadReposByCategory(category string) ([]string, error) {
	sources, err := LoadSources()
	if err != nil {
		return nil, err
	}

	var entries []RepoEntry
	switch category {
	case "code":
		entries = sources.Code
	case "writing":
		entries = sources.Writing
	default:
		return nil, fmt.Errorf("unknown category: %s", category)
	}

	var repos []string
	for _, entry := range entries {
		repos = append(repos, entry.Repo)
	}
	return repos, nil
}

// LoadReposByChannel returns repos that belong to a specific channel.
func LoadReposByChannel(channel string) ([]string, error) {
	sources, err := LoadSources()
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, entry := range sources.Code {
		if entry.Channel == channel {
			repos = append(repos, entry.Repo)
		}
	}
	for _, entry := range sources.Writing {
		if entry.Channel == channel {
			repos = append(repos, entry.Repo)
		}
	}
	return repos, nil
}

// ListChannels returns all unique channel names from sources.json.
func ListChannels() ([]string, error) {
	sources, err := LoadSources()
	if err != nil {
		return nil, err
	}

	channelSet := make(map[string]bool)
	for _, entry := range sources.Code {
		if entry.Channel != "" {
			channelSet[entry.Channel] = true
		}
	}
	for _, entry := range sources.Writing {
		if entry.Channel != "" {
			channelSet[entry.Channel] = true
		}
	}

	var channels []string
	for ch := range channelSet {
		channels = append(channels, ch)
	}
	sort.Strings(channels)
	return channels, nil
}

// GetDefaultBoard returns the first board from sources.json.
func GetDefaultBoard() (string, error) {
	sources, err := LoadSources()
	if err != nil {
		return "", err
	}

	for key := range sources.Boards {
		return key, nil
	}
	return "", errors.New("no boards configured in sources.json")
}

// LoadConfig loads config.json from the current directory.
// Returns defaults if config.json doesn't exist.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Paths: ConfigPaths{
			Code:    DefaultCodePath,
			Writing: DefaultWritingPath,
		},
	}

	data, err := os.ReadFile(ConfigFile)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config.json: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config.json: %w", err)
	}

	return cfg, nil
}

// ExtractRepoName extracts the repository name from an org/repo string.
func ExtractRepoName(orgRepo string) string {
	parts := strings.Split(orgRepo, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return orgRepo
}

// GetRepoLocalPath maps a GitHub repo (org/name) to its local path.
// Returns the path and whether the repo was found in sources.json.
func GetRepoLocalPath(orgRepo string) (string, bool) {
	sources, err := LoadSources()
	if err != nil {
		return "", false
	}

	cfg, err := LoadConfig()
	if err != nil {
		return "", false
	}

	repoName := ExtractRepoName(orgRepo)

	// Check writing repos first
	for _, entry := range sources.Writing {
		if entry.Repo == orgRepo {
			writingPath := expandPath(cfg.Paths.Writing)
			return filepath.Join(writingPath, repoName), true
		}
	}

	// Check code repos
	for _, entry := range sources.Code {
		if entry.Repo == orgRepo {
			codePath := expandPath(cfg.Paths.Code)
			return filepath.Join(codePath, repoName), true
		}
	}

	return "", false
}

// GetRepoContextPath returns the context file path for a repo if defined.
func GetRepoContextPath(orgRepo string) string {
	sources, err := LoadSources()
	if err != nil {
		return ""
	}

	if relPath, ok := sources.Context[orgRepo]; ok {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, relPath)
	}
	return ""
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// RepoInCategory checks if a repo is in a specific category.
func RepoInCategory(repo, category string) bool {
	sources, err := LoadSources()
	if err != nil {
		return false
	}

	var entries []RepoEntry
	switch category {
	case "code":
		entries = sources.Code
	case "writing":
		entries = sources.Writing
	default:
		return false
	}

	for _, entry := range entries {
		if entry.Repo == repo {
			return true
		}
	}
	return false
}
