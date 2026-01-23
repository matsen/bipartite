"""Tests for flowc.slack module."""

from __future__ import annotations

import os
from unittest.mock import patch

from flowc.slack import get_webhook_url


class TestGetWebhookUrl:
    """Tests for get_webhook_url."""

    def test_returns_url_from_env(self):
        """Returns webhook URL from environment variable."""
        with patch.dict(
            os.environ, {"SLACK_WEBHOOK_DASM2": "https://hooks.slack.com/test"}
        ):
            url = get_webhook_url("dasm2")
            assert url == "https://hooks.slack.com/test"

    def test_uppercases_channel_name(self):
        """Channel name is uppercased for env var lookup."""
        with patch.dict(
            os.environ, {"SLACK_WEBHOOK_DASM2": "https://hooks.slack.com/test"}
        ):
            url = get_webhook_url("DASM2")
            assert url == "https://hooks.slack.com/test"

    def test_returns_none_when_not_configured(self):
        """Returns None when webhook is not configured."""
        with patch.dict(os.environ, {}, clear=True):
            url = get_webhook_url("unconfigured")
            assert url is None

    def test_scratch_channel(self):
        """Test channel works for scratch."""
        with patch.dict(
            os.environ, {"SLACK_WEBHOOK_SCRATCH": "https://hooks.slack.com/scratch"}
        ):
            url = get_webhook_url("scratch")
            assert url == "https://hooks.slack.com/scratch"
