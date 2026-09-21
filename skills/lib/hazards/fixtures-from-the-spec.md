---
tags: [measurement, testing]
measured: 2026-09-21
---

# Mutation testing proves a test can fail; it cannot prove the test is about the right thing

**Looks right:** every behaviour in the spec has a test, and each test was confirmed red against an
implementation lacking that behaviour. The suite is not decorative — each assertion demonstrably
bites.

**What happens:** the fixtures were written from the same specification as the code, so a defect
*in the specification* is present in both and invisible to every mutation. Mutation testing varies
the implementation against a fixed population. It cannot vary the population.

**Measured on `matsen/bipartite` #253.** `bip epic collide` shipped with eleven mutation-verified
tests over real git fixtures — merge-base versus two-dot, rename old paths, untracked files, a
`.git` that is a file, both rebase backends. The first run against a real 17-slot pool reported
**21 directories**. Two of the extras were genuine clones of the same origin that no worker is ever
spawned into (CI clones that reset hard to `origin/main` at run time, so a moving HEAD there is the
expected state). A collision reported against one cannot involve a worker, and the action on a
reported collision is to delay a spawn.

The issue specified *"the same universe as the script"*, and the script's universe was every
directory. **The sibling script in the same directory already scoped correctly**, and the skill
already carried the sentence — *"a check at full power over the wrong population."* The author
followed the spec instead of the sibling; the fixtures inherited it.

**Check, and it is two questions rather than one:**

- *Does my test fail without the behaviour?* — mutation answers this.
- *Did I choose the population, or inherit it from the thing I am testing?* — **nothing in the
  suite answers this.** Run it once against real state and read the first line of output. The
  count of things examined is the cheapest discriminator; a population that disagrees with the
  authoritative list by any amount is the finding.

**Corollary for a spec.** When an issue tells you what a predecessor did, grep for a *sibling* that
answers the same question — the predecessor is the thing being replaced, so its choices are the
ones most likely to be the defect. Here `clone-currency.sh` was one directory away.
