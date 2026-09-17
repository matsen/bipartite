#!/usr/bin/env python3
"""Tests for pr-body.py.

Run with `python3 hooks/test_pr_body.py` or `make test-hooks`.

Each test runs the real hook against a small transcript. Exit 2 means it
blocked; exit 0 means it let the call through.
"""

import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HOOK = os.path.join(os.path.dirname(os.path.abspath(__file__)), "pr-body.py")
# The hook no-ops when the skill is absent, so the tests install a stand-in
# rather than depending on what this machine happens to have.
INSTALLED = tempfile.mkdtemp(prefix="bip-pr-file-")

SKILL_CALL = json.dumps(
    {
        "type": "assistant",
        "message": {
            "content": [
                {"type": "tool_use", "name": "Skill", "input": {"skill": "bip-pr-file"}}
            ]
        },
    }
)
OTHER_SKILL = SKILL_CALL.replace("bip-pr-file", "bip-pr-check")
# The hook's own message names the skill. A looser match would read this as
# evidence and pass on the retry.
MENTIONS_ONLY = json.dumps(
    {
        "type": "assistant",
        "message": {"content": [{"type": "text", "text": "I should use /bip-pr-file"}]},
    }
)


def run(command: str, transcript_lines: list, tool: str = "Bash") -> int:
    with tempfile.NamedTemporaryFile("w", suffix=".jsonl", delete=False) as handle:
        handle.write("\n".join(transcript_lines))
        path = handle.name
    try:
        payload = {
            "tool_name": tool,
            "tool_input": {"command": command},
            "transcript_path": path,
            "session_id": "test",
        }
        done = subprocess.run(
            [sys.executable, HOOK], input=json.dumps(payload),
            capture_output=True, text=True,
            env={**os.environ, "BIP_PR_FILE_SKILL_DIR": INSTALLED},
        )
        return done.returncode
    finally:
        os.unlink(path)


class TestBlocks(unittest.TestCase):
    def test_gh_pr_create_without_the_skill_is_blocked(self) -> None:
        assert run('gh pr create --title "x" --body "y"', [MENTIONS_ONLY]) == 2

    def test_gh_pr_edit_body_without_the_skill_is_blocked(self) -> None:
        assert run("gh pr edit 12 --body \"$(cat b.md)\"", []) == 2

    def test_body_file_counts_too(self) -> None:
        assert run("gh pr edit 12 --body-file b.md", []) == 2

    def test_a_bare_mention_is_not_evidence(self) -> None:
        """The hook's own message names the skill, so prose must not count."""
        assert run("gh pr create", [MENTIONS_ONLY]) == 2

    def test_a_different_skill_is_not_evidence(self) -> None:
        assert run("gh pr create", [OTHER_SKILL]) == 2


class TestAllows(unittest.TestCase):
    def test_the_skill_having_been_loaded_lets_it_through(self) -> None:
        assert run('gh pr create --body "$(cat b.md)"', [OTHER_SKILL, SKILL_CALL]) == 0

    def test_other_gh_pr_subcommands_are_not_touched(self) -> None:
        for command in ("gh pr view 12", "gh pr list", "gh pr merge --squash",
                        "gh pr checks 12", "gh pr edit 12 --add-label bug"):
            assert run(command, []) == 0, command

    def test_reading_the_help_is_not_filing(self) -> None:
        assert run("gh pr create --help", []) == 0

    def test_a_non_bash_tool_is_not_touched(self) -> None:
        assert run("gh pr create", [], tool="Write") == 0

    def test_an_unreadable_transcript_does_not_block(self) -> None:
        """Blocking on our own failure to read would be the wrong default."""
        payload = {
            "tool_name": "Bash",
            "tool_input": {"command": "gh pr create"},
            "transcript_path": "/nonexistent/transcript.jsonl",
        }
        done = subprocess.run(
            [sys.executable, HOOK], input=json.dumps(payload),
            capture_output=True, text=True,
            env={**os.environ, "BIP_PR_FILE_SKILL_DIR": INSTALLED},
        )
        assert done.returncode == 0, done.stderr


class TestNoSkillInstalled(unittest.TestCase):
    def test_nothing_is_blocked_when_there_is_no_remedy_to_point_at(self) -> None:
        """Blocking with the skill uninstalled leaves nowhere to go."""
        payload = {
            "tool_name": "Bash",
            "tool_input": {"command": "gh pr create"},
            "transcript_path": __file__,
        }
        done = subprocess.run(
            [sys.executable, HOOK], input=json.dumps(payload),
            capture_output=True, text=True,
            env={**os.environ, "BIP_PR_FILE_SKILL_DIR": "/nonexistent/skill"},
        )
        assert done.returncode == 0, done.stderr


if __name__ == "__main__":
    try:
        unittest.main(verbosity=2)
    finally:
        shutil.rmtree(INSTALLED, ignore_errors=True)
