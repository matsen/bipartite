package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matsen/bipartite/internal/config"
)

// withTempWorkDir creates a temp directory and changes to it for the test.
// Returns a cleanup function that restores the original directory.
func withTempWorkDir(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "slack-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)

	return func() {
		os.Chdir(oldDir)
		os.RemoveAll(tmpDir)
	}
}

func TestGetWebhookURL(t *testing.T) {
	// Tests use global config, so we need to set up a temp config
	config.ResetGlobalConfigCache()
	defer config.ResetGlobalConfigCache()

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("returns empty when not configured", func(t *testing.T) {
		config.ResetGlobalConfigCache()
		url := GetWebhookURL("unconfigured")
		if url != "" {
			t.Errorf("GetWebhookURL() = %q, want empty string", url)
		}
	})

	t.Run("returns URL from config", func(t *testing.T) {
		config.ResetGlobalConfigCache()
		configDir := filepath.Join(tmpDir, "bip")
		os.MkdirAll(configDir, 0755)
		cfgData := map[string]interface{}{
			"slack_webhooks": map[string]string{
				"dasm2": "https://hooks.slack.com/test",
			},
		}
		data, _ := json.Marshal(cfgData)
		os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)

		url := GetWebhookURL("dasm2")
		if url != "https://hooks.slack.com/test" {
			t.Errorf("GetWebhookURL() = %q, want %q", url, "https://hooks.slack.com/test")
		}
	})
}

func TestSendDigestError(t *testing.T) {
	// Test that SendDigest returns error when no webhook configured
	config.ResetGlobalConfigCache()
	defer config.ResetGlobalConfigCache()

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Point to empty config
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	err := SendDigest("unconfigured", "test message")
	if err == nil {
		t.Error("SendDigest() expected error for unconfigured channel")
	}
}

// Tests for Slack reading functionality

func TestNewSlackClient_MissingToken(t *testing.T) {
	config.ResetGlobalConfigCache()
	defer config.ResetGlobalConfigCache()

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Point to empty config
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := NewSlackClient()
	if err == nil {
		t.Error("expected error for missing token, got nil")
	}
}

func TestNewSlackClient_WithToken(t *testing.T) {
	config.ResetGlobalConfigCache()
	defer config.ResetGlobalConfigCache()

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Create config with token
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	configDir := filepath.Join(tmpDir, "bip")
	os.MkdirAll(configDir, 0755)
	cfgData := map[string]interface{}{
		"slack_bot_token": "xoxb-test-token",
	}
	data, _ := json.Marshal(cfgData)
	os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)

	client, err := NewSlackClient()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client == nil {
		t.Error("expected non-nil client")
	}
}

func TestUserCache_SaveLoad(t *testing.T) {
	cleanup := withTempWorkDir(t)
	defer cleanup()

	tmpDir, _ := os.Getwd()

	client := &SlackClient{
		userCache: map[string]string{
			"U123": "alice",
			"U456": "bob",
		},
	}

	// Save cache
	if err := client.saveUserCache(); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Verify file was created
	cachePath := filepath.Join(tmpDir, ".bipartite", "cache", "slack_users.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache file was not created")
	}

	// Load into new client
	client2 := &SlackClient{
		userCache: make(map[string]string),
	}
	if err := client2.loadUserCache(); err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	// Verify data
	if client2.userCache["U123"] != "alice" {
		t.Errorf("expected alice, got %s", client2.userCache["U123"])
	}
	if client2.userCache["U456"] != "bob" {
		t.Errorf("expected bob, got %s", client2.userCache["U456"])
	}
}

func TestUserCache_LoadNonExistent(t *testing.T) {
	cleanup := withTempWorkDir(t)
	defer cleanup()

	client := &SlackClient{
		userCache: make(map[string]string),
	}

	// Should not error on missing file
	if err := client.loadUserCache(); err != nil {
		t.Errorf("unexpected error loading non-existent cache: %v", err)
	}

	// Cache should remain empty
	if len(client.userCache) != 0 {
		t.Errorf("expected empty cache after loading non-existent file, got %d entries", len(client.userCache))
	}
}

func TestParseSlackTimestamp(t *testing.T) {
	tests := []struct {
		ts       string
		wantYear int
	}{
		{"1737990123.000100", 2025},
		{"1609459200.000000", 2021}, // Jan 1, 2021 00:00:00 UTC
	}

	for _, tt := range tests {
		t.Run(tt.ts, func(t *testing.T) {
			parsed, err := parseSlackTimestamp(tt.ts)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			// Check year in UTC to avoid timezone-dependent test failures
			if parsed.UTC().Year() != tt.wantYear {
				t.Errorf("expected year %d, got %d", tt.wantYear, parsed.UTC().Year())
			}
		})
	}
}

func TestParseSlackTimestamp_Errors(t *testing.T) {
	tests := []struct {
		name string
		ts   string
	}{
		{"empty string", ""},
		{"invalid integer", "notanumber.000100"},
		{"just dot", ".000100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSlackTimestamp(tt.ts)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestLoadSlackChannels_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid sources.json (missing slack section)
	sourcesContent := `{"boards": {}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "sources.json"), []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.json: %v", err)
	}

	_, err := LoadSlackChannels(tmpDir)
	if err == nil {
		t.Error("expected error for missing slack section")
	}
}

func TestLoadSlackChannels_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	// Write valid sources.json
	sourcesContent := `{
		"slack": {
			"channels": {
				"fortnight-goals": {"id": "C123", "purpose": "goals"},
				"fortnight-feats": {"id": "C456", "purpose": "retrospectives"}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "sources.json"), []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.json: %v", err)
	}

	channels, err := LoadSlackChannels(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}

	if channels["fortnight-goals"].ID != "C123" {
		t.Errorf("expected C123, got %s", channels["fortnight-goals"].ID)
	}
}

func TestGetSlackChannel_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Write valid sources.json with different channels
	sourcesContent := `{
		"slack": {
			"channels": {
				"existing-channel": {"id": "C123", "purpose": "test"}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "sources.json"), []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.json: %v", err)
	}

	_, err := GetSlackChannel(tmpDir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestMessage_JSONFields(t *testing.T) {
	msg := Message{
		Timestamp: "1737990123.000100",
		UserID:    "U123",
		UserName:  "alice",
		Date:      "2025-01-27",
		Text:      "Hello world",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify JSON field names match spec
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	requiredFields := []string{"ts", "user_id", "user_name", "date", "text"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}
}
