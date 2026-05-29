package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGlobalConfigPath(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Test with custom XDG_CONFIG_HOME
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	path := GlobalConfigPath()
	want := "/custom/config/bip/config.yml"
	if path != want {
		t.Errorf("GlobalConfigPath() = %q, want %q", path, want)
	}

	// Test with empty XDG_CONFIG_HOME (should use ~/.config)
	os.Setenv("XDG_CONFIG_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}
	path = GlobalConfigPath()
	want = filepath.Join(home, ".config", "bip", "config.yml")
	if path != want {
		t.Errorf("GlobalConfigPath() = %q, want %q", path, want)
	}
}

func TestLoadGlobalConfig_NotFound(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Point to a non-existent directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadGlobalConfig() returned nil")
	}

	// Should return empty config
	if cfg.NexusPath != "" {
		t.Errorf("NexusPath = %q, want empty", cfg.NexusPath)
	}
}

func TestLoadGlobalConfig_Valid(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Create config file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgData := GlobalConfig{
		NexusPath:     "~/re/nexus",
		S2APIKey:      "test-s2-key",
		ASTAAPIKey:    "test-asta-key",
		SlackBotToken: "xoxb-test",
		GitHubToken:   "ghp_test",
		SlackWebhooks: map[string]string{
			"dasm2": "https://hooks.slack.com/test",
		},
	}
	data, _ := yaml.Marshal(cfgData)
	configFile := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}

	// Check tilde expansion
	home, _ := os.UserHomeDir()
	wantPath := filepath.Join(home, "re/nexus")
	if cfg.NexusPath != wantPath {
		t.Errorf("NexusPath = %q, want %q", cfg.NexusPath, wantPath)
	}

	if cfg.S2APIKey != "test-s2-key" {
		t.Errorf("S2APIKey = %q, want test-s2-key", cfg.S2APIKey)
	}
	if cfg.ASTAAPIKey != "test-asta-key" {
		t.Errorf("ASTAAPIKey = %q, want test-asta-key", cfg.ASTAAPIKey)
	}
	if cfg.SlackBotToken != "xoxb-test" {
		t.Errorf("SlackBotToken = %q, want xoxb-test", cfg.SlackBotToken)
	}
	if cfg.GitHubToken != "ghp_test" {
		t.Errorf("GitHubToken = %q, want ghp_test", cfg.GitHubToken)
	}
	if cfg.SlackWebhooks["dasm2"] != "https://hooks.slack.com/test" {
		t.Errorf("SlackWebhooks[dasm2] = %q, want https://hooks.slack.com/test", cfg.SlackWebhooks["dasm2"])
	}
}

func TestLoadGlobalConfig_InvalidYAML(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Create invalid config file (tabs are not allowed in YAML indentation)
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(configFile, []byte("key:\n\t- bad"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := LoadGlobalConfig()
	if err == nil {
		t.Error("LoadGlobalConfig() should return error for invalid YAML")
	}
}

func TestGetS2APIKey(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Point to empty config first
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Without config, returns empty
	got := GetS2APIKey()
	if got != "" {
		t.Errorf("GetS2APIKey() = %q, want empty", got)
	}

	// Create config with key
	ResetGlobalConfigCache()
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{S2APIKey: "config-s2-key"}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.yml"), data, 0644)

	got = GetS2APIKey()
	if got != "config-s2-key" {
		t.Errorf("GetS2APIKey() = %q, want config-s2-key", got)
	}
}

func TestGetSlackWebhook(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Point to empty config
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Without config, returns empty
	got := GetSlackWebhook("dasm2")
	if got != "" {
		t.Errorf("GetSlackWebhook() = %q, want empty", got)
	}

	// Create config with webhook
	ResetGlobalConfigCache()
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{
		SlackWebhooks: map[string]string{"dasm2": "https://config-webhook"},
	}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.yml"), data, 0644)

	got = GetSlackWebhook("dasm2")
	if got != "https://config-webhook" {
		t.Errorf("GetSlackWebhook() = %q, want https://config-webhook", got)
	}
}

func TestHelpfulConfigMessage(t *testing.T) {
	msg := HelpfulConfigMessage()
	if msg == "" {
		t.Error("HelpfulConfigMessage() returned empty string")
	}

	// Check that it mentions key elements
	if len(msg) < 50 {
		t.Error("HelpfulConfigMessage() seems too short")
	}
}

func TestGlobalConfigCache(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Create config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{S2APIKey: "cached-key"}
	data, _ := yaml.Marshal(cfgData)
	configFile := filepath.Join(configDir, "config.yml")
	os.WriteFile(configFile, data, 0644)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// First load
	cfg1, _ := LoadGlobalConfig()
	if cfg1.S2APIKey != "cached-key" {
		t.Errorf("First load: S2APIKey = %q, want cached-key", cfg1.S2APIKey)
	}

	// Modify file
	cfgData.S2APIKey = "modified-key"
	data, _ = yaml.Marshal(cfgData)
	os.WriteFile(configFile, data, 0644)

	// Second load should return cached value
	cfg2, _ := LoadGlobalConfig()
	if cfg2.S2APIKey != "cached-key" {
		t.Errorf("Second load: S2APIKey = %q, want cached-key (cached)", cfg2.S2APIKey)
	}

	// Reset cache
	ResetGlobalConfigCache()

	// Third load should read modified file
	cfg3, _ := LoadGlobalConfig()
	if cfg3.S2APIKey != "modified-key" {
		t.Errorf("Third load: S2APIKey = %q, want modified-key", cfg3.S2APIKey)
	}
}

func TestValidateNexusPath_NotConfigured(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Point to empty config directory
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := ValidateNexusPath()
	if err == nil {
		t.Error("ValidateNexusPath() should return error when not configured")
	}
	if err != ErrNexusPathNotConfigured {
		t.Errorf("ValidateNexusPath() error = %v, want ErrNexusPathNotConfigured", err)
	}
}

func TestValidateNexusPath_PathNotExist(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Create config with non-existent path
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{NexusPath: "/nonexistent/path/that/does/not/exist"}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.yml"), data, 0644)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := ValidateNexusPath()
	if err == nil {
		t.Error("ValidateNexusPath() should return error when path doesn't exist")
	}
	if !errors.Is(err, ErrNexusPathNotExist) {
		t.Errorf("ValidateNexusPath() error = %v, want ErrNexusPathNotExist", err)
	}
}

// clearTokenEnv unsets all token env vars that the getters consult, so
// each subtest starts from a known-clean environment regardless of what
// the user's shell exports. t.Setenv restores values on cleanup.
func clearTokenEnv(t *testing.T) {
	t.Helper()
	names := append([]string{}, GitHubTokenEnvVars...)
	names = append(names, SlackBotTokenEnvVars...)
	names = append(names, ASTAAPIKeyEnvVars...)
	for _, name := range names {
		t.Setenv(name, "")
	}
}

// writeConfigWithASTAKey creates a config.yml under XDG_CONFIG_HOME with
// the given asta_api_key and points XDG_CONFIG_HOME at it.
func writeConfigWithASTAKey(t *testing.T, astaKey string) {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := yaml.Marshal(GlobalConfig{ASTAAPIKey: astaKey})
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	ResetGlobalConfigCache()
}

func TestGetASTAAPIKey_EnvPrecedence(t *testing.T) {
	cases := []struct {
		name      string
		envs      map[string]string
		configKey string
		want      string
	}{
		{
			name: "BIP_ASTA_API_KEY wins over ASTA_API_KEY",
			envs: map[string]string{
				"BIP_ASTA_API_KEY": "from-bip",
				"ASTA_API_KEY":     "from-asta",
			},
			configKey: "from-config",
			want:      "from-bip",
		},
		{
			name: "ASTA_API_KEY wins when BIP_ASTA_API_KEY unset",
			envs: map[string]string{
				"ASTA_API_KEY": "from-asta",
			},
			configKey: "from-config",
			want:      "from-asta",
		},
		{
			name:      "config used when no env vars set",
			envs:      nil,
			configKey: "from-config",
			want:      "from-config",
		},
		{
			name: "empty env vars treated as unset",
			envs: map[string]string{
				"BIP_ASTA_API_KEY": "",
				"ASTA_API_KEY":     "",
			},
			configKey: "from-config",
			want:      "from-config",
		},
		{
			name:      "empty config and no env returns empty",
			envs:      nil,
			configKey: "",
			want:      "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearTokenEnv(t)
			writeConfigWithASTAKey(t, tc.configKey)
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			if got := GetASTAAPIKey(); got != tc.want {
				t.Errorf("GetASTAAPIKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

// writeConfigWithTokens creates a config.yml under XDG_CONFIG_HOME with
// the given token values and points XDG_CONFIG_HOME at it. Returns the
// temp dir for cleanup-by-t.TempDir().
func writeConfigWithTokens(t *testing.T, githubToken, slackToken string) {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgData := GlobalConfig{
		GitHubToken:   githubToken,
		SlackBotToken: slackToken,
	}
	data, _ := yaml.Marshal(cfgData)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), data, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	ResetGlobalConfigCache()
}

func TestGetGitHubToken_EnvPrecedence(t *testing.T) {
	cases := []struct {
		name        string
		envs        map[string]string
		configToken string
		want        string
	}{
		{
			name: "BIP_GITHUB_TOKEN wins over all",
			envs: map[string]string{
				"BIP_GITHUB_TOKEN": "from-bip",
				"GITHUB_TOKEN":     "from-github",
				"GH_TOKEN":         "from-gh",
			},
			configToken: "from-config",
			want:        "from-bip",
		},
		{
			name: "GITHUB_TOKEN wins when BIP_GITHUB_TOKEN unset",
			envs: map[string]string{
				"GITHUB_TOKEN": "from-github",
				"GH_TOKEN":     "from-gh",
			},
			configToken: "from-config",
			want:        "from-github",
		},
		{
			name: "GH_TOKEN wins when bip and GITHUB_TOKEN unset",
			envs: map[string]string{
				"GH_TOKEN": "from-gh",
			},
			configToken: "from-config",
			want:        "from-gh",
		},
		{
			name:        "config used when no env vars set",
			envs:        nil,
			configToken: "from-config",
			want:        "from-config",
		},
		{
			name: "empty env vars treated as unset",
			envs: map[string]string{
				"BIP_GITHUB_TOKEN": "",
				"GITHUB_TOKEN":     "",
				"GH_TOKEN":         "",
			},
			configToken: "from-config",
			want:        "from-config",
		},
		{
			name:        "empty config and no env returns empty",
			envs:        nil,
			configToken: "",
			want:        "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearTokenEnv(t)
			writeConfigWithTokens(t, tc.configToken, "")
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			if got := GetGitHubToken(); got != tc.want {
				t.Errorf("GetGitHubToken() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetSlackBotToken_EnvPrecedence(t *testing.T) {
	cases := []struct {
		name        string
		envs        map[string]string
		configToken string
		want        string
	}{
		{
			name: "BIP_SLACK_TOKEN wins over SLACK_BOT_TOKEN",
			envs: map[string]string{
				"BIP_SLACK_TOKEN": "from-bip",
				"SLACK_BOT_TOKEN": "from-slack",
			},
			configToken: "from-config",
			want:        "from-bip",
		},
		{
			name: "SLACK_BOT_TOKEN wins when BIP_SLACK_TOKEN unset",
			envs: map[string]string{
				"SLACK_BOT_TOKEN": "from-slack",
			},
			configToken: "from-config",
			want:        "from-slack",
		},
		{
			name:        "config used when no env vars set",
			envs:        nil,
			configToken: "from-config",
			want:        "from-config",
		},
		{
			name: "empty env vars treated as unset",
			envs: map[string]string{
				"BIP_SLACK_TOKEN": "",
				"SLACK_BOT_TOKEN": "",
			},
			configToken: "from-config",
			want:        "from-config",
		},
		{
			name:        "empty config and no env returns empty",
			envs:        nil,
			configToken: "",
			want:        "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearTokenEnv(t)
			writeConfigWithTokens(t, "", tc.configToken)
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			if got := GetSlackBotToken(); got != tc.want {
				t.Errorf("GetSlackBotToken() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateNexusPath_Valid(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Create config with valid path
	tmpDir := t.TempDir()
	nexusDir := filepath.Join(tmpDir, "nexus")
	os.MkdirAll(nexusDir, 0755)

	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{NexusPath: nexusDir}
	data, _ := yaml.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.yml"), data, 0644)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := ValidateNexusPath()
	if err != nil {
		t.Errorf("ValidateNexusPath() error = %v, want nil", err)
	}
	if path != nexusDir {
		t.Errorf("ValidateNexusPath() = %q, want %q", path, nexusDir)
	}
}
