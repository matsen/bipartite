"""Tests for flowc.digest.cli module."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import pytest

from flowc.digest.cli import format_date_range, parse_duration


class TestParseDuration:
    """Tests for parse_duration."""

    def test_parses_days(self):
        """Parses day durations."""
        assert parse_duration("2d") == timedelta(days=2)
        assert parse_duration("7d") == timedelta(days=7)

    def test_parses_hours(self):
        """Parses hour durations."""
        assert parse_duration("12h") == timedelta(hours=12)
        assert parse_duration("24h") == timedelta(hours=24)

    def test_parses_weeks(self):
        """Parses week durations."""
        assert parse_duration("1w") == timedelta(weeks=1)
        assert parse_duration("2w") == timedelta(weeks=2)

    def test_raises_on_unknown_unit(self):
        """Raises ValueError for unknown unit."""
        with pytest.raises(ValueError, match="Unknown duration unit"):
            parse_duration("5m")

    def test_raises_on_empty_string(self):
        """Raises ValueError for empty string."""
        with pytest.raises(ValueError, match="Invalid duration format"):
            parse_duration("")

    def test_raises_on_too_short(self):
        """Raises ValueError for single character."""
        with pytest.raises(ValueError, match="Invalid duration format"):
            parse_duration("d")

    def test_raises_on_non_numeric(self):
        """Raises ValueError for non-numeric value."""
        with pytest.raises(ValueError, match="Invalid duration format"):
            parse_duration("abcd")


class TestFormatDateRange:
    """Tests for format_date_range."""

    def test_same_month(self):
        """Formats range within same month."""
        since = datetime(2025, 1, 12, tzinfo=timezone.utc)
        until = datetime(2025, 1, 18, tzinfo=timezone.utc)
        assert format_date_range(since, until) == "Jan 12-18"

    def test_different_months(self):
        """Formats range spanning months."""
        since = datetime(2025, 1, 28, tzinfo=timezone.utc)
        until = datetime(2025, 2, 3, tzinfo=timezone.utc)
        assert format_date_range(since, until) == "Jan 28-Feb 03"
