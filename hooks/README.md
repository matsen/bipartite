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

Prose leaves an agent roughly ten thousand times a session — chat, `Write`
and `Edit`, and Bash. Bodies posted to GitHub are about 3% of that, and the
least likely 3%: such a body is written in one sitting with the whole thing
in view, while renaming is a drift across hours. So the check belongs on the
chat, which is what `novel-words.py` watches. For documents, the renaming
rule is part of [`/bip-remove-metaspeak`](../skills/bip-remove-metaspeak),
[`/bip-issue-check`](../skills/bip-issue-check) and
[`/bip-ms-sweep`](../skills/bip-ms-sweep), where a pass over the text is
already happening.

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
theory that a word being used as a name gets used more than once. Measured
over 3,622 turns in three real sessions:

| rule | fires on |
|---|---|
| any word new to the session | 31.9% |
| a new word reused in a later turn | 29.7% |
| a new word used twice in one turn | 4.8% |
| the same, ignoring plurals and verb forms of words already seen | **3.8%** |

The first two are too frequent to be read, which is the failure this exists
to avoid. The last is what runs. It is a partial net: of two known renamings
in those transcripts it catches one.

Not counted as the agent's own prose, and so excluded before the comparison:
code blocks, inline code, quoted lines, URLs and paths. Nor are tokens that
could not be words — tool output is full of base64 and hex, which reached 1.3
million vocabulary entries on one long session, so a token needs a vowel, no
letter repeated three times over, and at most 15 characters.

Inflections are dropped too. The first live firing was on `spawns`, with
`spawn` already in the session, which is not a new name for anything.

Vocabulary is cached per session under `~/.claude/termcheck/` with the byte
offset already read, so each run parses only what is new: 0.8s a turn on a
62 MB transcript, against 3.3s with no cache.

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
