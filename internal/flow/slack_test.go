package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	// Save and restore original env
	origEnv := os.Getenv("SLACK_WEBHOOK_DASM2")
	defer func() {
		if origEnv != "" {
			os.Setenv("SLACK_WEBHOOK_DASM2", origEnv)
		} else {
			os.Unsetenv("SLACK_WEBHOOK_DASM2")
		}
	}()

	t.Run("returns URL from env", func(t *testing.T) {
		os.Setenv("SLACK_WEBHOOK_DASM2", "https://hooks.slack.com/test")
		url := GetWebhookURL("dasm2")
		if url != "https://hooks.slack.com/test" {
			t.Errorf("GetWebhookURL() = %q, want %q", url, "https://hooks.slack.com/test")
		}
	})

	t.Run("uppercases channel name", func(t *testing.T) {
		os.Setenv("SLACK_WEBHOOK_DASM2", "https://hooks.slack.com/test")
		url := GetWebhookURL("DASM2")
		if url != "https://hooks.slack.com/test" {
			t.Errorf("GetWebhookURL() = %q, want %q", url, "https://hooks.slack.com/test")
		}
	})

	t.Run("returns empty when not configured", func(t *testing.T) {
		os.Unsetenv("SLACK_WEBHOOK_UNCONFIGURED")
		url := GetWebhookURL("unconfigured")
		if url != "" {
			t.Errorf("GetWebhookURL() = %q, want empty string", url)
		}
	})

	t.Run("works for scratch channel", func(t *testing.T) {
		os.Setenv("SLACK_WEBHOOK_SCRATCH", "https://hooks.slack.com/scratch")
		defer os.Unsetenv("SLACK_WEBHOOK_SCRATCH")

		url := GetWebhookURL("scratch")
		if url != "https://hooks.slack.com/scratch" {
			t.Errorf("GetWebhookURL() = %q, want %q", url, "https://hooks.slack.com/scratch")
		}
	})
}

func TestSendDigestError(t *testing.T) {
	// Test that SendDigest returns error when no webhook configured
	os.Unsetenv("SLACK_WEBHOOK_UNCONFIGURED")

	err := SendDigest("unconfigured", "test message")
	if err == nil {
		t.Error("SendDigest() expected error for unconfigured channel")
	}
}

// Tests for Slack reading functionality

func TestNewSlackClient_MissingToken(t *testing.T) {
	// Temporarily unset token
	oldToken := os.Getenv("SLACK_BOT_TOKEN")
	os.Unsetenv("SLACK_BOT_TOKEN")
	defer func() {
		if oldToken != "" {
			os.Setenv("SLACK_BOT_TOKEN", oldToken)
		}
	}()

	_, err := NewSlackClient()
	if err == nil {
		t.Error("expected error for missing token, got nil")
	}
}

func TestNewSlackClient_WithToken(t *testing.T) {
	// Temporarily set token
	oldToken := os.Getenv("SLACK_BOT_TOKEN")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	defer func() {
		if oldToken != "" {
			os.Setenv("SLACK_BOT_TOKEN", oldToken)
		} else {
			os.Unsetenv("SLACK_BOT_TOKEN")
		}
	}()

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
		{"1609459200.000000", 2020}, // Dec 31, 2020 (UTC)
	}

	for _, tt := range tests {
		t.Run(tt.ts, func(t *testing.T) {
			parsed, err := parseSlackTimestamp(tt.ts)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if parsed.Year() != tt.wantYear {
				t.Errorf("expected year %d, got %d", tt.wantYear, parsed.Year())
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
	cleanup := withTempWorkDir(t)
	defer cleanup()

	// Write invalid sources.json (missing slack section)
	sourcesContent := `{"boards": {}}`
	if err := os.WriteFile("sources.json", []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.json: %v", err)
	}

	_, err := LoadSlackChannels()
	if err == nil {
		t.Error("expected error for missing slack section")
	}
}

func TestLoadSlackChannels_Valid(t *testing.T) {
	cleanup := withTempWorkDir(t)
	defer cleanup()

	// Write valid sources.json
	sourcesContent := `{
		"slack": {
			"channels": {
				"fortnight-goals": {"id": "C123", "purpose": "goals"},
				"fortnight-feats": {"id": "C456", "purpose": "retrospectives"}
			}
		}
	}`
	if err := os.WriteFile("sources.json", []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.json: %v", err)
	}

	channels, err := LoadSlackChannels()
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
	cleanup := withTempWorkDir(t)
	defer cleanup()

	// Write valid sources.json with different channels
	sourcesContent := `{
		"slack": {
			"channels": {
				"existing-channel": {"id": "C123", "purpose": "test"}
			}
		}
	}`
	if err := os.WriteFile("sources.json", []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.json: %v", err)
	}

	_, err := GetSlackChannel("nonexistent")
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
