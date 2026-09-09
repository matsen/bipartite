# Hooks

Claude Code hooks are shell commands the harness runs by itself at fixed
moments. `make install` copies the scripts here into `~/.claude/hooks/`, but
a hook only runs once it is named in `~/.claude/settings.json`, and that file
is user configuration the Makefile does not touch. Add the entries below by
hand.

They are copied rather than symlinked, unlike agents and skills. A symlink
into this repo dangles whenever another branch is checked out, and a hook
whose command is missing exits 127, which Claude Code treats as a
non-blocking error — the check would stop running with nothing said. Re-run
`make install` after editing a hook.

Tests: `make test-hooks`.

## What these hooks are for

One thing gets one name. An agent that calls something `requeues` and then
`rescue` a message later, or writes `crossings` for what the user has been
calling shared molecules, forces the reader to work out that two words mean
one thing. Stating the rule in `CLAUDE.md` does not hold it: the rule is read
once, at the start of a session, and a session runs for hundreds of thousands
of tokens after that.

Prose leaves an agent through chat, through `Write` and `Edit`, and through
Bash — a body handed to `gh`, a heredoc into a document. `novel-words.py`
checks all of it: the turn is everything the agent wrote since the last user
prompt, wherever it was going. Code is exempt, since an identifier is not a
sentence, so of the files only prose suffixes count (`.md`, `.markdown`,
`.txt`, `.tex`, `.rst`).

Checking the writes as well as the chat matters more than it sounds. Until it
did, a document went into the vocabulary as tool text, so a name invented on
its way into a file looked established the moment it was written, and the
chat could then use it freely.

The renaming rule is also part of
[`/bip-remove-metaspeak`](../skills/bip-remove-metaspeak),
[`/bip-issue-check`](../skills/bip-issue-check) and
[`/bip-ms-sweep`](../skills/bip-ms-sweep). The hook sees each write as it
happens and only in this session's terms; those passes read the finished
document. In one week the passes caught five renamings and the hook none,
because each replaced name was a word the session had already used —
`survivors` for what the document elsewhere called kept donors, `exclusion
table` for the sharing table.

## `novel-words.py`

`Stop`. Reports words in the turn just finished that nobody has used earlier
in the session, and blocks the turn from ending until they are answered for.

```json
"Stop": [
  { "matcher": "",
    "hooks": [ { "type": "command", "command": "~/.claude/hooks/novel-words.py" } ] }
]
```

A substituted name is by construction new to the session and the name it
replaces is old, so the candidates are computable with no model. A dictionary
is no help — `rescue`, `crossings` and `residue` are all ordinary English,
which is why they slip past.

A word is reported only if it is used at least twice in the turn, on the
theory that a word being used as a name gets used more than once. Plurals and
verb forms of words already used are ignored: the first live firing was on
`spawns`, with `spawn` already in the session, which is not a new name for
anything.

As configured it fires on **11.5%** of turns, measured over the 174 turns
since it went live, median one word. Reporting every word new to the session
instead, with no two-use rule, fires on about a third of turns — too often to
be read, which is the failure this exists to avoid. It is a partial net, and
so far an empty one: four firings in the log, all ordinary English, no
renaming caught.

Not counted as the agent's own prose, and so removed before the comparison:
code blocks, inline code, quoted lines, URLs, paths, maths, and the
`old_string` of an edit, which is quoted from the file rather than written
now.

A turn that is answering a report of its own cannot be blocked again —
Claude Code sets `stop_hook_active` so that a hook cannot loop — but it is
prose like any other, and it is where an agent defends the name it has just
coined. That turn is held in `~/.claude/termcheck/turn-<session>.txt` and
checked with the next one, a turn late rather than never.

## What counts as a word already in use

Anything you wrote, and anything this agent wrote in an earlier turn, counts
straight away — those are names in use.

Tool output is different. It is mostly base64 and hex, and taken at face
value it reached a 934,000-word vocabulary on one session, nearly all of it
junk that a vowel-and-no-tripled-letter filter waved through. So a word from
tool output has to appear **at least twice** before it counts. A real name
recurs — a column heading appears in every row, a function at every call site
— while base64 fragments are unique by construction. On the same session that
gives 114,000 words, with `crossings`, `residue`, `partis`, `conscount`,
`subagent` and `termcheck` all surviving.

Tool output has to count for something, or every term picked up from a file
would look new. A column called `crossings` makes "the crossings" the data's
own name, not an invention.

## Cache, log and cleanup

Vocabulary is cached per session under `~/.claude/termcheck/`, with the byte
offset already read, so each run parses only what is new: **0.10s** a turn on
a 62 MB transcript and a 774 KB cache, against 3.2s with no cache. The first
run in a resumed session has the whole transcript to read; it goes a line at
a time, which on a 307 MB one is 10s and 100 MB rather than 10s and 2.0 GB.

Every firing is appended to `~/.claude/termcheck/firings.jsonl` — time,
session, working directory, the words, and the vocabulary size — so the
question of whether this is worth running has an answer after a few weeks
rather than an impression. Silence is not logged.

Caches for sessions untouched for a week are removed on each run.

## `terminology.sh`

`UserPromptSubmit`. Restates the rule, so it sits at the end of the context
window on every turn rather than at the start of the session.

```json
"UserPromptSubmit": [
  { "matcher": "",
    "hooks": [ { "type": "command", "command": "~/.claude/hooks/terminology.sh" } ] }
]
```

It carries no example words. The text is an instruction to the agent reading
it, and a list of words reads as a list of banned words — an earlier version
listed `residue`, which is the correct domain term in one of the projects
this fires in.

This is the less proven of the two. On the day it was written it did not
prevent three renamings by the agent that wrote it. It is kept because it
costs almost nothing and it is the only part that acts while the text is
being written rather than after.
