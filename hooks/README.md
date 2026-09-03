# Hooks

Claude Code hooks are shell commands the harness runs by itself at fixed
moments. `make install` symlinks the scripts here into `~/.claude/hooks/`,
but a hook only runs once it is named in `~/.claude/settings.json`, and that
file is user configuration the Makefile does not touch. Add the entries
below by hand.

## What these hooks are for

One thing gets one name. An agent that calls something `requeues` and then
`rescue` a message later, or writes `crossings` for what the user has been
calling shared molecules, forces the reader to work out that two words mean
one thing. Stating the rule in `CLAUDE.md` does not hold it: the instruction
is read once, at the start of a session, and a session runs for hundreds of
thousands of tokens after that.

These three hooks restate the rule where it cannot decay, and stop a body
reaching GitHub until it has been read by
[`terminology-checker`](../agents/terminology-checker.md).

## `terminology.sh`

`UserPromptSubmit`. Prints the rule and the failures worth recognising. The
output is added to the context on every turn, so the rule sits at the end of
the window rather than at the start.

```json
"UserPromptSubmit": [
  { "matcher": "",
    "hooks": [ { "type": "command", "command": "~/.claude/hooks/terminology.sh" } ] }
]
```

## `gh-termcheck.py` and `termcheck-stamp`

`PreToolUse` on `Bash`. Exit 2 blocks the command and shows the message to
the agent, which makes this the one point in the system where something can
actually be stopped.

```json
"PreToolUse": [
  { "matcher": "Bash",
    "hooks": [ { "type": "command", "command": "~/.claude/hooks/gh-termcheck.py" } ] }
]
```

It refuses these until the body has been stamped:

```
gh issue|pr comment|create|edit|review  --body-file FILE
gh api ... -F body=@FILE                (or `cat FILE | ... -F body=@-`)
gh api ... --input FILE
```

Calls with no body pass straight through, so `gh api` reads and
`gh pr edit --add-label` are untouched.

Bodies written inline (`--body "..."`, `-f body=...`) are refused outright.
Recovering body text from an arbitrary shell command means handling quoting,
heredocs, and `@-` fed by an upstream `cat`; requiring a file means the hook
only ever hashes a file, and the checker has something real to read.

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

The agent writes its own stamps, so the hook enforces that an agent stops
and looks rather than that it looked properly. Having the hook run a
separate headless Claude on the body instead would not be game-able, but
that Claude sees only the body — it would find a document calling one thing
by two names, and miss a name that differs from the one the user has been
using all along, which is the larger half of the problem.

Posts made through `curl`, or through a script calling the API directly, are
not covered.
