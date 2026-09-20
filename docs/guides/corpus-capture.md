# Corpus capture: why rules land outside the repo, and the plan to fix it

**Status: plan, not doctrine.** Written 2026-09-20 by four sessions (`bip-agent`,
`phyz-conductor`, `sf-conductor`, `phyz-epic`, `sf-epic`). Delete this file when the work
lands; it describes a change, not a convention.

## The deliverable

Four mechanisms that give a rule a cheap, tracked place to land — a hazard file, a test case,
a tracked continuation entry, or a spawn preamble — so that guidance stops accumulating in
untracked files and retyped prompts. One of them also retires ~31% of `bip-epic`.

## The problem, measured

Guidance that demonstrably fires is substantially outside the tracked corpus.

| where | size | tracked? |
|---|---|---|
| `skills/` + root doctrine, four files above 50 KB | 493,379 bytes ≈ 123k tokens | yes |
| three `CONTINUATION*.md`, one per epic programme | 72,225 bytes | **no** |
| spawn-prompt fleet facts, recomposed per spawn | ~900 words of ~1,597, per prompt | **no** |

The highest-firing rule in one measured `/bip-epic` session — *"check one load-bearing claim
against its source"* — is in **no tracked file**. It lives in a gitignored continuation.

### Root cause: the corpus has one container size and one gate

A rule can be a multi-thousand-word section in a 20–28k-word skill, reviewed under
*conductor drafts, epic reviews, both agree, then push*. There is nothing smaller. So a rule
too small to justify that lands in an untracked continuation, or is retyped from memory into
the next spawn prompt.

The gate is correct. The **granularity** is what is disproportionate — which is why
exhortation to use the gate has not worked and will not.

Two consequences, both observed:

- **Mirroring is the only copy operation the corpus supports.** There is no addressable unit
  to cite, so text is duplicated by hand and the copies drift. This produced the
  `Two practices` / `Three practices` desync between `bip-epic` and `bip-conductor`.
- **Retyped corpus is where the errors are.** In one spawn prompt, the measured content was
  correct (commands were run); the retyped corpus half produced two defects — timestamps
  ~15 minutes in the future, and an unlabelled figure relayed without its arm.

## Plan

Four layers. Each is a mechanism, not a rule. Only layer 3 reduces load-time cost; the others
fix capture and drift, and they say so.

### 1. `skills/lib/hazards/<slug>.md` — the missing container

5–15 lines each, tagged with the contexts they apply to (`nextflow-cache`, `measurement`,
`tmux`, `landing`, `remote-host`). Shape: the wrong-looking-right artifact, one number showing
it is not a one-off, the discriminating check. Skills link rather than contain.

This is the cause, so it lands first. A hazard no brief ever includes becomes a deletion
candidate **on evidence** (`grep -l` across emitted briefs), which makes the
anticipatory-rule test mechanical instead of a judgement call.

### 2. Track the continuations, and give them a drain

The continuation file is a correct artifact doing a job nothing else does: carrying live,
session-scoped guidance to the next session in that clone. It works. What it lacks is a drain.
Do not abolish it — the need is real and would re-create it under another name.

- **Track them.** One `.gitignore` line. Committed directly to `main` by the owning session,
  **no PR** — an EPIC's working notes are the same category as its issue body, which is
  already edited without review. This adds no friction to the currently frictionless path,
  which is the only reason it will be adopted. Buys durability, peer visibility (the
  alternation argument requires peers to read each other's state, and today they cannot), and
  `git blame`, without which no aging check is possible.
- **Promotion marker.** `⏳ PROMOTE → docs/ml/369-method.md` on the line or block. That is the
  entire syntax. Writing it must be cheaper than deciding not to.
- **The drain.** `scripts/check_promotions.py`, wired into `make check-fast`: scan tracked
  `CONTINUATION-*.md` for markers, `git blame` each to its commit date, fail when any exceeds
  N days (start at 14). `make check-fast` already runs on every PR, so the drain becomes a gate
  on work that is already happening and cannot be forgotten, because it turns red.

Precedent, verified: `matsengrp/phyz` `scripts/check_experiment_test_configs.py` is in
`CHECK_FAST_TARGETS`, carries `EXEMPTION_CUTOFF = "2026-09-19"`, and its exemption list is
shrink-only. Reuse all three properties — including printing the population every run, so
suppression by extending the date is **visible** rather than silent. Every TODO gate is
eventually silenced; a printed population is what makes that legible.

**Routing rule, so promotion is mechanical rather than a judgement:**

- Names a repo-specific artifact (file, fixture, experiment, issue number) → **repo-scoped**
  (`docs/`, that repo's `CLAUDE.md`). Normal repo PR. **No epic/conductor agreement.** This is
  most of it.
- Names only roles and artifacts present in every programme → **cross-programme** (`skills/`,
  `EVIDENCE-DISCIPLINE.md`). Existing gate, unchanged.

The gate was never the right instrument for most of what is piling up.

### 3. Retire the mechanisable bulk — the only layer that cuts load-time cost

"Most rules are mechanisable" is **false**: of nine traced rules, ~3 are fully mechanisable, 3
partly, 3 pure judgement. But measure **bytes per rule, not rules**, and the picture inverts —
the mechanisable ones are the biggest, and the pure-judgement ones are 791, 1,709 and 1,820
bytes.

`skills/bip-epic/SKILL.md`, 97,580 bytes, 29 sections. The top two are **31% of the file** and
both are a command plus its regression suite, written out as prose:

| bytes | section | is really |
|---|---|---|
| 15,116 | Step 4b: File-overlap collision detection | `bip collide <branch>...` |
| 15,086 | EPIC body update pattern | `bip epic-edit <issue> --body-file X` |

- **`bip epic-edit`** — pull, record `updatedAt`, guard *with the emptiness null-check*, push,
  verify read-back length and tail.
- **`bip collide`** — pairwise `git diff --name-only main...<branch>`, intersect,
  `git merge-tree`, hunk-position report, and `git grep -ln <SYMBOL>` blast radius split into
  code hits versus prose hits.

**A mechanisable rule that fires tonight lands as a test case, not a paragraph.** A test is
ungated (nobody reviews doctrine to add one), tracked, and runs instead of needing to be read
— and it cannot drift from the behaviour it describes, which is exactly how the
`Two practices` defect happened. Worked instance: the discovery that the conflict guard
compares `""` against `""` and fails open is three lines of test; it currently occupies ~400
words of prose.

**Falsifier, to be tested on the first command rather than assumed for both:** if `bip collide`
or `bip epic-edit` needs a judgement call per invocation, the prose returns as comments and
nothing is saved.

### 4. `bip spawn-preamble` — stop recomposing the corpus per spawn

One spawn prompt's fleet-facts section, by measured word count: ~900 of 1,597 words could not
have differed from the previous spawn. They are corpus, retyped from memory by a session that
has read the corpus. The rest are measurements about a particular instant — host load, live
slots, whether a brief's claim still holds — which genuinely cannot be tracked.

**The diagnostic, one question per sentence:** *would this sentence be true for the next spawn
too?* Yes → corpus, must not be composed. No → measurement, must not be tracked.

`bip spawn-preamble [--repo X]` emits the durable half from a tracked file; `bip spawn`
concatenates it ahead of the composed half, which shrinks to ~700 words. `bip spawn
--prompt-file` already exists, so this is one more input to a command that already runs — no
new convention, no hook, no review gate.

**Design constraint: the assembler emits inline text, not links.** A pointer is strictly weaker
than inline text for an agent reading cold. Measured: an EPIC-body ruling never reached an
issue-lead, because the lead reads the status file, worklog, issue and PR, and a ruling in none
of those does not exist. **De-duplicate the source; do not de-duplicate the delivery.**

## Sequencing

1. **`skills/lib/hazards/` and the `.gitignore` line.** Near-zero cost, unblocks the rest.
2. **`bip collide`.** Build this one first because it carries the falsifier — if it needs
   per-invocation judgement, the mechanisation layer dies before a second command is built.
3. **`check_promotions.py` drain**, then **`bip spawn-preamble`**.

## What this does not do

- **Layers 1, 2 and 4 do not reduce load-time cost.** Layer 2 routes growth to repo-scoped
  files that load only in their own repo, which beats everything aiming at `skills/`, but that
  is a routing win. Layer 4's preamble still lands in the worker's window. Only layer 3 cuts
  the 123k-token figure.
- **Tracking the continuations puts 72 KB into git and grows it.** That is judged correct —
  they are the record of what each programme learned — but it is a real cost, stated rather
  than buried.
- **Subtraction is not a criterion anywhere here**, for three independent reasons established
  before this plan: it cuts the case histories that make rules fire; it does not relieve
  load-time cost, since a skill is already a tracked file with no ceiling to escape; and it
  raises friction on the gated path, routing more rules into the ungated files.

## Why mechanisms rather than better prose

In one session, three mechanical checks held and the one judgement check failed — a substring
grep reported as a rule's location. The rule was loaded, in active use on adjacent citations in
the same document, and did not reach the adjacent sentence. Prose quality was not the binding
constraint. A command cannot misread itself.
