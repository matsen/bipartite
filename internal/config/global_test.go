package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalConfigPath(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Test with custom XDG_CONFIG_HOME
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	path := GlobalConfigPath()
	want := "/custom/config/bip/config.json"
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
	want = filepath.Join(home, ".config", "bip", "config.json")
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
	data, _ := json.Marshal(cfgData)
	configFile := filepath.Join(configDir, "config.json")
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

func TestLoadGlobalConfig_InvalidJSON(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore XDG_CONFIG_HOME
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	// Create invalid config file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configFile, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := LoadGlobalConfig()
	if err == nil {
		t.Error("LoadGlobalConfig() should return error for invalid JSON")
	}
}

func TestGetConfigValue(t *testing.T) {
	// Save and restore env
	orig := os.Getenv("TEST_CONFIG_KEY")
	defer os.Setenv("TEST_CONFIG_KEY", orig)

	// Env var takes priority
	os.Setenv("TEST_CONFIG_KEY", "from-env")
	got := GetConfigValue("TEST_CONFIG_KEY", "from-config")
	if got != "from-env" {
		t.Errorf("GetConfigValue() = %q, want from-env", got)
	}

	// Fall back to config value
	os.Setenv("TEST_CONFIG_KEY", "")
	got = GetConfigValue("TEST_CONFIG_KEY", "from-config")
	if got != "from-config" {
		t.Errorf("GetConfigValue() = %q, want from-config", got)
	}
}

func TestGetS2APIKey(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore env
	orig := os.Getenv("S2_API_KEY")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("S2_API_KEY", orig)
		os.Setenv("XDG_CONFIG_HOME", origXDG)
	}()

	// Point to empty config
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Env var takes priority
	os.Setenv("S2_API_KEY", "env-s2-key")
	got := GetS2APIKey()
	if got != "env-s2-key" {
		t.Errorf("GetS2APIKey() = %q, want env-s2-key", got)
	}

	// Without env var, falls back to config
	os.Setenv("S2_API_KEY", "")
	ResetGlobalConfigCache()

	// Create config
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{S2APIKey: "config-s2-key"}
	data, _ := json.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)

	got = GetS2APIKey()
	if got != "config-s2-key" {
		t.Errorf("GetS2APIKey() = %q, want config-s2-key", got)
	}
}

func TestGetSlackWebhook(t *testing.T) {
	ResetGlobalConfigCache()
	defer ResetGlobalConfigCache()

	// Save and restore env
	orig := os.Getenv("SLACK_WEBHOOK_DASM2")
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("SLACK_WEBHOOK_DASM2", orig)
		os.Setenv("XDG_CONFIG_HOME", origXDG)
	}()

	// Point to empty config
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Env var takes priority
	os.Setenv("SLACK_WEBHOOK_DASM2", "https://env-webhook")
	got := GetSlackWebhook("dasm2")
	if got != "https://env-webhook" {
		t.Errorf("GetSlackWebhook() = %q, want https://env-webhook", got)
	}

	// Without env var, falls back to config
	os.Setenv("SLACK_WEBHOOK_DASM2", "")
	ResetGlobalConfigCache()

	// Create config
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := GlobalConfig{
		SlackWebhooks: map[string]string{"dasm2": "https://config-webhook"},
	}
	data, _ := json.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)

	got = GetSlackWebhook("dasm2")
	if got != "https://config-webhook" {
		t.Errorf("GetSlackWebhook() = %q, want https://config-webhook", got)
	}
}

func TestToUpperSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"dasm2", "DASM2"},
		{"my-channel", "MY_CHANNEL"},
		{"MyChannel", "MYCHANNEL"},
		{"foo_bar", "FOO_BAR"},
		{"test123", "TEST123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toUpperSnake(tt.input)
			if got != tt.want {
				t.Errorf("toUpperSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
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
	data, _ := json.Marshal(cfgData)
	configFile := filepath.Join(configDir, "config.json")
	os.WriteFile(configFile, data, 0644)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// First load
	cfg1, _ := LoadGlobalConfig()
	if cfg1.S2APIKey != "cached-key" {
		t.Errorf("First load: S2APIKey = %q, want cached-key", cfg1.S2APIKey)
	}

	// Modify file
	cfgData.S2APIKey = "modified-key"
	data, _ = json.Marshal(cfgData)
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
