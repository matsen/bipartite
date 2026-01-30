// Package config handles repository and global configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GlobalConfig represents configuration stored in ~/.config/bip/config.json.
type GlobalConfig struct {
	NexusPath     string            `json:"nexus_path,omitempty"`
	S2APIKey      string            `json:"s2_api_key,omitempty"`
	ASTAAPIKey    string            `json:"asta_api_key,omitempty"`
	SlackBotToken string            `json:"slack_bot_token,omitempty"`
	GitHubToken   string            `json:"github_token,omitempty"`
	SlackWebhooks map[string]string `json:"slack_webhooks,omitempty"`
}

const (
	// GlobalConfigDir is the directory name under XDG_CONFIG_HOME.
	GlobalConfigDir = "bip"
	// GlobalConfigFile is the config file name.
	GlobalConfigFile = "config.json"
)

// globalConfigCache caches the loaded global config.
var globalConfigCache *GlobalConfig

// GlobalConfigPath returns the path to the global config file.
// Respects XDG_CONFIG_HOME, defaults to ~/.config/bip/config.json.
func GlobalConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, GlobalConfigDir, GlobalConfigFile)
}

// LoadGlobalConfig loads the global configuration file.
// Returns an empty config (not an error) if the file doesn't exist.
func LoadGlobalConfig() (*GlobalConfig, error) {
	if globalConfigCache != nil {
		return globalConfigCache, nil
	}

	path := GlobalConfigPath()
	if path == "" {
		return &GlobalConfig{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{}, nil
		}
		return nil, fmt.Errorf("reading global config: %w", err)
	}

	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}

	// Expand tilde in nexus_path
	if cfg.NexusPath != "" {
		cfg.NexusPath = ExpandTilde(cfg.NexusPath)
	}

	globalConfigCache = &cfg
	return &cfg, nil
}

// ResetGlobalConfigCache clears the cached global config.
// Useful for testing.
func ResetGlobalConfigCache() {
	globalConfigCache = nil
}

// GetS2APIKey returns the Semantic Scholar API key from global config.
func GetS2APIKey() string {
	cfg, _ := LoadGlobalConfig()
	return cfg.S2APIKey
}

// GetASTAAPIKey returns the ASTA API key from global config.
func GetASTAAPIKey() string {
	cfg, _ := LoadGlobalConfig()
	return cfg.ASTAAPIKey
}

// GetSlackBotToken returns the Slack bot token from global config.
func GetSlackBotToken() string {
	cfg, _ := LoadGlobalConfig()
	return cfg.SlackBotToken
}

// GetGitHubToken returns the GitHub token from global config.
func GetGitHubToken() string {
	cfg, _ := LoadGlobalConfig()
	return cfg.GitHubToken
}

// GetSlackWebhook returns the Slack webhook URL for a channel from global config.
func GetSlackWebhook(channel string) string {
	cfg, _ := LoadGlobalConfig()
	if cfg.SlackWebhooks != nil {
		return cfg.SlackWebhooks[channel]
	}
	return ""
}

// GetNexusPath returns the configured nexus path from global config.
func GetNexusPath() string {
	cfg, _ := LoadGlobalConfig()
	return cfg.NexusPath
}

// MustGetNexusPath returns the nexus path from global config.
// Prints helpful message and exits if not configured or path doesn't exist.
func MustGetNexusPath() string {
	path := GetNexusPath()
	if path == "" {
		fmt.Fprintln(os.Stderr, HelpfulConfigMessage())
		os.Exit(2) // ExitConfigError
	}
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(os.Stderr, "Configured nexus_path does not exist: %s\n\n%s\n",
			path, HelpfulConfigMessage())
		os.Exit(2)
	}
	return path
}

// HelpfulConfigMessage returns a helpful message when no repository is found.
func HelpfulConfigMessage() string {
	configPath := GlobalConfigPath()
	return fmt.Sprintf(`No bipartite repository found.

Tip: Create %s to set a default nexus:
  mkdir -p %s
  echo '{"nexus_path": "/path/to/your/nexus"}' > %s

See https://matsen.github.io/bipartite/guides/getting-started/`,
		configPath,
		filepath.Dir(configPath),
		configPath)
}
