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

The turn is everything the agent wrote since the last user prompt, wherever it
was going: chat, a document written with Write or Edit, a body posted with
`gh`. All of it is checked and all of it joins the vocabulary. Code is not
prose, so of the files only prose suffixes count.

A word is only reported once it is used at least twice in the same turn, on
the theory that a word being used as a name gets used more than once while a
word in passing does not. Over the 174 turns since it went live it fires on
11.5% of them, median one word, against about a third of turns if every new
word were reported. It has not yet caught a renaming: the four firings in the
log are ordinary English, and the renamings a document pass found over the
same days were all words the session had already used.

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

# Files whose contents are sentences rather than identifiers.
PROSE_SUFFIXES = (".md", ".markdown", ".txt", ".tex", ".rst")

# Spans whose words are not the agent's prose: code, markup, paths,
# identifiers, anything quoted from elsewhere.
FENCED = re.compile(r"```.*?```", re.DOTALL)
INLINE_CODE = re.compile(r"`[^`]*`")
QUOTED_LINE = re.compile(r"^\s*>.*$", re.MULTILINE)
URL = re.compile(r"https?://\S+")
PATH = re.compile(r"[~/\w.-]*/[\w./-]+")
# LaTeX. The hook's first firing in another session was on "qquad", from a
# \qquad inside display math. Maths is code that happens not to sit in a
# fenced block.
DISPLAY_MATH = re.compile(r"\$\$.*?\$\$|\\\[.*?\\\]", re.DOTALL)
INLINE_MATH = re.compile(r"\$[^$\n]+\$|\\\(.*?\\\)", re.DOTALL)
CONTROL_SEQUENCE = re.compile(r"\\[a-zA-Z]+")
HTML_TAG = re.compile(r"</?[a-zA-Z][^>]*>")
WORD = re.compile(r"[a-z]+")

# Prose written through a shell command: a heredoc into a document, or a body
# handed to `gh`.
HEREDOC = re.compile(r"<<-?\s*['\"]?(\w+)['\"]?\s*\n(.*?)\n\s*\1\b", re.DOTALL)
GH_BODY = re.compile(r"--(?:body|title)[= ]\s*(['\"])(.*?)\1", re.DOTALL)
PROSE_REDIRECT = re.compile(
    r">>?\s*\S+\.(?:%s)\b" % "|".join(suffix.lstrip(".") for suffix in PROSE_SUFFIXES)
)

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
    """`text` with code, markup, quotations, URLs and paths removed."""
    for pattern in (FENCED, INLINE_CODE, QUOTED_LINE, DISPLAY_MATH,
                    INLINE_MATH, CONTROL_SEQUENCE, HTML_TAG, URL, PATH):
        text = pattern.sub(" ", text)
    return text


def shell_prose(command: str) -> str:
    """Prose in a shell command: a heredoc into a document, or a `gh` body."""
    if " gh " not in f" {command}" and not PROSE_REDIRECT.search(command):
        return ""
    written = [body for _, body in HEREDOC.findall(command)]
    written += [body for _, body in GH_BODY.findall(command)]
    return "\n".join(written)


def written_prose(name: str, tool_input: dict) -> str:
    """Prose the agent composed inside a tool call.

    A name swapped on its way into a document is the same defect as one
    swapped in a sentence, and it used to be worse than unchecked: the
    document went into the vocabulary as tool text, so the name looked
    established the moment it was invented. Code is exempt -- an identifier is
    not a sentence, and checking one fires on every new symbol.
    """
    if not isinstance(tool_input, dict):
        return ""
    path = str(tool_input.get("file_path", ""))
    if name == "Write" and path.endswith(PROSE_SUFFIXES):
        return str(tool_input.get("content", ""))
    if name == "Edit" and path.endswith(PROSE_SUFFIXES):
        return str(tool_input.get("new_string", ""))  # old_string is quoted, not written
    if name == "NotebookEdit" and tool_input.get("cell_type") == "markdown":
        return str(tool_input.get("new_source", ""))
    if name == "Bash":
        return shell_prose(str(tool_input.get("command", "")))
    return ""


def split_entry(entry: dict) -> tuple[str, str, str]:
    """(prose the agent wrote, prose anyone else wrote, tool text) for one entry.

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
    authored, other, tools = [], [], []
    for block in content:
        if not isinstance(block, dict):
            continue
        if block.get("type") == "text":
            (authored if kind == "assistant" else other).append(block.get("text", ""))
        elif block.get("type") == "tool_use":
            tool_input = block.get("input", "")
            written = written_prose(block.get("name", ""), tool_input)
            # A call whose prose is checked must not also count as tool text,
            # or its own words would establish themselves in the same turn.
            if written:
                authored.append(written)
            else:
                tools.append(json.dumps(tool_input))
        elif block.get("type") == "tool_result":
            body = block.get("content", "")
            tools.append(body if isinstance(body, str) else json.dumps(body))
    return ("\n".join(authored), "\n".join(other), "\n".join(tools))


def is_prompt(entry: dict) -> bool:
    """Does this entry begin a new turn? A message from the user does."""
    if entry.get("type") != "user":
        return False
    content = entry.get("message", {}).get("content")
    if isinstance(content, str):
        return True
    if isinstance(content, list):
        return any(
            isinstance(block, dict) and block.get("type") == "text" for block in content
        )
    return False


def scan(path: str, offset: int) -> tuple[set[str], str, int]:
    """Words established from `offset` on, the last turn's prose, and the new
    offset.

    A word from tool output has to appear at least twice before it counts. A
    real name recurs -- a column heading appears in every row, a function at
    every call site -- while base64 fragments are unique by construction. On
    one 62 MB transcript this is the difference between a 934,000-word
    vocabulary and a 113,000-word one, with every real term surviving.

    Read a line at a time. A resumed session hands the first run a whole
    transcript, and reading a 307 MB one as a single string cost 2 GB.
    """
    established: set[str] = set()
    tool_counts: collections.Counter = collections.Counter()
    turn: list[str] = []
    with open(path, "rb") as handle:
        handle.seek(offset)
        for raw in handle:
            if not raw.endswith(b"\n"):
                break  # half-written line; leave it for the next run
            offset += len(raw)
            line = raw.decode("utf-8", "replace")
            if not line.strip():
                continue
            try:
                entry = json.loads(line)
            except ValueError:
                continue
            if is_prompt(entry):
                established |= words_in("\n".join(turn))  # earlier turns are history
                turn = []
            authored, other, tools = split_entry(entry)
            if authored:
                turn.append(authored)
            if other:
                established |= words_in(other)
            if tools:
                tool_counts.update(counted_words(tools))
    established |= {word for word, count in tool_counts.items() if count >= 2}
    return established, "\n".join(turn), offset


def read_cache(path: str) -> tuple[set[str], int]:
    """Vocabulary and byte offset from a previous run.

    Plain text rather than JSON: the vocabulary is the bulk of the file and
    quoting every word costs more to write and parse than it is worth. First
    line is the byte offset already read, the rest is one word per line.
    """
    if not os.path.exists(path):
        return set(), 0
    with open(path) as handle:
        offset = int(handle.readline())
        vocabulary = set(handle.read().split("\n"))
    vocabulary.discard("")
    return vocabulary, offset


def write_cache(path: str, offset: int, vocabulary: set[str]) -> None:
    with open(path, "w") as handle:
        handle.write(f"{offset}\n")
        handle.write("\n".join(sorted(vocabulary)))


def prune(current: str) -> None:
    """Drop state for sessions that have not been written to in a while."""
    cutoff = datetime.datetime.now().timestamp() - PRUNE_AFTER_DAYS * 86400
    for name in os.listdir(CACHE_DIR):
        if not name.startswith(("vocab-", "turn-")) or name == current:
            continue
        path = os.path.join(CACHE_DIR, name)
        try:
            if os.path.getmtime(path) < cutoff:
                os.remove(path)
        except OSError:
            pass


def main() -> None:
    payload = json.load(sys.stdin)
    path = payload.get("transcript_path")
    if not path or not os.path.isfile(path):
        sys.exit(0)

    os.makedirs(CACHE_DIR, exist_ok=True)
    session = payload.get("session_id", "unknown")
    cache_name = f"vocab-{session}.txt"
    cache_path = os.path.join(CACHE_DIR, cache_name)
    held_path = os.path.join(CACHE_DIR, f"turn-{session}.txt")

    vocabulary, offset = read_cache(cache_path)
    established, turn, position = scan(path, offset)
    vocabulary |= established

    # A turn held from a previous run is checked here, with this one.
    if os.path.exists(held_path):
        with open(held_path) as handle:
            turn = f"{handle.read()}\n{turn}"
        os.remove(held_path)

    # A turn answering a report of its own cannot be blocked again -- Claude
    # Code sets stop_hook_active so that a hook cannot loop -- but it is prose
    # like any other, and it is where an agent defends a name it just coined.
    # Hold it for the next turn, where blocking is allowed again, and keep its
    # words out of the vocabulary until they have been looked at.
    if payload.get("stop_hook_active"):
        with open(held_path, "w") as handle:
            handle.write(turn)
        write_cache(cache_path, position, vocabulary)
        prune(cache_name)
        sys.exit(0)

    novel = sorted(
        word for word in repeated_words(prose_only(turn)) - vocabulary
        if not inflection_of_seen(word, vocabulary)
    )
    vocabulary |= words_in(turn)
    write_cache(cache_path, position, vocabulary)
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
