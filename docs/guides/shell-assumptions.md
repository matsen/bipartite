# Shell assumptions

We don't know what shell you run. This is what the shipped shell code assumes, and the one rule
that matters if you change it.

## Executed vs sourced

| file | how skills invoke it | runs under |
|---|---|---|
| `skills/lib/clone-currency.sh` | as a path — `".../lib/clone-currency.sh"` | its shebang, `bash` |
| `skills/lib/fleet-collisions.sh` | as a path | its shebang, `bash` |
| `scripts/marimo-check.sh` | as a path | its shebang |
| `skills/lib/spawn-intent.sh` | **`source`d** | **your shell**, whatever it is |

**A shebang is ignored when a file is sourced.** So the first three are safe in any login shell
— they re-exec under bash no matter who calls them. `spawn-intent.sh` has no shebang, by design,
because it defines functions the caller needs in its own environment. It runs in your shell.

## The rule

**`skills/lib/spawn-intent.sh` must work in bash and zsh.** Don't add `[[ ]]`, arrays, `local -n`,
`${var,,}`, or anything else that is bash-only. It currently uses none of these.

`TestSpawnIntentSourcesUnderZsh` in `tests/integration/spawn_intent_test.go` sources it under both
and requires identical output; it skips when zsh isn't installed, so the suite still passes
without it. Every other test in that file invokes bash explicitly and does not cover this.

If you add another sourced helper, give it no shebang and add it to that test.

## Two traps that are not ours

These bite in agent sessions and in zsh terminals regardless of this package, and both produce
wrong answers rather than errors:

- Inside a Claude Code Bash call, `grep` is a shim that skips gitignored files and uses basic
  regex — `skills/lib/hazards/claude-code-grep-shim.md`.
- In zsh, `$VAR:` is a parameter-modifier site, not string concatenation —
  `skills/lib/hazards/zsh-colon-modifier.md`.

Neither is fixed by choosing a shell for this package; they're listed here because the first
thing a shell-portability question runs into is one of them.

And the shells are not strictly ordered by safety, so don't read the portability rule above as
"bash is the safe one". Bash reads a script incrementally by byte offset, so editing a file while
it runs lands in the running process — a longer replacement resumes mid-token, a shorter one
stops early and silently, and **both exit 0**. Measured on this fleet; zsh did not reproduce the
silent-early-stop case. Run anything long-lived from a copy outside the worktree
(`cp s /tmp/s.$$.sh && bash /tmp/s.$$.sh`), which also protects it from a `git checkout` or a
merge rewriting it underneath.
