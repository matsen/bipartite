package flow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"gopkg.in/yaml.v3"
)

// ErrSlackNotInChannel is returned when the bot is not a member of the channel.
var ErrSlackNotInChannel = errors.New("bot is not a member of this channel")

// slackAPITimeout is the HTTP client timeout for Slack API calls.
const slackAPITimeout = 30 * time.Second

// newSlackHTTPClient creates an HTTP client configured for Slack API calls.
func newSlackHTTPClient() *http.Client {
	return &http.Client{Timeout: slackAPITimeout}
}

// SlackClient provides read access to Slack channels via the API.
type SlackClient struct {
	token      string
	httpClient *http.Client
	userCache  map[string]string
}

// NewSlackClient creates a new SlackClient from SLACK_BOT_TOKEN environment variable or global config.
func NewSlackClient() (*SlackClient, error) {
	token := config.GetSlackBotToken()
	if token == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN not configured; set environment variable or add to %s", config.GlobalConfigPath())
	}

	return &SlackClient{
		token:      token,
		httpClient: newSlackHTTPClient(),
		userCache:  make(map[string]string),
	}, nil
}

// Message represents a Slack message from history.
type Message struct {
	Timestamp string `json:"ts"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	Date      string `json:"date"`
	Text      string `json:"text"`
}

// HistoryResponse is the JSON output for bip slack history.
type HistoryResponse struct {
	Channel   string    `json:"channel"`
	ChannelID string    `json:"channel_id"`
	Period    Period    `json:"period"`
	Messages  []Message `json:"messages"`
}

// Period represents a time range for queries.
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// SlackChannelConfig is a configured Slack channel from sources.yml.
type SlackChannelConfig struct {
	ID      string `json:"id" yaml:"id"`
	Purpose string `json:"purpose" yaml:"purpose"`
}

// ChannelsResponse is the JSON output for bip slack channels.
type ChannelsResponse struct {
	Channels []ChannelInfo `json:"channels"`
}

// ChannelInfo contains information about a configured channel.
type ChannelInfo struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Purpose string `json:"purpose"`
}

// webhookEnvVar returns the environment variable name for a channel's webhook.
func webhookEnvVar(channel string) string {
	return "SLACK_WEBHOOK_" + strings.ToUpper(channel)
}

// GetWebhookURL returns the Slack webhook URL for a channel.
// Checks SLACK_WEBHOOK_<CHANNEL> environment variable first, then global config.
func GetWebhookURL(channel string) string {
	return config.GetSlackWebhook(channel)
}

// PostToSlack posts a message to Slack via webhook.
func PostToSlack(webhookURL, message string) error {
	payload := map[string]string{"text": message}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	client := newSlackHTTPClient()
	req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("posting to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API error: %s", resp.Status)
	}

	return nil
}

// SendDigest sends a digest message to a Slack channel.
func SendDigest(channel, message string) error {
	webhookURL := GetWebhookURL(channel)
	if webhookURL == "" {
		return fmt.Errorf("no webhook configured for channel '%s'; set %s", channel, webhookEnvVar(channel))
	}
	return PostToSlack(webhookURL, message)
}

// userCachePath returns the path to the Slack user cache file.
func userCachePath() string {
	return filepath.Join(".bipartite", "cache", "slack_users.json")
}

// loadUserCache loads the user ID to name mapping from disk.
func (c *SlackClient) loadUserCache() error {
	path := userCachePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // No cache yet, not an error
	}
	if err != nil {
		return fmt.Errorf("reading user cache: %w", err)
	}

	if err := json.Unmarshal(data, &c.userCache); err != nil {
		return fmt.Errorf("parsing user cache: %w", err)
	}
	return nil
}

// saveUserCache saves the user ID to name mapping to disk.
func (c *SlackClient) saveUserCache() error {
	path := userCachePath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(c.userCache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling user cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing user cache: %w", err)
	}
	return nil
}

// slackAPIResponse is a generic Slack API response wrapper.
type slackAPIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// slackUsersResponse is the response from users.list API.
type slackUsersResponse struct {
	slackAPIResponse
	Members          []slackUser           `json:"members"`
	ResponseMetadata slackResponseMetadata `json:"response_metadata"`
}

// slackResponseMetadata contains pagination info.
type slackResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

// slackUser represents a user from the Slack API.
type slackUser struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Profile slackUserProfile `json:"profile"`
}

// slackUserProfile contains user profile info.
type slackUserProfile struct {
	DisplayName string `json:"display_name"`
	RealName    string `json:"real_name"`
}

// displayNameWithFallback returns the best display name for the user.
// Priority: display name → real name → username.
func (u slackUser) displayNameWithFallback() string {
	if u.Profile.DisplayName != "" {
		return u.Profile.DisplayName
	}
	if u.Profile.RealName != "" {
		return u.Profile.RealName
	}
	return u.Name
}

// GetUsers fetches all users from the Slack workspace and updates the cache.
// Handles pagination to ensure all users are fetched.
func (c *SlackClient) GetUsers() (map[string]string, error) {
	// Try to load existing cache first
	if err := c.loadUserCache(); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: could not load user cache: %v\n", err)
	}

	cursor := ""
	for {
		url := "https://slack.com/api/users.list?limit=200"
		if cursor != "" {
			url += "&cursor=" + cursor
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching users: %w", err)
		}

		var result slackUsersResponse
		err = func() error {
			defer resp.Body.Close()
			return json.NewDecoder(resp.Body).Decode(&result)
		}()
		if err != nil {
			return nil, fmt.Errorf("parsing users response: %w", err)
		}

		if !result.OK {
			return nil, fmt.Errorf("Slack API error: %s", result.Error)
		}

		// Update cache with users from this page
		for _, user := range result.Members {
			c.userCache[user.ID] = user.displayNameWithFallback()
		}

		// Check for more pages
		if result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	// Save updated cache
	if err := c.saveUserCache(); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: could not save user cache: %v\n", err)
	}

	return c.userCache, nil
}

// slackHistoryResponse is the response from conversations.history API.
type slackHistoryResponse struct {
	slackAPIResponse
	Messages []slackMessage `json:"messages"`
	HasMore  bool           `json:"has_more"`
}

// slackMessage represents a message from the Slack API.
type slackMessage struct {
	Type    string `json:"type"`
	User    string `json:"user"`
	Text    string `json:"text"`
	TS      string `json:"ts"`
	SubType string `json:"subtype,omitempty"`
}

// GetChannelHistory fetches messages from a Slack channel.
func (c *SlackClient) GetChannelHistory(channelID string, oldest time.Time, limit int) ([]Message, error) {
	// Validate inputs
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	// Load user cache
	if err := c.loadUserCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load user cache: %v\n", err)
	}

	// Fetch messages from Slack API
	slackMessages, err := c.fetchChannelMessages(channelID, oldest, limit)
	if err != nil {
		return nil, err
	}

	// Convert to our Message format with user name resolution
	return c.convertSlackMessages(slackMessages)
}

// fetchChannelMessages calls the Slack conversations.history API.
func (c *SlackClient) fetchChannelMessages(channelID string, oldest time.Time, limit int) ([]slackMessage, error) {
	url := fmt.Sprintf("https://slack.com/api/conversations.history?channel=%s&oldest=%d&limit=%d",
		channelID, oldest.Unix(), limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching history: %w", err)
	}
	defer resp.Body.Close()

	var result slackHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing history response: %w", err)
	}

	if !result.OK {
		if result.Error == "channel_not_found" || result.Error == "not_in_channel" {
			return nil, fmt.Errorf("%w: invite the bot with /invite @bot-name", ErrSlackNotInChannel)
		}
		return nil, fmt.Errorf("Slack API error: %s", result.Error)
	}

	return result.Messages, nil
}

// convertSlackMessages transforms Slack API messages to our Message format.
func (c *SlackClient) convertSlackMessages(slackMessages []slackMessage) ([]Message, error) {
	var messages []Message
	for _, m := range slackMessages {
		// Skip non-user messages (subtypes like channel_join, etc.)
		if m.SubType != "" {
			continue
		}

		// Parse timestamp to get date
		ts, err := parseSlackTimestamp(m.TS)
		if err != nil {
			return nil, fmt.Errorf("parsing message timestamp: %w", err)
		}

		// Resolve user name
		userName := c.resolveUserNameWithFallback(m.User)

		messages = append(messages, Message{
			Timestamp: m.TS,
			UserID:    m.User,
			UserName:  userName,
			Date:      ts.Format("2006-01-02"),
			Text:      m.Text,
		})
	}
	return messages, nil
}

// resolveUserNameWithFallback looks up a user name from cache, falling back to API lookup.
func (c *SlackClient) resolveUserNameWithFallback(userID string) string {
	if name, ok := c.userCache[userID]; ok {
		return name
	}

	// Try to fetch individual user (handles Enterprise Grid)
	if name, err := c.lookupUser(userID); err == nil && name != "" {
		return name
	}

	return userID // Fall back to user ID
}

// slackUserInfoResponse is the response from users.info API.
type slackUserInfoResponse struct {
	slackAPIResponse
	User slackUser `json:"user"`
}

// lookupUser fetches a single user by ID via users.info API.
// Updates the cache if successful. Returns an error on failure.
func (c *SlackClient) lookupUser(userID string) (string, error) {
	url := fmt.Sprintf("https://slack.com/api/users.info?user=%s", userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching user info: %w", err)
	}
	defer resp.Body.Close()

	var result slackUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing user info response: %w", err)
	}

	if !result.OK {
		return "", fmt.Errorf("Slack API error: %s", result.Error)
	}

	name := result.User.displayNameWithFallback()

	// Update cache
	c.userCache[userID] = name
	if err := c.saveUserCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save user cache: %v\n", err)
	}

	return name, nil
}

// parseSlackTimestamp parses a Slack timestamp string (e.g., "1737990123.000100").
// The format is Unix seconds, optionally followed by a decimal point and microseconds.
func parseSlackTimestamp(ts string) (time.Time, error) {
	if ts == "" {
		return time.Time{}, fmt.Errorf("timestamp cannot be empty")
	}

	// Extract seconds (before decimal point)
	secStr := ts
	if idx := strings.IndexByte(ts, '.'); idx != -1 {
		secStr = ts[:idx]
	}

	if secStr == "" {
		return time.Time{}, fmt.Errorf("invalid Slack timestamp: %q", ts)
	}

	sec, err := strconv.ParseInt(secStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid Slack timestamp %q: %w", ts, err)
	}
	return time.Unix(sec, 0), nil
}

// LoadSlackChannels loads Slack channel configuration from sources.yml in the given nexus directory.
func LoadSlackChannels(nexusPath string) (map[string]SlackChannelConfig, error) {
	data, err := os.ReadFile(SourcesPath(nexusPath))
	if err != nil {
		return nil, fmt.Errorf("reading sources.yml: %w", err)
	}

	// Parse into a struct to extract the slack section
	var sourcesConfig struct {
		Slack struct {
			Channels map[string]SlackChannelConfig `yaml:"channels"`
		} `yaml:"slack"`
	}
	if err := yaml.Unmarshal(data, &sourcesConfig); err != nil {
		return nil, fmt.Errorf("parsing sources.yml: %w", err)
	}

	if len(sourcesConfig.Slack.Channels) == 0 {
		return nil, fmt.Errorf("no 'slack.channels' section in sources.yml")
	}

	return sourcesConfig.Slack.Channels, nil
}

// GetSlackChannel returns the configuration for a specific channel from the given nexus directory.
func GetSlackChannel(nexusPath, channelName string) (*SlackChannelConfig, error) {
	channels, err := LoadSlackChannels(nexusPath)
	if err != nil {
		return nil, err
	}

	config, ok := channels[channelName]
	if !ok {
		// Build list of valid channels for error message
		var names []string
		for name := range channels {
			names = append(names, name)
		}
		return nil, fmt.Errorf("channel '%s' not found in configuration; valid channels: %s", channelName, strings.Join(names, ", "))
	}

	return &config, nil
}

// ResolveUserName resolves a user ID to a display name using the cache.
func (c *SlackClient) ResolveUserName(userID string) string {
	if name, ok := c.userCache[userID]; ok {
		return name
	}
	return userID
}
