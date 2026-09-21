---
tags: [shell, measurement]
measured: 2026-09-21
---

# `cmd | wc -l` counts a failed command as zero

**Looks right:** `dirty=$(git -C "$d" status --porcelain | wc -l)`.

**What happens:** a pipeline's value comes from the last stage. If `git` fails — not a repo, path
gone, permissions — it writes nothing to stdout and `wc -l` faithfully reports `0`. "Clean tree"
and "could not look" are the same number. Adding `2>/dev/null` hides the only remaining evidence.
This is POSIX pipeline behaviour, not a zsh or ugrep quirk, and no shell choice fixes it.

**It is always fail-open, because zero is the reassuring answer.** Three instances in this repo,
all in code whose job is to protect uncommitted work:

| file | what `0` meant |
|---|---|
| `skills/lib/spawn-intent.sh` | clone reported CLEAN by the durability check |
| `skills/lib/fleet-collisions.sh` | dead session reported `uncommitted=0` — nothing to preserve |
| `skills/lib/clone-currency.sh` | clone marked **READY** and handed to a new worker |

All three sat next to careful code — `clone-currency.sh` guards `merge-base --is-ancestor` before
`rev-list --count` one line later, for exactly this reason.

**Fix:** capture first, then count, and give the failure its own value.

```sh
if out=$(git -C "$d" status --porcelain 2>/dev/null); then
  n=$(printf '%s' "$out" | grep -c . || true)
else
  n="UNREADABLE"          # must not compare equal to 0 downstream
fi
```

Check that the failure value fails *closed* where it is used. `"UNREADABLE"` works because
`[ "$n" = "0" ]` is false; a numeric sentinel like `-1` would silently pass a `-gt 0` test.

**Detector:**

```sh
/usr/bin/grep -rnE '\$\((git|gh) [^)]*\| *(wc -l|grep -c)' --include='*.sh' .
```

**Related but distinct:** `grep -q`/`-s` silences the *message*, not the *exit code* — a guard
ending a sourced file leaks that status as the shell's `$?`.
