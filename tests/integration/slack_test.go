// Package integration provides integration tests for bipartite commands.
package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/matsen/bipartite/internal/flow"
)

// TestSlackHistoryJSONFormat verifies the JSON output format meets US3 requirements.
// This test requires SLACK_BOT_TOKEN to be set and a configured channel.
func TestSlackHistoryJSONFormat(t *testing.T) {
	// Skip if no token is set (not in CI with credentials)
	if os.Getenv("SLACK_BOT_TOKEN") == "" {
		t.Skip("SLACK_BOT_TOKEN not set, skipping Slack integration test")
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
		t.Skip("No slack.channels configured in nexus sources.json, skipping test")
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
		t.Skip("No channels configured in nexus sources.json")
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
// Note: This test is skipped if a .env file exists in the nexus directory,
// because godotenv.Load() will read the token from disk regardless of env vars.
func TestSlackHistoryMissingToken(t *testing.T) {
	bp := getBPBinary(t)
	nexusDir := getNexusDir(t)

	// Check if nexus has Slack config
	if !hasSlackConfig(nexusDir) {
		t.Skip("No slack.channels configured in nexus sources.json, skipping test")
	}

	// Skip if .env exists (godotenv loads from disk, bypassing env filter)
	if _, err := os.Stat(filepath.Join(nexusDir, ".env")); err == nil {
		t.Skip(".env file exists in nexus directory, cannot test missing token scenario")
	}

	// Run command without token - using a channel that exists in config
	cmd := exec.Command(bp, "slack", "history", "fortnight-goals")
	cmd.Dir = nexusDir
	cmd.Env = filterEnv(os.Environ(), "SLACK_BOT_TOKEN")

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
	if os.Getenv("SLACK_BOT_TOKEN") == "" {
		t.Skip("SLACK_BOT_TOKEN not set, skipping test")
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
	// Prefer NEXUS_DIR env var, fall back to ~/re/nexus
	if dir := os.Getenv("NEXUS_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home directory: %v", err)
	}
	return filepath.Join(home, "re", "nexus")
}

// filterEnv returns a copy of env with the given key removed.
func filterEnv(env []string, key string) []string {
	result := make([]string, 0, len(env))
	prefix := key + "="
	for _, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			continue
		}
		result = append(result, e)
	}
	return result
}

// hasSlackConfig checks if the nexus directory has Slack channels configured.
func hasSlackConfig(nexusDir string) bool {
	sourcesPath := filepath.Join(nexusDir, "sources.json")
	data, err := os.ReadFile(sourcesPath)
	if err != nil {
		return false
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}

	slackRaw, ok := raw["slack"]
	if !ok {
		return false
	}

	var slackConfig struct {
		Channels map[string]interface{} `json:"channels"`
	}
	if err := json.Unmarshal(slackRaw, &slackConfig); err != nil {
		return false
	}

	return len(slackConfig.Channels) > 0
}

// Ensure we import runtime (used by getBPBinary in edge_test.go)
var _ = runtime.GOOS
