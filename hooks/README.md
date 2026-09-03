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
and `Edit`, and Bash. About 3% of that is a body posted to GitHub, and it is
the least likely 3%: a PR body is written in one sitting with the whole thing
in view, while renaming is a drift across hours. So `novel-words.py`, which
watches the chat, is the one that covers where the problem happens, and
`gh-termcheck.py` covers the narrow path where the damage is public.

## `novel-words.py`

`Stop`. Reports words in the turn just finished that nobody has used earlier
in the session. Exit 2 blocks the turn from ending, so the agent has to
answer for the words before it can stop.

```json
"Stop": [
  { "matcher": "",
    "hooks": [ { "type": "command", "command": "~/.claude/hooks/novel-words.py" } ] }
]
```

A substituted name is by construction new to the session, and the name it
replaces is old, so the candidates are computable with no model. A dictionary
is no help — `rescue`, `crossings` and `residue` are all ordinary English,
which is why they slip past.

A word has to be used at least twice in the turn before it is reported, on
the theory that a word being used as a name gets used more than once.
Measured over 3,622 turns in three real sessions:

| rule | fires on |
|---|---|
| any word new to the session | 31.9% |
| a new word reused in a later turn | 29.7% |
| a new word used twice in one turn | **3.6%**, median one word |

The first two are too frequent to be read, which is the failure this whole
thing exists to avoid. The third is a partial net: of two known renamings in
those transcripts it catches one. Code blocks, inline code, quoted lines,
URLs and paths are stripped before the comparison, so identifiers and quoted
material do not count as the agent's own prose.

Vocabulary is cached per session under `~/.claude/termcheck/` with the byte
offset already read, so each run parses only what is new.

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
this fires in. Worked examples live in
[`terminology-checker`](../agents/terminology-checker.md) instead, which
reads someone else's text rather than writing its own.

This is the least proven of the three. On the day it was written it did not
prevent three renamings by the agent that wrote it.

## `gh-termcheck.py` and `termcheck-stamp`

`PreToolUse` on `Bash`. Exit 2 blocks the command and shows the message to
the agent.

```json
"PreToolUse": [
  { "matcher": "Bash",
    "hooks": [ { "type": "command", "command": "~/.claude/hooks/gh-termcheck.py" } ] }
]
```

It refuses these until the body has been stamped:

```
gh issue|pr comment|create|edit|review|merge  --body-file FILE
gh release create|edit                        --notes-file FILE
gh api ... -F body=@FILE                      (or `cat FILE | ... -F body=@-`)
gh api ... --input FILE
gh api graphql                                with a body in the mutation
```

Calls with no body pass straight through, so `gh api` reads and
`gh pr edit --add-label` are untouched. Every body in the command is checked,
not just the first.

Bodies written inline (`--body "..."`, `-f body=...`) are refused outright.
Recovering body text from an arbitrary shell command means handling quoting,
heredocs, and `@-` fed by an upstream `cat`; requiring a file means the hook
only ever hashes a file, and the checker has something real to read.
Anything the script cannot parse is refused rather than allowed — a check
that fails open is worse than one that is occasionally annoying, and an
apostrophe in a shell comment used to be enough to disable it silently.

The working sequence is:

```bash
# write the body to a file, then
# run @agent-terminology-checker on it and fix what it reports
~/.claude/hooks/termcheck-stamp /abs/path/body.md
# then, as a separate command:
gh issue comment 42 --body-file /abs/path/body.md
```

`termcheck-stamp` writes a file under `~/.claude/termcheck/` named for the
sha256 of the body's bytes. Editing the body afterwards changes the hash, so
the stamp stops matching and the post is refused again. Because the stamp is
keyed on content rather than path, copying or moving the body carries its
stamp with it.

Three mechanics follow from the hook running *before* the command, and cost
a round trip each if you meet them unprepared:

- **Stamping and posting cannot be chained.** `termcheck-stamp body.md && gh
  issue create --body-file body.md` is refused: at the moment the hook looks,
  the stamp does not exist yet. Same for creating the body file in the
  command that posts it. Separate commands.
- **The body path must be absolute.** The hook cannot follow a `cd` earlier
  in the command, so it cannot resolve a relative path.
- **The body must be somewhere `gh` can read it.** Under Claude Code's auto
  mode, a command that reaches the network runs with its own view of `/tmp`,
  so `gh` reports a body under `/tmp/$USER/` as missing even though `ls` and
  `head` read it in the same shell. Keep the body inside the repository
  working directory and delete it after posting.

## What this does not do

The agent writes its own stamps, so the hook enforces that an agent stops and
looks rather than that it looked properly. Having the hook run a separate
headless Claude on the body instead would not be game-able, but that Claude
sees only the body — it would find a document calling one thing by two names
and miss a name that differs from the one the user has been using all along,
which is the larger half of the problem. `terminology-checker` has the same
blindness, which is why it asks for a names brief and says so when it does
not get one.

Posts made through `curl`, or through a script calling the API directly, are
not covered.
