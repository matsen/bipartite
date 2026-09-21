---
tags: [shell, measurement]
measured: 2026-09-21
---

# A bare `grep` in a Claude Code Bash call skips gitignored files

**Scope: inside a Claude Code Bash call, on any machine.** Not workstation-specific, and not your
terminal — there `grep` is an ordinary alias to GNU grep. `type grep` distinguishes them.

**What happens:** the shell snapshot defines `grep` as a function routing to an embedded `ugrep`
with `-G --ignore-files --hidden -I --exclude-dir=.git …`. `--ignore-files` honours `.gitignore`,
so a recursive search returns a **confident partial result**:

```
echo NEEDLE > junk/hit.txt      # junk/ is gitignored
grep -rl NEEDLE .            ->  (nothing)
/usr/bin/grep -rl NEEDLE .   ->  ./junk/hit.txt
```

**Why it matters here specifically, and the conjunction is the whole failure:** this project's
convention puts continuation notes in `_ignore/CONTINUE.md`, and the three `CONTINUATION*.md`
files holding the fleet's highest-firing guidance are gitignored. A gitignored home for
load-bearing notes plus a grep that skips gitignored paths means **searching them returns empty
and reads as "absent."** Neither half is remarkable alone.

**Check:** `/usr/bin/grep` for any absence claim, any alternation, and any search that might
reach an ignored path. `command grep` also bypasses it, as do `-z`, `-Z`, `--null` and
`--filter`, which the shim detects and passes through.

**Other differences from GNU grep in the same shim:** `-G` is basic-regex mode, which is the
mechanism behind the alternation truncation pinned in `/bip-pr-land`; `-I` skips binary files
silently where GNU would print "Binary file matches"; a mid-pattern `$` is an anchor, not a
literal (see `EVIDENCE-DISCIPLINE.md`, fourth costume).

**`find` is shadowed too, but faithfully** — `bfs` with `-S dfs -regextype findutils-default`.
Measured: identical output to `/usr/bin/find`, gitignored files included. It is a speed
substitution, not a semantic one. The consequence is elsewhere: a bare `find` on `pax` *is*
`bfs`, which is the tool the global `CLAUDE.md` records walking `/fh/fast` for 25 hours.

**Not shell-scoped, so pinning bash does not remove it.** The shadow is injected by the harness,
not inherited from shell config, and the binary carries a second snapshot generator on the bash
path (evidence is the binary, not an observed bash snapshot). `CLAUDE_CODE_SHELL` fixes the
zsh-parsing class in `zsh-colon-modifier.md` and leaves this one untouched. Related but also not
shell-scoped: `cmd 2>/dev/null | grep -c` reporting `0` for a failed command is POSIX pipeline
behaviour.
