#!/usr/bin/env python3
"""Hold `gh pr create` until the body has been through /bip-pr-file.

Runs as a PreToolUse hook on Bash. Exit 2 blocks the call and shows the message
to Claude, which can load the skill and try again.

It checks that the rules were loaded, not that the body is good. Every
unreadable body so far was written by a session with no PR-body rules in front
of it, so catching that case is the whole of the enforcement: a skill nobody
opens at the moment it is needed changes nothing. What it cannot do is grade
prose -- sections that read well and never say what the diff does are exactly
what no regex sees -- so a pass here must not be read as a body that is fine.

A gate is worth having because a skill only fires when something invokes it. The
same argument as the terminology hooks in this directory: a rule stated in
CLAUDE.md is read once, at the start of a session, and a session runs for
hundreds of thousands of tokens after that. `gh pr create` is a precise moment;
CLAUDE.md is not.

The evidence it looks for is the Skill tool-use record in the transcript, not
the skill's name in prose -- this hook's own message names the skill, so a
looser match would pass on the retry without the skill ever being read.

Fires at most once per PR, and never in a session that used the skill, so it
costs nothing in the case it is not needed.
"""

import json
import os
import re
import sys

SKILL = "bip-pr-file"
# Overridable so the tests do not depend on what is installed on this machine.
SKILL_DIR = os.environ.get(
    "BIP_PR_FILE_SKILL_DIR", os.path.expanduser(f"~/.claude/skills/{SKILL}")
)

# The Skill tool's own record. Whitespace-tolerant because the transcript's
# JSON formatting is not ours to rely on.
LOADED = re.compile(
    r'"name"\s*:\s*"Skill"\s*,\s*"input"\s*:\s*\{\s*"skill"\s*:\s*"' + SKILL + '"'
)

# `gh pr create` files a body; `gh pr edit --body`/`--body-file` replaces one.
FILES_A_BODY = (
    re.compile(r"\bgh\s+pr\s+create\b"),
    re.compile(r"\bgh\s+pr\s+edit\b.*--body"),
)
# Reading about the command is not running it.
NOT_REALLY = re.compile(r"--help\b|\bgh\s+pr\s+create\s+--help")

MESSAGE = (
    f"This files a PR body and /{SKILL} has not been loaded this session. "
    f"Load it, write the body to a file, then file the PR. The body opens with "
    f"a short paragraph saying what the PR does and why, for a reviewer who is "
    f"not in the details; then one section per change, each opening with one "
    f"sentence naming what changed."
)


def files_a_pr_body(command: str) -> bool:
    """Whether this command posts or replaces a PR body."""
    if NOT_REALLY.search(command):
        return False
    return any(pattern.search(command) for pattern in FILES_A_BODY)


def skill_was_loaded(transcript_path: str) -> bool:
    """Whether the Skill tool was called with this skill in this session."""
    try:
        with open(transcript_path, errors="replace") as handle:
            return any(LOADED.search(line) for line in handle)
    except OSError:
        # An unreadable transcript is not evidence of anything, and blocking on
        # it would be blocking on our own failure.
        return True


def main() -> None:
    payload = json.load(sys.stdin)
    if payload.get("tool_name") != "Bash":
        sys.exit(0)
    if not files_a_pr_body(payload.get("tool_input", {}).get("command", "")):
        sys.exit(0)
    # With the skill not installed there is no remedy to point at.
    if not os.path.isdir(SKILL_DIR):
        sys.exit(0)
    transcript = payload.get("transcript_path")
    if not transcript or skill_was_loaded(transcript):
        sys.exit(0)

    print(MESSAGE, file=sys.stderr)
    sys.exit(2)


if __name__ == "__main__":
    main()
