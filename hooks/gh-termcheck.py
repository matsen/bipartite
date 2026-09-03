#!/usr/bin/env python3
"""Refuse GitHub posts whose body has not been through the terminology-checker.

Runs as a PreToolUse hook on Bash. Exit 2 blocks the command and shows the
message to Claude.

Covers both ways prose reaches GitHub:

  gh issue|pr comment|create|edit|review   --body-file FILE
  gh api ... -F body=@FILE                 (or `cat FILE | ... -F body=@-`)
  gh api ... --input FILE

In every case the body must live in a file, so the check runs on real bytes
and the stamp is tied to them: edit the file after checking and the stamp
stops matching. Bodies written inline on the command line are refused.

Reads through `gh api` are untouched -- a call with no body passes straight
through.
"""

import hashlib
import json
import os
import re
import shlex
import sys

STAMP_DIR = os.path.expanduser("~/.claude/termcheck")
SUBCOMMAND = re.compile(r"\bgh\s+(issue|pr)\s+(comment|create|edit|review)\b")
API = re.compile(r"\bgh\s+api\b")
FIELD_FLAGS = ("-f", "-F", "--field", "--raw-field")

INLINE_ADVICE = (
    "Blocked: this posts a body written inline. Put the body in a file, run\n"
    "@agent-terminology-checker on it, fix every renaming it reports, then:\n"
    "  ~/.claude/hooks/termcheck-stamp <file>\n"
    "and pass the file instead of the inline text."
)


def refuse(message: str) -> None:
    print(message, file=sys.stderr)
    sys.exit(2)


def flag_value(words: list[str], flags: tuple[str, ...]) -> str | None:
    """Value of the first of `flags` present, as `--flag value` or `--flag=value`."""
    for flag in flags:
        if flag in words:
            position = words.index(flag)
            if position + 1 < len(words):
                return words[position + 1]
        for word in words:
            if word.startswith(flag + "="):
                return word.split("=", 1)[1]
    return None


def has_flag(words: list[str], flags: tuple[str, ...]) -> bool:
    return any(word == flag or word.startswith(flag + "=") for word in words for flag in flags)


def piped_source(words: list[str]) -> str | None:
    """Path in a `cat FILE |` that feeds this command's stdin."""
    for position, word in enumerate(words):
        if word == "cat" and position + 1 < len(words):
            candidate = words[position + 1]
            if not candidate.startswith("-"):
                return candidate
    return None


def api_body(words: list[str]) -> tuple[str, str] | None:
    """('file', path) or ('inline', text) for a gh api body, or None if no body."""
    for position, word in enumerate(words):
        if word in FIELD_FLAGS and position + 1 < len(words):
            field = words[position + 1]
            if not field.startswith("body="):
                continue
            value = field.split("=", 1)[1]
            if value == "@-":
                source = piped_source(words)
                if source is None:
                    refuse(
                        "Blocked: body comes from stdin and the hook cannot tell which\n"
                        "file. Use `cat FILE | ... -F body=@-`, having stamped FILE."
                    )
                return ("file", source)
            if value.startswith("@"):
                return ("file", value[1:])
            return ("inline", value)

    given = flag_value(words, ("--input",))
    if given is not None:
        if given == "-":
            refuse(
                "Blocked: body comes from stdin via --input -. Write the JSON to a\n"
                "file, stamp that file, and pass it with --input FILE."
            )
        return ("file", given)
    return None


def check_file(path: str) -> None:
    path = os.path.expanduser(os.path.expandvars(path))
    if not os.path.isabs(path):
        refuse(
            f"Blocked: body file {path} is a relative path. The hook runs before\n"
            f"the command does, so it cannot follow a `cd` and cannot resolve it.\n"
            f"Pass an absolute path."
        )
    if not os.path.isfile(path):
        refuse(
            f"Blocked: body file {path} does not exist. If the same command is\n"
            f"about to create it, write the file first -- the hook runs before any\n"
            f"of the command does."
        )
    with open(path, "rb") as handle:
        digest = hashlib.sha256(handle.read()).hexdigest()
    if os.path.exists(os.path.join(STAMP_DIR, digest)):
        sys.exit(0)
    refuse(
        f"Blocked: {path} has not been through the terminology-checker, or it\n"
        f"changed since it was. Run @agent-terminology-checker on it, fix every\n"
        f"renaming it reports, then:\n"
        f"  ~/.claude/hooks/termcheck-stamp {path}\n"
        f"and re-run this command."
    )


payload = json.load(sys.stdin)
if payload.get("tool_name") != "Bash":
    sys.exit(0)

command = payload.get("tool_input", {}).get("command", "")
is_subcommand = bool(SUBCOMMAND.search(command))
is_api = bool(API.search(command))
if not is_subcommand and not is_api:
    sys.exit(0)

try:
    words = shlex.split(command)
except ValueError:
    sys.exit(0)

if is_subcommand:
    if has_flag(words, ("--body", "-b")):
        refuse(INLINE_ADVICE)
    path = flag_value(words, ("--body-file", "-F"))
    if path is None:
        sys.exit(0)  # no body supplied, e.g. `gh pr edit --add-label`
    check_file(path)

body = api_body(words)
if body is None:
    sys.exit(0)  # a read, or a write with no prose in it
kind, value = body
if kind == "inline":
    refuse(INLINE_ADVICE)
check_file(value)
