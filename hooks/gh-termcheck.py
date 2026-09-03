#!/usr/bin/env python3
"""Refuse GitHub posts whose body has not been through the terminology-checker.

Runs as a PreToolUse hook on Bash. Exit 2 blocks the command and shows the
message to Claude.

Covers the ways prose reaches GitHub:

  gh issue|pr comment|create|edit|review   --body-file FILE
  gh pr merge                              --body-file FILE
  gh release create|edit                   --notes-file FILE
  gh api ... -F body=@FILE                 (or `cat FILE | ... -F body=@-`)
  gh api ... --input FILE
  gh api graphql                           with a body in the mutation

In every case the body must live in a file, so the check runs on real bytes
and the stamp is tied to them: edit the file after checking and the stamp
stops matching. Bodies written inline on the command line are refused.

Every body in the command is checked, not just the first. Anything this
script cannot parse is refused rather than allowed: a check that fails open
is worse than one that is occasionally annoying.

Reads through `gh api` are untouched -- a call with no body passes through.
"""

import hashlib
import json
import os
import re
import shlex
import sys

STAMP_DIR = os.path.expanduser("~/.claude/termcheck")

# Subcommands that can carry prose, with the flag each uses for a file and
# the flag each uses for inline text.
SUBCOMMANDS = (
    (re.compile(r"\bgh\s+(?:issue|pr)\s+(?:comment|create|edit|review|merge)\b"),
     ("--body-file", "-F"), ("--body", "-b")),
    (re.compile(r"\bgh\s+release\s+(?:create|edit)\b"),
     ("--notes-file", "-F"), ("--notes", "-n")),
)
API = re.compile(r"\bgh\s+api\b")
GRAPHQL = re.compile(r"\bgh\s+api\s+graphql\b")
FIELD_FLAGS = ("-f", "-F", "--field", "--raw-field")
HEREDOC = re.compile(r"<<-?\s*['\"]?(\w+)['\"]?")

INLINE_ADVICE = (
    "Blocked: this posts a body written inline. Put the body in a file, run\n"
    "@agent-terminology-checker on it, fix every renaming it reports, then:\n"
    "  ~/.claude/hooks/termcheck-stamp <absolute path>\n"
    "and pass the file instead of the inline text. Stamping and posting have\n"
    "to be separate commands -- this hook runs before any of the command does."
)


def without_heredocs(command: str) -> str:
    """`command` with the contents of any heredoc removed.

    A commit message or a README that talks about these subcommands would
    otherwise look like a command that posts one.
    """
    while True:
        found = HEREDOC.search(command)
        if not found:
            return command
        lines = command[found.end():].split("\n")
        kept: list[str] = []
        for position, line in enumerate(lines):
            if line.strip() == found.group(1):
                kept = lines[position + 1:]
                break
        command = command[:found.start()] + "\n".join(kept)


def refuse(message: str) -> None:
    print(message, file=sys.stderr)
    sys.exit(2)


def flag_value(words: list[str], flags: tuple[str, ...]) -> str | None:
    """Value of the first of `flags` present, as `--flag value` or `--flag=value`."""
    for position, word in enumerate(words):
        for flag in flags:
            if word == flag and position + 1 < len(words):
                return words[position + 1]
            if word.startswith(flag + "="):
                return word.split("=", 1)[1]
    return None


def has_flag(words: list[str], flags: tuple[str, ...]) -> bool:
    return any(word == flag or word.startswith(flag + "=") for word in words for flag in flags)


def piped_sources(words: list[str]) -> list[str]:
    """Paths in every `cat FILE` in the command."""
    found = []
    for position, word in enumerate(words):
        if word == "cat" and position + 1 < len(words):
            candidate = words[position + 1]
            if not candidate.startswith("-"):
                found.append(candidate)
    return found


def api_bodies(words: list[str]) -> list[tuple[str, str]]:
    """Every ('file', path) or ('inline', text) body in a gh api call."""
    found: list[tuple[str, str]] = []
    for position, word in enumerate(words):
        field = None
        if word in FIELD_FLAGS and position + 1 < len(words):
            field = words[position + 1]
        else:
            for flag in FIELD_FLAGS:
                if word.startswith(flag + "="):
                    field = word.split("=", 1)[1]
                    break
        if field is None or not field.startswith("body="):
            continue
        value = field.split("=", 1)[1]
        if value == "@-":
            sources = piped_sources(words)
            if len(sources) != 1:
                refuse(
                    "Blocked: the body comes from stdin and this command has "
                    f"{len(sources)} `cat` calls,\nso the hook cannot tell which "
                    "file feeds it. Use exactly one `cat FILE |`."
                )
            found.append(("file", sources[0]))
        elif value.startswith("@"):
            found.append(("file", value[1:]))
        else:
            found.append(("inline", value))

    given = flag_value(words, ("--input",))
    if given is not None:
        if given == "-":
            refuse(
                "Blocked: the body comes from stdin via --input -. Write the JSON to\n"
                "a file, stamp that file, and pass it with --input FILE."
            )
        found.append(("file", given))
    return found


def check_file(path: str) -> None:
    """Refuse unless `path` is an absolute path to a stamped file."""
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
    try:
        with open(path, "rb") as handle:
            digest = hashlib.sha256(handle.read()).hexdigest()
    except OSError as error:
        refuse(f"Blocked: cannot read body file {path}: {error}")
    if os.path.exists(os.path.join(STAMP_DIR, digest)):
        return
    refuse(
        f"Blocked: {path} has not been through the terminology-checker, or it\n"
        f"changed since it was. Run @agent-terminology-checker on it, fix every\n"
        f"renaming it reports, then, as a separate command:\n"
        f"  ~/.claude/hooks/termcheck-stamp {path}\n"
        f"and re-run this one."
    )


payload = json.load(sys.stdin)
if payload.get("tool_name") != "Bash":
    sys.exit(0)

command = without_heredocs(payload.get("tool_input", {}).get("command", ""))
matched = [entry for entry in SUBCOMMANDS if entry[0].search(command)]
is_api = bool(API.search(command))
if not matched and not is_api:
    sys.exit(0)

try:
    words = shlex.split(command)
except ValueError as error:
    refuse(
        f"Blocked: this command cannot be parsed ({error}), so the hook cannot\n"
        f"find the body in it. An apostrophe in a comment or a heredoc will do\n"
        f"this. Reword it, or write the body to a file and pass --body-file."
    )

bodies: list[tuple[str, str]] = []

for _, file_flags, inline_flags in matched:
    if has_flag(words, inline_flags):
        refuse(INLINE_ADVICE)
    if words and words[-1] in file_flags:
        refuse(
            "Blocked: a body-file flag is the last token, so the hook cannot see\n"
            "its value. Pass the path directly rather than through a pipe."
        )
    path = flag_value(words, file_flags)
    if path is not None:
        bodies.append(("file", path))

if is_api:
    if GRAPHQL.search(command) and re.search(r"\bbody\s*:", command):
        if not any(kind == "file" for kind, _ in api_bodies(words)):
            refuse(
                "Blocked: this GraphQL mutation carries a body inline. Put the body\n"
                "in a file, stamp it, and pass it as a variable:\n"
                "  gh api graphql -F body=@/abs/path.md -f query='... body: $body ...'"
            )
    bodies.extend(api_bodies(words))

if not bodies:
    sys.exit(0)  # a read, or a write with no prose in it

for kind, value in bodies:
    if kind == "inline":
        refuse(INLINE_ADVICE)
    check_file(value)
