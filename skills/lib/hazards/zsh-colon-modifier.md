---
tags: [shell, measurement]
measured: 2026-09-20
---

# `$VAR:` in zsh is a modifier site, not string concatenation

**Scope: what a human types in an interactive terminal.** The agent Bash tool can be pinned to
bash via `CLAUDE_CODE_SHELL` (see `matsen/setup#1`, `#2`), which removes this from tool calls.
The login shell on these boxes stays zsh either way, so the trap survives the harness fix — it
just stops being a harness problem and stays a terminal one.

**Looks right:** `git show $c:path/to/file`, `scp $host:$dir`, any `$VAR:` followed by text.

**What happens:** zsh reads `:` after a parameter as an expansion modifier, applies it, and
appends whatever is left. Of 15 letters measured, **13 are modifiers and only `:s` is loud**.

This is worse than dropping a character, and reads as one only when the value has no slashes:

```
B=abc123       $B:tests/x  ->  abc123ests/x    looks like a typo
B=origin/main  $B:tests/x  ->  mainests/x      :t took the tail, then appended "ests/x"
B=a/b/c.txt    $B:tests/x  ->  c.txtests/x
```

`:a`, `:A`, `:P` prepend the working directory; `:h` gives a dirname; `:u` uppercases; `:e`
drops the value entirely. `:d`, `:p`, `:x` pass through.

**Check:** `"${VAR}:rest"` — brace the parameter, quote the argument. Unconditionally, not for
a remembered set: this entry first published a letter list and had two of them backwards, then
described the mechanism as "eats a character", which understated it in the benign direction.
A memorised list is the wrong instrument for a rule whose content is that the exceptions are
unguessable.

**Expect to underestimate it.** One letter aborts and gets fixed; twelve return an answer. You
remember the instances that interrupted you, not the ones that agreed with you. The same shape
elsewhere: an omitted or misparsed scope argument yields *somebody's* scope rather than an
error — `tmux kill-server` without `-L` killed every session on this box.

**Not part of this trap, though it co-occurred:** `cmd 2>/dev/null | grep -c` reporting `0` for
a failed command is POSIX pipeline behaviour, not zsh, and switching shells does not fix it.
Fixed at the source in `skills/lib/spawn-intent.sh` rather than guarded by a note.
