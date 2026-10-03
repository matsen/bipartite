// Package config handles repository and global configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig represents configuration stored in ~/.config/bip/config.yml.
type GlobalConfig struct {
	NexusPath     string            `yaml:"nexus_path,omitempty"`
	ASTAAPIKey    string            `yaml:"asta_api_key,omitempty"`
	SlackBotToken string            `yaml:"slack_bot_token,omitempty"`
	GitHubToken   string            `yaml:"github_token,omitempty"`
	SlackWebhooks map[string]string `yaml:"slack_webhooks,omitempty"`
	SpawnAgent    string            `yaml:"spawn_agent,omitempty"`
	NtfyTopic     string            `yaml:"ntfy_topic,omitempty"`

	// Layout, when set, is the per-machine default for repo working-directory
	// resolution. Read by flow.ResolveRepoPath. Optional; an absent block
	// leaves bip in its pre-issue-149 clone-mode behavior.
	Layout *LayoutConfig `yaml:"layout,omitempty"`
}

const (
	// GlobalConfigDir is the directory name under XDG_CONFIG_HOME.
	GlobalConfigDir = "bip"
	// GlobalConfigFile is the config file name.
	GlobalConfigFile = "config.yml"
	// SecretsFile holds tokens as NAME=value lines, kept apart from
	// config.yml because agents read that file routinely.
	SecretsFile = "secrets.env"
)

// globalConfigCache caches the loaded global config.
var globalConfigCache *GlobalConfig

// GlobalConfigPath returns the path to the global config file.
// Respects XDG_CONFIG_HOME, defaults to ~/.config/bip/config.yml.
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing global config: %w", err)
	}

	// Expand tilde in nexus_path
	if cfg.NexusPath != "" {
		cfg.NexusPath = ExpandTilde(cfg.NexusPath)
	}

	globalConfigCache = &cfg
	return &cfg, nil
}

// ResetGlobalConfigCache clears the cached global config and secrets.
// Useful for testing.
func ResetGlobalConfigCache() {
	globalConfigCache = nil
	secretsCache = nil
}

// secretsCache caches the parsed secrets file.
var secretsCache map[string]string

// loadSecrets reads secrets.env beside config.yml: NAME=value lines, with
// an optional leading "export ", optional quotes, and # comments. A missing
// or unreadable file yields no secrets.
func loadSecrets() map[string]string {
	if secretsCache != nil {
		return secretsCache
	}
	secretsCache = map[string]string{}
	path := GlobalConfigPath()
	if path == "" {
		return secretsCache
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(path), SecretsFile))
	if err != nil {
		return secretsCache
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		secretsCache[strings.TrimSpace(name)] = value
	}
	return secretsCache
}

// firstEnvOrConfig returns the first non-empty value for names, looking in
// the environment, then in secrets.env under the same names, and falling
// back to configValue. Empty values are treated as unset.
func firstEnvOrConfig(names []string, configValue string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	secrets := loadSecrets()
	for _, name := range names {
		if v := secrets[name]; v != "" {
			return v
		}
	}
	return configValue
}

// ASTAAPIKeyEnvVars lists the environment variables consulted by
// GetASTAAPIKey, in precedence order. BIP_ASTA_API_KEY is the
// recommended bip-specific name; ASTA_API_KEY is the conventional
// fallback advertised in `bip asta --help` and loaded from a local
// .env file by cmd/bip/asta.go.
var ASTAAPIKeyEnvVars = []string{"BIP_ASTA_API_KEY", "ASTA_API_KEY"}

// GetASTAAPIKey returns the AI2 API key used for both the ASTA MCP API
// and the Semantic Scholar Graph API — AI2 issues a single key for both
// as of the 2026-08 key upgrade.
//
// Precedence:
//  1. $BIP_ASTA_API_KEY
//  2. $ASTA_API_KEY
//  3. the same names in ~/.config/bip/secrets.env
//  4. asta_api_key in ~/.config/bip/config.yml
//
// Empty env vars are treated as unset.
func GetASTAAPIKey() string {
	cfg, _ := LoadGlobalConfig()
	configValue := ""
	if cfg != nil {
		configValue = cfg.ASTAAPIKey
	}
	return firstEnvOrConfig(ASTAAPIKeyEnvVars, configValue)
}

// GitHubTokenEnvVars lists the environment variables consulted by
// GetGitHubToken, in precedence order. BIP_GITHUB_TOKEN is the
// recommended bip-specific name; GITHUB_TOKEN and GH_TOKEN are honored
// as fallbacks for compatibility with the gh CLI and existing setups.
var GitHubTokenEnvVars = []string{"BIP_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"}

// SlackBotTokenEnvVars lists the environment variables consulted by
// GetSlackBotToken, in precedence order. BIP_SLACK_TOKEN is the
// recommended bip-specific name; SLACK_BOT_TOKEN is the conventional
// fallback.
var SlackBotTokenEnvVars = []string{"BIP_SLACK_TOKEN", "SLACK_BOT_TOKEN"}

// GetSlackBotToken returns the Slack bot token.
//
// Precedence:
//  1. $BIP_SLACK_TOKEN
//  2. $SLACK_BOT_TOKEN
//  3. the same names in ~/.config/bip/secrets.env
//  4. slack_bot_token in ~/.config/bip/config.yml
//
// Empty env vars are treated as unset.
func GetSlackBotToken() string {
	cfg, _ := LoadGlobalConfig()
	configValue := ""
	if cfg != nil {
		configValue = cfg.SlackBotToken
	}
	return firstEnvOrConfig(SlackBotTokenEnvVars, configValue)
}

// GetGitHubToken returns the GitHub token.
//
// Precedence:
//  1. $BIP_GITHUB_TOKEN
//  2. $GITHUB_TOKEN
//  3. $GH_TOKEN
//  4. the same names in ~/.config/bip/secrets.env
//  5. github_token in ~/.config/bip/config.yml
//
// Empty env vars are treated as unset.
func GetGitHubToken() string {
	cfg, _ := LoadGlobalConfig()
	configValue := ""
	if cfg != nil {
		configValue = cfg.GitHubToken
	}
	return firstEnvOrConfig(GitHubTokenEnvVars, configValue)
}

// SpawnAgentEnvVars lists the environment variables consulted by
// GetSpawnAgent, in precedence order.
var SpawnAgentEnvVars = []string{"BIP_SPAWN_AGENT"}

// GetSpawnAgent returns the configured agent runner for spawned sessions.
//
// Precedence:
//  1. $BIP_SPAWN_AGENT
//  2. spawn_agent in ~/.config/bip/config.yml
//
// Empty env vars are treated as unset.
func GetSpawnAgent() string {
	cfg, _ := LoadGlobalConfig()
	configValue := ""
	if cfg != nil {
		configValue = cfg.SpawnAgent
	}
	return firstEnvOrConfig(SpawnAgentEnvVars, configValue)
}

// NtfyTopicEnvVars lists the environment variables consulted by
// GetNtfyTopic, in precedence order.
var NtfyTopicEnvVars = []string{"BIP_NTFY_TOPIC"}

// GetNtfyTopic returns the ntfy.sh topic that `bip page` posts to.
//
// Precedence:
//  1. $BIP_NTFY_TOPIC
//  2. ntfy_topic in ~/.config/bip/config.yml
//
// Empty env vars are treated as unset.
func GetNtfyTopic() string {
	cfg, _ := LoadGlobalConfig()
	configValue := ""
	if cfg != nil {
		configValue = cfg.NtfyTopic
	}
	return firstEnvOrConfig(NtfyTopicEnvVars, configValue)
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

// ErrNexusPathNotConfigured is returned when nexus_path is not set in config.
var ErrNexusPathNotConfigured = errors.New("nexus_path not configured")

// ErrNexusPathNotExist is returned when the configured nexus_path doesn't exist.
var ErrNexusPathNotExist = errors.New("nexus_path does not exist")

// ValidateNexusPath returns the nexus path from global config after validation.
// Returns error if not configured or if the path doesn't exist.
// This is the testable version - use MustGetNexusPath for CLI commands.
func ValidateNexusPath() (string, error) {
	path := GetNexusPath()
	if path == "" {
		return "", ErrNexusPathNotConfigured
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("%w: %s", ErrNexusPathNotExist, path)
	}
	return path, nil
}

// MustGetNexusPath returns the nexus path from global config.
// Prints helpful message and exits if not configured or path doesn't exist.
// For testable code, use ValidateNexusPath instead.
func MustGetNexusPath() string {
	path, err := ValidateNexusPath()
	if err != nil {
		if errors.Is(err, ErrNexusPathNotConfigured) {
			fmt.Fprintln(os.Stderr, HelpfulConfigMessage())
		} else {
			fmt.Fprintf(os.Stderr, "Configured nexus_path does not exist: %s\n\n%s\n",
				GetNexusPath(), HelpfulConfigMessage())
		}
		os.Exit(2)
	}
	return path
}

// HelpfulConfigMessage returns a helpful message when nexus_path is not configured.
func HelpfulConfigMessage() string {
	configPath := GlobalConfigPath()
	return fmt.Sprintf(`No bipartite repository found.

Tip: Create %s to set a default nexus:
  mkdir -p %s
  echo 'nexus_path: /path/to/your/nexus' > %s

See https://matsen.github.io/bipartite/guides/getting-started/`,
		configPath,
		filepath.Dir(configPath),
		configPath)
}
