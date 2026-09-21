---
tags: [shell, measurement]
measured: 2026-09-20
---

# `$VAR:` in zsh is a modifier site, not string concatenation

**Looks right:** `git show $c:path/to/file`, `scp $host:$dir`, any `$VAR:` followed by text.

**What happens:** zsh reads `:` after a parameter as an expansion modifier and consumes the
next character. Of 15 letters measured, **13 fire and only one says so** — `:s` errors, and
`a A P c e h l q Q r t u` all silently rewrite the value. Three of them (`:a`, `:A`, `:P`)
prepend the working directory, producing a plausible absolute path that is simply wrong.

**Check:** `"${VAR}:rest"` — brace the parameter, quote the argument. Unconditionally, not for
a remembered set of letters: this entry originally published a letter list and got two of them
backwards, which is the argument against carrying a list at all. bash is unaffected, so a line
that works in a script fails when pasted into an interactive zsh.

**Expect to underestimate it.** The one loud letter aborts and gets fixed; the twelve silent
ones return an answer. You remember the instances that interrupted you, not the ones that
agreed with you.

**Measured 2026-09-20, `zsh` on `pax`, one isolated shell per case** (`c=abc123`): `$c:e…`
dropped the value entirely, `$c:t…` `$c:r…` `$c:q…` ate one character, `$c:u…` uppercased,
`$c:a…` prefixed the cwd. In a live check that night a loop over two commits returned `0` from
both arms — git's error went to stderr and `grep -c` counted an empty pipe as zero. A false
negative, which is the direction `EVIDENCE-DISCIPLINE.md` says needs evidence the test could
have gone positive. That second half is fixed at the source in `skills/lib/spawn-intent.sh`
rather than guarded by this note.
