---
tags: [shell, measurement]
measured: 2026-09-20
---

# `$VAR:path` in zsh is a modifier site, not string concatenation

**Looks right:** `git show $c:experiments/foo.py`, with `c` holding a commit SHA.

**What happens:** zsh parses `:e` as a parameter-expansion modifier. Behaviour depends on the
character after the colon, and the majority of this repo's directory names hit a *silent* mode
(`c=abc123`, reproduced in one run per row, 2026-09-20 on `pax`):

| written | got | |
|---|---|---|
| `$c:experiments/x` | `xperiments/x` | **silent** — SHA dropped entirely |
| `$c:tests/z` | `abc123ests/z` | **silent** — one char eaten |
| `$c:results/w` | `abc123esults/w` | **silent** |
| `$c:lib/t` | `abc123ib/t` | **silent** |
| `$c:src/y`, `$c:scripts/v`, `$c:skills/q` | `bad substitution` | loud |
| `$c:docs/u`, `$c:build/s` | `abc123:docs/u` | correct |

**Not a one-off:** it produced a wrong answer in a live check the same night — a loop over two
commits returned `0` from both, contradicting a diff just read. `git`'s error went to stderr and
`grep -c` returned `0` from the empty pipe, so stdout looked uniform and the failure read as
"nothing found." It fails *toward a negative result*, which is the direction
`EVIDENCE-DISCIPLINE.md` already says needs evidence the test could have gone positive.

**Check:** `git show "${c}:${path}"` — brace the variable, quote the argument.

**Recognise it without memorising the letter list:** a `$VAR` immediately followed by `:` is a
modifier site in zsh. The affected letters are `e t r l s a h q u x`, so `experiments/`,
`tests/`, `results/`, `lib/` and `skills/` are all affected and `docs/` and `build/` are not —
unguessable from outside.
