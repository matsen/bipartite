#!/usr/bin/env python3
"""Report words in the turn just finished that have not been seen this session.

Runs as a Stop hook. Exit 2 blocks the turn from ending and shows the message
to Claude, which then has to answer for the words before it can stop.

The point is that a substituted name is, by construction, new to the session
while the name it replaces is old. So the words worth looking at can be found
with no model at all: keep every word anyone has used, and report the ones in
this turn that are not among them. A dictionary is no use here -- `rescue`,
`crossings` and `residue` are all ordinary English, and that is exactly why
they slip past.

A word is only reported once it is used at least twice in the same turn, on
the theory that a word being used as a name gets used more than once while a
word in passing does not. Measured over 3,622 turns in three real sessions:
as configured it fires on 3.1% of turns, median one word, against about a
third of turns if every new word were reported. That is a partial net -- of
two known renamings in those transcripts it catches one -- and it is the
version cheap enough to be worth blocking on.

Vocabulary is cached per session under ~/.claude/termcheck/, with the byte
offset already read, so each run only parses what is new. Firings are logged
to the same directory, and caches for sessions that have stopped are pruned.
"""

import collections
import datetime
import json
import os
import re
import sys

CACHE_DIR = os.path.expanduser("~/.claude/termcheck")
LOG = os.path.join(CACHE_DIR, "firings.jsonl")
PRUNE_AFTER_DAYS = 7
MIN_LENGTH = 4
MAX_LENGTH = 15
MAX_REPORTED = 12

# Spans whose words are not the agent's prose: code, paths, identifiers,
# anything quoted from elsewhere.
FENCED = re.compile(r"```.*?```", re.DOTALL)
INLINE_CODE = re.compile(r"`[^`]*`")
QUOTED_LINE = re.compile(r"^\s*>.*$", re.MULTILINE)
URL = re.compile(r"https?://\S+")
PATH = re.compile(r"[~/\w.-]*/[\w./-]+")
WORD = re.compile(r"[a-z]+")

VOWEL = re.compile(r"[aeiouy]")
RUN = re.compile(r"(.)\1\1")
SUFFIXES = ("s", "es", "ed", "ing", "ings", "d", "ly", "er", "ers")


def plausible(word: str) -> bool:
    """Could this be a word someone would write in a sentence?"""
    return (
        MIN_LENGTH <= len(word) <= MAX_LENGTH
        and VOWEL.search(word) is not None
        and RUN.search(word) is None
    )


def words_in(text: str) -> set[str]:
    return {word for word in WORD.findall(text.lower()) if plausible(word)}


def counted_words(text: str) -> collections.Counter:
    return collections.Counter(
        word for word in WORD.findall(text.lower()) if plausible(word)
    )


def repeated_words(text: str) -> set[str]:
    """Words used at least twice in `text`."""
    return {word for word, count in counted_words(text).items() if count >= 2}


def inflection_of_seen(word: str, vocabulary: set[str]) -> bool:
    """Is `word` just a plural or verb form of something already used?

    The first live firing of this hook was on "spawns", where "spawn" was
    already in the session. That is not a new name for anything.
    """
    for suffix in SUFFIXES:
        if word.endswith(suffix) and len(word) - len(suffix) >= MIN_LENGTH:
            base = word[: -len(suffix)]
            if base in vocabulary or base + "e" in vocabulary:
                return True
    return any(word + suffix in vocabulary for suffix in ("s", "es"))


def prose_only(text: str) -> str:
    """`text` with code, quotations, URLs and paths removed."""
    for pattern in (FENCED, INLINE_CODE, QUOTED_LINE, URL, PATH):
        text = pattern.sub(" ", text)
    return text


def split_entry(entry: dict) -> tuple[str, str, str]:
    """(assistant prose, other prose, tool text) for one transcript entry.

    Kept apart because they carry different authority. What the user wrote,
    and what this agent wrote earlier, are names in use. Tool output is
    mostly base64 and hex, and a single appearance in it establishes nothing.
    """
    kind = entry.get("type")
    content = entry.get("message", {}).get("content")
    if isinstance(content, str):
        return (content, "", "") if kind == "assistant" else ("", content, "")
    if not isinstance(content, list):
        return ("", "", "")
    spoken, other, tools = [], [], []
    for block in content:
        if not isinstance(block, dict):
            continue
        if block.get("type") == "text":
            (spoken if kind == "assistant" else other).append(block.get("text", ""))
        elif block.get("type") == "tool_use":
            tools.append(json.dumps(block.get("input", "")))
        elif block.get("type") == "tool_result":
            body = block.get("content", "")
            tools.append(body if isinstance(body, str) else json.dumps(body))
    return ("\n".join(spoken), "\n".join(other), "\n".join(tools))


def scan(path: str, offset: int) -> tuple[set[str], str, int]:
    """Words established from `offset` on, the last assistant prose, and the
    new offset.

    A word from tool output has to appear at least twice before it counts. A
    real name recurs -- a column heading appears in every row, a function at
    every call site -- while base64 fragments are unique by construction. On
    one 62 MB transcript this is the difference between a 934,000-word
    vocabulary and a 113,000-word one, with every real term surviving.
    """
    established: set[str] = set()
    tool_counts: collections.Counter = collections.Counter()
    latest = ""
    with open(path, "rb") as handle:
        handle.seek(offset)
        raw = handle.read()
        position = offset + len(raw)
    for line in raw.decode("utf-8", "replace").splitlines():
        if not line.strip():
            continue
        try:
            entry = json.loads(line)
        except ValueError:
            continue
        spoken, other, tools = split_entry(entry)
        if other:
            established |= words_in(other)
        if tools:
            tool_counts.update(counted_words(tools))
        if spoken:
            established |= words_in(latest)  # the previous turn is history now
            latest = spoken
    established |= {word for word, count in tool_counts.items() if count >= 2}
    return established, latest, position


def prune(current: str) -> None:
    """Drop caches for sessions that have not been written to in a while."""
    cutoff = datetime.datetime.now().timestamp() - PRUNE_AFTER_DAYS * 86400
    for name in os.listdir(CACHE_DIR):
        if not name.startswith("vocab-") or name == current:
            continue
        path = os.path.join(CACHE_DIR, name)
        try:
            if os.path.getmtime(path) < cutoff:
                os.remove(path)
        except OSError:
            pass


def main() -> None:
    payload = json.load(sys.stdin)
    if payload.get("stop_hook_active"):
        sys.exit(0)  # already answering for this turn; do not loop
    path = payload.get("transcript_path")
    if not path or not os.path.isfile(path):
        sys.exit(0)

    os.makedirs(CACHE_DIR, exist_ok=True)
    session = payload.get("session_id", "unknown")

    # Plain text rather than JSON: the vocabulary is the bulk of the file and
    # quoting every word costs more to write and parse than it is worth. First
    # line is the byte offset already read, the rest is one word per line.
    cache_name = f"vocab-{session}.txt"
    cache_path = os.path.join(CACHE_DIR, cache_name)
    vocabulary: set[str] = set()
    offset = 0
    if os.path.exists(cache_path):
        with open(cache_path) as handle:
            offset = int(handle.readline())
            vocabulary = set(handle.read().split("\n"))
        vocabulary.discard("")

    established, latest, position = scan(path, offset)
    vocabulary |= established
    novel = sorted(
        word for word in repeated_words(prose_only(latest)) - vocabulary
        if not inflection_of_seen(word, vocabulary)
    )
    vocabulary |= words_in(latest)

    with open(cache_path, "w") as handle:
        handle.write(f"{position}\n")
        handle.write("\n".join(sorted(vocabulary)))
    prune(cache_name)

    if not novel:
        sys.exit(0)

    with open(LOG, "a") as handle:
        handle.write(json.dumps({
            "when": datetime.datetime.now().isoformat(timespec="seconds"),
            "session": session,
            "cwd": payload.get("cwd", ""),
            "words": novel,
            "vocabulary": len(vocabulary),
        }) + "\n")

    shown = ", ".join(novel[:MAX_REPORTED])
    if len(novel) > MAX_REPORTED:
        shown += f", and {len(novel) - MAX_REPORTED} more"
    print(
        f"New words in that turn, not used by anyone earlier in this session:\n"
        f"  {shown}\n"
        f"For each one: does it name something that already has a different name "
        f"here?\nIf so, say so and use the existing name. If they are all naming "
        f"new things,\nor are ordinary English rather than names, stop -- nothing "
        f"needs saying.",
        file=sys.stderr,
    )
    sys.exit(2)


if __name__ == "__main__":
    main()
