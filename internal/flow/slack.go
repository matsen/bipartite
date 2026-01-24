package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

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
