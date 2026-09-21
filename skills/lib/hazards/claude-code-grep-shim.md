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

**Not shell-scoped. `CLAUDE_CODE_SHELL=bash` does not remove it** — the shadow is injected by the
harness, not inherited from shell config. Verified by reading an actual `snapshot-bash-*.sh`
produced after the pin landed: it carries the identical `ugrep -G --ignore-files …` and
`bfs -S dfs …` definitions. Do not read "the Bash tool now runs bash" as "the shim was a zsh
artifact" and retire the `/usr/bin/grep` habit.

| hazard | fixed by pinning bash? |
|---|---|
| zsh `$VAR:` colon modifier (`zsh-colon-modifier.md`) | yes — for sessions started after |
| zsh not word-splitting an unquoted `$VAR` | yes — same |
| `grep` → ugrep, `--ignore-files` skipping gitignored | **no — shell-independent** |
| `find` → bfs | **no — shell-independent** |
| `cmd \| wc -l` counting a failed command as 0 (`counting-a-failed-command.md`) | **no — POSIX** |

**And the fix reaches new sessions only**, because a snapshot is captured at session start. A
session running when the pin landed keeps its old shell until it restarts — measured, an
already-running session still truncated `$c:experiments/x` to `xperiments/x` afterwards.
