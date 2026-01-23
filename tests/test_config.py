"""Tests for flowc.shared.config module."""

from __future__ import annotations

import json
from unittest.mock import patch

import pytest

from flowc.shared.config import (
    _repo_in_category,
    list_channels,
    load_repos,
    load_repos_by_channel,
    normalize_repo_entry,
)


class TestNormalizeRepoEntry:
    """Tests for normalize_repo_entry."""

    def test_string_entry(self):
        """String entries are returned as-is."""
        assert normalize_repo_entry("matsengrp/repo") == "matsengrp/repo"

    def test_dict_entry(self):
        """Dict entries have their 'repo' key extracted."""
        entry = {"repo": "matsengrp/repo", "channel": "dasm2"}
        assert normalize_repo_entry(entry) == "matsengrp/repo"

    def test_dict_entry_without_channel(self):
        """Dict entries without channel still work."""
        entry = {"repo": "matsengrp/repo"}
        assert normalize_repo_entry(entry) == "matsengrp/repo"


class TestRepoInCategory:
    """Tests for _repo_in_category."""

    def test_string_in_list(self):
        """Finds string repos in list."""
        category = ["matsengrp/repo1", "matsengrp/repo2"]
        assert _repo_in_category("matsengrp/repo1", category)
        assert not _repo_in_category("matsengrp/repo3", category)

    def test_dict_in_list(self):
        """Finds dict repos in list."""
        category = [
            {"repo": "matsengrp/repo1", "channel": "dasm2"},
            {"repo": "matsengrp/repo2"},
        ]
        assert _repo_in_category("matsengrp/repo1", category)
        assert _repo_in_category("matsengrp/repo2", category)
        assert not _repo_in_category("matsengrp/repo3", category)

    def test_mixed_list(self):
        """Finds repos in mixed string/dict list."""
        category = [
            "matsengrp/repo1",
            {"repo": "matsengrp/repo2", "channel": "dasm2"},
        ]
        assert _repo_in_category("matsengrp/repo1", category)
        assert _repo_in_category("matsengrp/repo2", category)


class TestLoadReposByChannel:
    """Tests for load_repos_by_channel."""

    @pytest.fixture
    def mock_sources(self, tmp_path):
        """Create a mock sources.json file."""
        sources = {
            "boards": {},
            "context": {},
            "code": [
                {"repo": "matsengrp/repo1", "channel": "dasm2"},
                {"repo": "matsengrp/repo2", "channel": "dasm2"},
                {"repo": "matsengrp/repo3", "channel": "loris"},
                {"repo": "matsengrp/repo4"},  # No channel
            ],
            "writing": ["matsengrp/tex-repo"],
        }
        sources_file = tmp_path / "sources.json"
        sources_file.write_text(json.dumps(sources))
        return sources_file

    def test_loads_repos_for_channel(self, mock_sources):
        """Returns repos for the specified channel."""
        with patch("flowc.shared.config.SOURCES_FILE", mock_sources):
            repos = load_repos_by_channel("dasm2")
            assert repos == ["matsengrp/repo1", "matsengrp/repo2"]

    def test_returns_empty_for_unknown_channel(self, mock_sources):
        """Returns empty list for unknown channel."""
        with patch("flowc.shared.config.SOURCES_FILE", mock_sources):
            repos = load_repos_by_channel("unknown")
            assert repos == []

    def test_ignores_string_repos(self, mock_sources):
        """String repos (no channel) are not returned."""
        with patch("flowc.shared.config.SOURCES_FILE", mock_sources):
            # writing category has only strings
            repos = load_repos_by_channel("dasm2")
            assert "matsengrp/tex-repo" not in repos


class TestListChannels:
    """Tests for list_channels."""

    @pytest.fixture
    def mock_sources(self, tmp_path):
        """Create a mock sources.json file."""
        sources = {
            "boards": {},
            "context": {},
            "code": [
                {"repo": "matsengrp/repo1", "channel": "dasm2"},
                {"repo": "matsengrp/repo2", "channel": "loris"},
                {"repo": "matsengrp/repo3", "channel": "dasm2"},  # Duplicate
                {"repo": "matsengrp/repo4"},  # No channel
            ],
            "writing": ["matsengrp/tex-repo"],
        }
        sources_file = tmp_path / "sources.json"
        sources_file.write_text(json.dumps(sources))
        return sources_file

    def test_lists_unique_channels(self, mock_sources):
        """Returns sorted unique channels."""
        with patch("flowc.shared.config.SOURCES_FILE", mock_sources):
            channels = list_channels()
            assert channels == ["dasm2", "loris"]

    def test_empty_when_no_channels(self, tmp_path):
        """Returns empty list when no channels defined."""
        sources = {
            "boards": {},
            "code": ["matsengrp/repo1", "matsengrp/repo2"],
        }
        sources_file = tmp_path / "sources.json"
        sources_file.write_text(json.dumps(sources))

        with patch("flowc.shared.config.SOURCES_FILE", sources_file):
            channels = list_channels()
            assert channels == []


class TestLoadRepos:
    """Tests for load_repos with mixed format."""

    @pytest.fixture
    def mock_sources(self, tmp_path):
        """Create a mock sources.json file."""
        sources = {
            "boards": {},
            "context": {},
            "code": [
                {"repo": "matsengrp/repo1", "channel": "dasm2"},
                {"repo": "matsengrp/repo2"},
                "matsengrp/repo3",  # String format still works
            ],
            "writing": ["matsengrp/tex-repo"],
        }
        sources_file = tmp_path / "sources.json"
        sources_file.write_text(json.dumps(sources))
        return sources_file

    def test_loads_all_repos(self, mock_sources):
        """Loads all repos regardless of format."""
        with patch("flowc.shared.config.SOURCES_FILE", mock_sources):
            repos = load_repos()
            assert "matsengrp/repo1" in repos
            assert "matsengrp/repo2" in repos
            assert "matsengrp/repo3" in repos
            assert "matsengrp/tex-repo" in repos
