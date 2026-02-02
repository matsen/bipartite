// Package integration provides integration tests for bipartite commands.
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"gopkg.in/yaml.v3"
)

// TestSlackHistoryJSONFormat verifies the JSON output format meets US3 requirements.
// This test requires slack_bot_token in global config and a configured channel.
func TestSlackHistoryJSONFormat(t *testing.T) {
	// Skip if no token is configured
	if config.GetSlackBotToken() == "" {
		t.Skip("slack_bot_token not configured, skipping Slack integration test")
	}

	bp := getBPBinary(t)
	nexusDir := getNexusDir(t)

	// Run bip slack history command
	cmd := exec.Command(bp, "slack", "history", "fortnight-goals", "--days", "7", "--limit", "10")
	cmd.Dir = nexusDir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("command failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("command failed: %v", err)
	}

	// Parse JSON output
	var response flow.HistoryResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, string(output))
	}

	// Verify required fields are present (US3 requirement)
	if response.Channel == "" {
		t.Error("channel field is empty")
	}
	if response.ChannelID == "" {
		t.Error("channel_id field is empty")
	}
	if response.Period.Start == "" {
		t.Error("period.start field is empty")
	}
	if response.Period.End == "" {
		t.Error("period.end field is empty")
	}

	// Verify message structure if we got any messages
	for i, msg := range response.Messages {
		if msg.Timestamp == "" {
			t.Errorf("message[%d].ts is empty", i)
		}
		if msg.UserID == "" {
			t.Errorf("message[%d].user_id is empty", i)
		}
		if msg.Date == "" {
			t.Errorf("message[%d].date is empty", i)
		}
		if msg.Text == "" {
			t.Errorf("message[%d].text is empty", i)
		}
		// user_name might be empty if user lookup failed, but should exist
	}
}

// TestSlackChannelsJSONFormat verifies the channels command JSON output.
func TestSlackChannelsJSONFormat(t *testing.T) {
	bp := getBPBinary(t)
	nexusDir := getNexusDir(t)

	// Check if nexus has Slack config
	if !hasSlackConfig(nexusDir) {
		t.Skip("No slack.channels configured in nexus sources.yml, skipping test")
	}

	// Run bip slack channels command
	cmd := exec.Command(bp, "slack", "channels")
	cmd.Dir = nexusDir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("command failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("command failed: %v", err)
	}

	// Parse JSON output
	var response flow.ChannelsResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, string(output))
	}

	// Verify we have channels
	if len(response.Channels) == 0 {
		t.Skip("No channels configured in nexus sources.yml")
	}

	// Verify channel structure
	for i, ch := range response.Channels {
		if ch.Name == "" {
			t.Errorf("channel[%d].name is empty", i)
		}
		if ch.ID == "" {
			t.Errorf("channel[%d].id is empty", i)
		}
	}
}

// TestSlackHistoryMissingToken verifies proper error handling for missing token.
// This test creates a temp config without slack_bot_token to verify error handling.
func TestSlackHistoryMissingToken(t *testing.T) {
	bp := getBPBinary(t)
	nexusDir := getNexusDir(t)

	// Check if nexus has Slack config
	if !hasSlackConfig(nexusDir) {
		t.Skip("No slack.channels configured in nexus sources.yml, skipping test")
	}

	// Create a temp config directory without slack_bot_token but WITH nexus_path
	tmpConfigDir := t.TempDir()
	configDir := filepath.Join(tmpConfigDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	// Write config with nexus_path but no slack_bot_token
	cfgYAML := fmt.Sprintf("nexus_path: %s\n", nexusDir)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Run command with XDG_CONFIG_HOME pointing to our config
	cmd := exec.Command(bp, "slack", "history", "fortnight-goals")
	cmd.Dir = nexusDir
	cmd.Env = append(filterEnv(os.Environ(), "XDG_CONFIG_HOME"), "XDG_CONFIG_HOME="+tmpConfigDir)

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for missing token, got success")
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	}
}

// TestSlackHistoryInvalidChannel verifies error handling for unknown channel.
func TestSlackHistoryInvalidChannel(t *testing.T) {
	if config.GetSlackBotToken() == "" {
		t.Skip("slack_bot_token not configured, skipping test")
	}

	bp := getBPBinary(t)
	nexusDir := getNexusDir(t)

	// Run command with invalid channel
	cmd := exec.Command(bp, "slack", "history", "nonexistent-channel-xyz")
	cmd.Dir = nexusDir
	err := cmd.Run()

	if err == nil {
		t.Fatal("expected error for invalid channel, got success")
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 2 {
			t.Errorf("expected exit code 2 (channel not found), got %d", exitErr.ExitCode())
		}
	}
}

// getNexusDir returns the nexus directory for tests.
func getNexusDir(t *testing.T) string {
	t.Helper()
	if dir := config.GetNexusPath(); dir != "" {
		return dir
	}
	t.Skip("nexus_path not configured in global config")
	return ""
}

// filterEnv returns a copy of env with the given key removed.
func filterEnv(env []string, key string) []string {
	result := make([]string, 0, len(env))
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		result = append(result, e)
	}
	return result
}

// hasSlackConfig checks if the nexus directory has Slack channels configured.
func hasSlackConfig(nexusDir string) bool {
	sourcesPath := filepath.Join(nexusDir, "sources.yml")
	data, err := os.ReadFile(sourcesPath)
	if err != nil {
		return false
	}

	var raw struct {
		Slack struct {
			Channels map[string]interface{} `yaml:"channels"`
		} `yaml:"slack"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false
	}

	return len(raw.Slack.Channels) > 0
}

// TestSlackIngestMissingStore verifies error when store doesn't exist.
func TestSlackIngestMissingStore(t *testing.T) {
	if config.GetSlackBotToken() == "" {
		t.Skip("slack_bot_token not configured, skipping test")
	}

	bp := getBPBinary(t)
	nexusDir := getNexusDir(t)

	// Check if nexus has Slack config
	if !hasSlackConfig(nexusDir) {
		t.Skip("No slack.channels configured in nexus sources.yml, skipping test")
	}

	// Run ingest with nonexistent store
	cmd := exec.Command(bp, "slack", "ingest", "fortnight-goals", "--store", "nonexistent_test_store_xyz")
	cmd.Dir = nexusDir
	err := cmd.Run()

	if err == nil {
		t.Fatal("expected error for missing store, got success")
	}

	// Should fail with exit code 1 (store error)
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1 (store error), got %d", exitErr.ExitCode())
		}
	}
}

// TestSlackIngestJSONFormat verifies the JSON output format for successful ingest.
// This test creates a temporary store and ingests messages into it.
func TestSlackIngestJSONFormat(t *testing.T) {
	if config.GetSlackBotToken() == "" {
		t.Skip("slack_bot_token not configured, skipping test")
	}

	bp := getBPBinary(t)
	setup := setupTempDirWithSlackConfig(t)
	defer os.RemoveAll(setup.TmpDir)
	defer os.RemoveAll(setup.TmpConfigDir)

	// Run ingest with --create-store to create a new store
	cmd := exec.Command(bp, "slack", "ingest", "fortnight-goals", "--store", "test_slack_msgs", "--create-store", "--limit", "5", "--days", "7")
	cmd.Dir = setup.TmpDir
	cmd.Env = append(filterEnv(os.Environ(), "XDG_CONFIG_HOME"), "XDG_CONFIG_HOME="+setup.TmpConfigDir)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("command failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("command failed: %v", err)
	}

	// Parse JSON output
	var result struct {
		Channel      string `json:"channel"`
		Store        string `json:"store"`
		Ingested     int    `json:"ingested"`
		Skipped      int    `json:"skipped"`
		StoreCreated bool   `json:"store_created"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, string(output))
	}

	// Verify required fields
	if result.Channel != "fortnight-goals" {
		t.Errorf("expected channel 'fortnight-goals', got %q", result.Channel)
	}
	if result.Store != "test_slack_msgs" {
		t.Errorf("expected store 'test_slack_msgs', got %q", result.Store)
	}
	if !result.StoreCreated {
		t.Error("expected store_created to be true")
	}

	// Verify store was actually created
	storePath := filepath.Join(setup.TmpDir, ".bipartite", "test_slack_msgs.jsonl")
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Error("store JSONL file was not created")
	}

	// Verify schema was created
	schemaPath := filepath.Join(setup.TmpDir, ".bipartite", "schemas", "test_slack_msgs.json")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Error("store schema file was not created")
	}
}

// TestSlackIngestIdempotency verifies running ingest twice skips duplicates.
func TestSlackIngestIdempotency(t *testing.T) {
	if config.GetSlackBotToken() == "" {
		t.Skip("slack_bot_token not configured, skipping test")
	}

	bp := getBPBinary(t)
	setup := setupTempDirWithSlackConfig(t)
	defer os.RemoveAll(setup.TmpDir)
	defer os.RemoveAll(setup.TmpConfigDir)

	// First ingest - creates store and ingests messages
	cmd := exec.Command(bp, "slack", "ingest", "fortnight-goals", "--store", "idem_test_store", "--create-store", "--limit", "3", "--days", "7")
	cmd.Dir = setup.TmpDir
	cmd.Env = append(filterEnv(os.Environ(), "XDG_CONFIG_HOME"), "XDG_CONFIG_HOME="+setup.TmpConfigDir)
	output1, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("first ingest failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("first ingest failed: %v", err)
	}

	var result1 struct {
		Ingested int `json:"ingested"`
		Skipped  int `json:"skipped"`
	}
	if err := json.Unmarshal(output1, &result1); err != nil {
		t.Fatalf("failed to parse first ingest output: %v", err)
	}

	// Second ingest - should skip duplicates
	cmd2 := exec.Command(bp, "slack", "ingest", "fortnight-goals", "--store", "idem_test_store", "--limit", "3", "--days", "7")
	cmd2.Dir = setup.TmpDir
	cmd2.Env = append(filterEnv(os.Environ(), "XDG_CONFIG_HOME"), "XDG_CONFIG_HOME="+setup.TmpConfigDir)
	output2, err := cmd2.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("second ingest failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("second ingest failed: %v", err)
	}

	var result2 struct {
		Ingested int `json:"ingested"`
		Skipped  int `json:"skipped"`
	}
	if err := json.Unmarshal(output2, &result2); err != nil {
		t.Fatalf("failed to parse second ingest output: %v", err)
	}

	// Second run should skip the same messages that were ingested first time
	if result2.Skipped != result1.Ingested {
		t.Errorf("expected second run to skip %d messages (same as first ingested), got skipped=%d", result1.Ingested, result2.Skipped)
	}
	if result2.Ingested != 0 {
		t.Errorf("expected second run to ingest 0 new messages, got %d", result2.Ingested)
	}
}

// Ensure we import runtime (used by getBPBinary in edge_test.go)
var _ = runtime.GOOS

// testEnvSetup holds paths for integration test environment setup.
type testEnvSetup struct {
	TmpDir       string // Temp directory for test nexus
	TmpConfigDir string // Temp XDG_CONFIG_HOME directory
}

// setupTempDirWithSlackConfig creates a temp directory with sources.yml and a temp global config.
// Returns paths for cleanup. The caller is responsible for cleanup with os.RemoveAll.
func setupTempDirWithSlackConfig(t *testing.T) *testEnvSetup {
	t.Helper()

	// Create temp nexus directory
	tmpDir, err := os.MkdirTemp("", "slack-test-nexus-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	nexusDir := getNexusDir(t)
	if !hasSlackConfig(nexusDir) {
		os.RemoveAll(tmpDir)
		t.Skip("No slack.channels configured in nexus sources.yml, skipping test")
	}

	// Create .bipartite directory structure (required for FindRepository)
	bipartiteDir := filepath.Join(tmpDir, ".bipartite")
	if err := os.MkdirAll(bipartiteDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create .bipartite dir: %v", err)
	}

	// Copy sources.yml
	sourcesData, err := os.ReadFile(filepath.Join(nexusDir, "sources.yml"))
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to read sources.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "sources.yml"), sourcesData, 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to write sources.yml: %v", err)
	}

	// Create temp config directory with nexus_path and slack_bot_token
	tmpConfigDir, err := os.MkdirTemp("", "slack-test-config-*")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create temp config dir: %v", err)
	}
	configDir := filepath.Join(tmpConfigDir, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		os.RemoveAll(tmpConfigDir)
		t.Fatalf("failed to create bip config dir: %v", err)
	}

	// Get slack_bot_token from current config
	slackToken := config.GetSlackBotToken()
	cfgYAML := fmt.Sprintf("nexus_path: %s\nslack_bot_token: %s\n", tmpDir, slackToken)
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(cfgYAML), 0644); err != nil {
		os.RemoveAll(tmpDir)
		os.RemoveAll(tmpConfigDir)
		t.Fatalf("failed to write config: %v", err)
	}

	return &testEnvSetup{
		TmpDir:       tmpDir,
		TmpConfigDir: tmpConfigDir,
	}
}
