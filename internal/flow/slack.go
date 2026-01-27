package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Exit codes for Slack commands.
const (
	ExitSlackMissingToken    = 1 // SLACK_BOT_TOKEN not set
	ExitSlackChannelNotFound = 2 // Channel not in configuration
	ExitSlackNotMember       = 3 // Bot not member of channel
)

// SlackClient provides read access to Slack channels via the API.
type SlackClient struct {
	token      string
	httpClient *http.Client
	userCache  map[string]string
}

// NewSlackClient creates a new SlackClient from SLACK_BOT_TOKEN environment variable.
func NewSlackClient() (*SlackClient, error) {
	token := os.Getenv("SLACK_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN environment variable not set; required for Slack API access")
	}

	return &SlackClient{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
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

// SlackChannelConfig is a configured Slack channel from sources.json.
type SlackChannelConfig struct {
	ID      string `json:"id"`
	Purpose string `json:"purpose"`
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

// GetWebhookURL returns the Slack webhook URL for a channel from environment.
// Looks for SLACK_WEBHOOK_<CHANNEL> environment variable.
func GetWebhookURL(channel string) string {
	envVar := fmt.Sprintf("SLACK_WEBHOOK_%s", strings.ToUpper(channel))
	return os.Getenv(envVar)
}

// PostToSlack posts a message to Slack via webhook.
func PostToSlack(webhookURL, message string) error {
	payload := map[string]string{"text": message}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
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
		return fmt.Errorf("no webhook configured for channel '%s'; set SLACK_WEBHOOK_%s", channel, strings.ToUpper(channel))
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
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("parsing users response: %w", err)
		}
		resp.Body.Close()

		if !result.OK {
			return nil, fmt.Errorf("Slack API error: %s", result.Error)
		}

		// Update cache with users from this page
		for _, user := range result.Members {
			// Prefer display name, fall back to real name, then username
			name := user.Profile.DisplayName
			if name == "" {
				name = user.Profile.RealName
			}
			if name == "" {
				name = user.Name
			}
			c.userCache[user.ID] = name
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
	// Load user cache
	if err := c.loadUserCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load user cache: %v\n", err)
	}

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
			return nil, fmt.Errorf("not_in_channel: bot is not a member of this channel; invite the bot with /invite @bot-name")
		}
		return nil, fmt.Errorf("Slack API error: %s", result.Error)
	}

	// Convert to our Message format
	var messages []Message
	for _, m := range result.Messages {
		// Skip non-user messages (subtypes like channel_join, etc.)
		if m.SubType != "" {
			continue
		}

		// Parse timestamp to get date
		ts, _ := parseSlackTimestamp(m.TS)
		date := ts.Format("2006-01-02")

		// Look up user name (with fallback to API for Enterprise Grid users)
		userName := c.userCache[m.User]
		if userName == "" {
			// Try to fetch individual user (handles Enterprise Grid)
			if fetched := c.lookupUser(m.User); fetched != "" {
				userName = fetched
			} else {
				userName = m.User // Fall back to user ID
			}
		}

		messages = append(messages, Message{
			Timestamp: m.TS,
			UserID:    m.User,
			UserName:  userName,
			Date:      date,
			Text:      m.Text,
		})
	}

	return messages, nil
}

// slackUserInfoResponse is the response from users.info API.
type slackUserInfoResponse struct {
	slackAPIResponse
	User slackUser `json:"user"`
}

// lookupUser fetches a single user by ID via users.info API.
// Updates the cache if successful. Returns empty string on failure.
func (c *SlackClient) lookupUser(userID string) string {
	url := fmt.Sprintf("https://slack.com/api/users.info?user=%s", userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result slackUserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if !result.OK {
		return ""
	}

	// Extract name
	name := result.User.Profile.DisplayName
	if name == "" {
		name = result.User.Profile.RealName
	}
	if name == "" {
		name = result.User.Name
	}

	// Update cache
	c.userCache[userID] = name
	// Save cache (ignore errors)
	_ = c.saveUserCache()

	return name
}

// parseSlackTimestamp parses a Slack timestamp string (e.g., "1737990123.000100").
func parseSlackTimestamp(ts string) (time.Time, error) {
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return time.Time{}, fmt.Errorf("invalid timestamp: %s", ts)
	}
	var sec int64
	fmt.Sscanf(parts[0], "%d", &sec)
	return time.Unix(sec, 0), nil
}

// LoadSlackChannels loads Slack channel configuration from sources.json.
func LoadSlackChannels() (map[string]SlackChannelConfig, error) {
	data, err := os.ReadFile(SourcesFile)
	if err != nil {
		return nil, fmt.Errorf("reading sources.json: %w", err)
	}

	// Parse into a map to extract the slack section
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing sources.json: %w", err)
	}

	slackRaw, ok := raw["slack"]
	if !ok {
		return nil, fmt.Errorf("no 'slack' section in sources.json")
	}

	var slackConfig struct {
		Channels map[string]SlackChannelConfig `json:"channels"`
	}
	if err := json.Unmarshal(slackRaw, &slackConfig); err != nil {
		return nil, fmt.Errorf("parsing slack config: %w", err)
	}

	if len(slackConfig.Channels) == 0 {
		return nil, fmt.Errorf("no channels configured in sources.json slack.channels")
	}

	return slackConfig.Channels, nil
}

// GetSlackChannel returns the configuration for a specific channel.
func GetSlackChannel(channelName string) (*SlackChannelConfig, error) {
	channels, err := LoadSlackChannels()
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
