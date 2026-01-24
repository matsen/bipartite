package flow

import (
	"os"
	"testing"
)

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
