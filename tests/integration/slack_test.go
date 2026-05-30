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

	// Run command with XDG_CONFIG_HOME pointing to our config.
	// Strip token env vars (BIP_SLACK_TOKEN, SLACK_BOT_TOKEN) so the env-var
	// precedence in GetSlackBotToken() doesn't supply a token from the dev shell.
	// Use a clean cwd (not nexusDir) so godotenv.Load() in cmd/bip/slack.go can't
	// pick up the nexus's .env and reintroduce SLACK_BOT_TOKEN.
	env := filterEnv(os.Environ(), "XDG_CONFIG_HOME")
	env = filterEnv(env, "BIP_SLACK_TOKEN")
	env = filterEnv(env, "SLACK_BOT_TOKEN")
	cmd := exec.Command(bp, "slack", "history", "fortnight-goals")
	cmd.Dir = t.TempDir()
	cmd.Env = append(env, "XDG_CONFIG_HOME="+tmpConfigDir)

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

// writeTestNexus builds a hermetic test environment: a temp nexus directory
// (marked with .bipartite) holding the given sources.yml content, plus a temp
// XDG_CONFIG_HOME whose bip/config.yml points nexus_path at it. A non-empty
// token is written as slack_bot_token. Both directories are registered for
// automatic cleanup via t.TempDir, so callers need no manual os.RemoveAll.
func writeTestNexus(t *testing.T, sourcesContent, token string) *testEnvSetup {
	t.Helper()

	nexusDir := t.TempDir()
	// .bipartite marks the directory as a bipartite nexus (required for FindRepository).
	if err := os.MkdirAll(filepath.Join(nexusDir, ".bipartite"), 0755); err != nil {
		t.Fatalf("failed to create .bipartite dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nexusDir, "sources.yml"), []byte(sourcesContent), 0644); err != nil {
		t.Fatalf("failed to write sources.yml: %v", err)
	}

	configHome := t.TempDir()
	configDir := filepath.Join(configHome, "bip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create bip config dir: %v", err)
	}
	cfgYAML := fmt.Sprintf("nexus_path: %s\n", nexusDir)
	if token != "" {
		cfgYAML += fmt.Sprintf("slack_bot_token: %s\n", token)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yml"), []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	return &testEnvSetup{TmpDir: nexusDir, TmpConfigDir: configHome}
}

// setupTempDirWithSlackConfig builds a test nexus from a copy of the real
// sources.yml and the configured Slack token, for tests that exercise the live
// Slack API. It skips when no Slack config is present.
func setupTempDirWithSlackConfig(t *testing.T) *testEnvSetup {
	t.Helper()

	nexusDir := getNexusDir(t)
	if !hasSlackConfig(nexusDir) {
		t.Skip("No slack.channels configured in nexus sources.yml, skipping test")
	}
	sourcesData, err := os.ReadFile(filepath.Join(nexusDir, "sources.yml"))
	if err != nil {
		t.Fatalf("failed to read sources.yml: %v", err)
	}
	return writeTestNexus(t, string(sourcesData), config.GetSlackBotToken())
}

// hermeticSourcesYML is a controlled sources.yml for API-free tests. C03T8U5RATY
// appears only in project_channels, C044B4JUE5U is reachable via the reversed
// channels map, and C123 is shared between the two maps to exercise the override
// rule (project_channels wins).
const hermeticSourcesYML = `slack:
  channels:
    antigen:
      id: C044B4JUE5U
      purpose: collab
    old-name:
      id: C123
      purpose: legacy
  project_channels:
    C08JB3LRDU2: flu-mut-rates
    C03T8U5RATY: multidms
    C123: new-name
`

// setupResolveEnv creates a hermetic environment for `bip slack resolve` tests.
// It needs no real nexus and no Slack token, so these tests always run. Returns
// the XDG config dir to pass via XDG_CONFIG_HOME.
func setupResolveEnv(t *testing.T) string {
	t.Helper()
	return writeTestNexus(t, hermeticSourcesYML, "").TmpConfigDir
}

// runResolve runs `bip slack resolve` with the given stdin and hermetic config,
// returning stdout. It fails the test if the command errors.
func runResolve(t *testing.T, bp, configHome, stdin string) string {
	t.Helper()
	cmd := exec.Command(bp, "slack", "resolve")
	cmd.Env = append(filterEnv(os.Environ(), "XDG_CONFIG_HOME"), "XDG_CONFIG_HOME="+configHome)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("resolve failed (exit %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("resolve failed: %v", err)
	}
	return string(out)
}

// TestSlackResolve_EndToEnd drives the full stdin/stdout filter, no token required.
func TestSlackResolve_EndToEnd(t *testing.T) {
	bp := getBPBinary(t)
	configHome := setupResolveEnv(t)

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"known project ID", "• <#C03T8U5RATY>: iterate on spike data", "• #multidms: iterate on spike data"},
		{"reversed channels ID", "<#C044B4JUE5U>", "#antigen"},
		{"empty alias on reversed ID", "<#C044B4JUE5U|>", "#antigen"},
		{"override prefers project_channels", "<#C123>", "#new-name"},
		{"unknown ID passes through", "<#CZZZZ99>", "<#CZZZZ99>"},
		{"alias fallback for unknown", "<#CZZZZ99|fallback>", "#fallback"},
		{"adjacent markup", "<#C08JB3LRDU2><#C03T8U5RATY>", "#flu-mut-rates#multidms"},
		{"no trailing newline preserved", "prefix <#C03T8U5RATY>", "prefix #multidms"},
		{"trailing punctuation", "see <#C03T8U5RATY>.", "see #multidms."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := runResolve(t, bp, configHome, tc.in); got != tc.want {
				t.Errorf("resolve(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSlackResolve_EmptyStdin verifies empty input yields empty output, exit 0.
func TestSlackResolve_EmptyStdin(t *testing.T) {
	bp := getBPBinary(t)
	configHome := setupResolveEnv(t)

	if got := runResolve(t, bp, configHome, ""); got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

// TestSlackResolve_PreservesNonMentionText verifies the filter only touches mentions.
func TestSlackResolve_PreservesNonMentionText(t *testing.T) {
	bp := getBPBinary(t)
	configHome := setupResolveEnv(t)

	in := "line one\nline two with #literal-hashtag and a <#C03T8U5RATY> mention\nline three\n"
	want := "line one\nline two with #literal-hashtag and a #multidms mention\nline three\n"
	if got := runResolve(t, bp, configHome, in); got != want {
		t.Errorf("resolve multi-line = %q, want %q", got, want)
	}
}
