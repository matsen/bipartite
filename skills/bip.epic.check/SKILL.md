---
name: bip.epic.check
description: Review an EPIC issue markdown file for strategic clarity, cost soundness, and scope discipline
allowed-tools: Agent, Bash, Read, Edit
---

# /bip.epic.check

Review an EPIC issue markdown file for the qualities that matter at
the strategic/research level. EPICs are roadmaps that guide months of
work — they are NOT implementation tasks.

This skill complements `/bip.issue.check` (which reviews implementation-
ready issues for data paths, column names, test configs). `/bip.epic.check`
reviews research direction documents for the failure modes that waste
months, not hours.

## Usage

```
/bip.epic.check ISSUE-my-epic.md
/bip.epic.check                     # uses most recently discussed ISSUE-*.md
```

## Workflow

### Step 1: Determine the file path

- If `$ARGUMENTS` is provided, use that as the file path
- Otherwise, check conversation context for the most recently discussed
  EPIC issue file (ISSUE-*.md)
- If unclear, ask the user which file to use

### Step 2: Load context

Read the EPIC file. Also check for:
- `CLAUDE.md` in the repo root (project conventions)
- Any existing EPICs in the target repo (for style consistency):
  ```bash
  # If the EPIC targets a specific repo, check its existing EPICs
  gh issue list --repo <org/repo> --label "EPIC" --limit 5 --json number,title
  ```

### Step 3: Spawn a review subagent

Use the Agent tool to launch a general-purpose subagent that reads the
EPIC file and checks the following. The subagent should verify claims
by reading referenced files, running cost calculations, and checking
existing repo structure.

Pass the full EPIC text and any context from Step 2 to the subagent.

#### 1. Vision clarity

- **Objective vs strategy**: Does the EPIC clearly distinguish the
  optimization objective (what we're maximizing) from the search
  strategy (how we navigate the space)? A common failure is claiming a
  "new objective" when only the search strategy changed, or vice versa.
  Flag as **HIGH** if conflated.

- **Novelty claim**: Is the "what's novel" claim precise? Can you tell
  exactly which existing method does the closest thing and what the
  delta is? Vague novelty ("combines X and Y") without explaining why
  no one has done this is a red flag.

- **Achievability**: Is there a plausible argument that this can work?
  Not a proof, but at least a reason to believe the approach isn't
  doomed (e.g., a related method works in a simpler setting, a
  theoretical argument, or preliminary evidence).

#### 2. Scope discipline

- **"What this is NOT" section**: Is there one? EPICs attract scope
  creep because they're long-running. Explicitly listing out-of-scope
  items prevents drift. Flag as **HIGH** if missing.

- **Phase boundaries**: Could someone tell, for each phase, whether
  it's done or not? Each phase needs a concrete gate (a test, a
  metric threshold, a comparison). "Implement X" is not a gate;
  "X produces identical output to reference on test case Y" is.

- **Phase dependencies**: Are co-dependencies between components
  explicit? If Phase N requires something from Phase M that isn't
  listed as a prerequisite, flag as **HIGH**.

#### 3. Cost and complexity claims

This is the single highest-value check. EPICs routinely wave hands
at computational cost with analogies ("same pattern as X") that don't
survive arithmetic.

- **Back-of-envelope verification**: For every complexity claim
  (O(...) notation, "same as X", "< Nx slower"), work out the actual
  cost on a concrete example. If the EPIC says "O(n * tree_pass)",
  compute what that means for n=16, n=64, n=256. If it says "< 10x
  wall time", check whether the stated complexity allows that.
  Flag as **CRITICAL** if a cost claim is wrong by > 2x.

- **Scaling cliffs**: Are there operations whose cost changes
  qualitatively at some scale? (e.g., fits in cache at n=32 but not
  n=64; profile size grows polynomially but DP cost is multiplicative).
  Flag as **HIGH** if a scaling cliff exists but isn't discussed.

- **Success criteria vs cost**: Do the success criteria (especially
  wall-time criteria) survive the cost analysis? If criterion says
  "< 10x" but the complexity analysis shows 32x, flag as **CRITICAL**.

#### 4. Risk analysis

- **Risks and mitigations**: Is there a section? At minimum, the EPIC
  should identify the top 3 ways the approach could fail and name an
  early signal for each. Flag as **HIGH** if missing.

- **Kill criterion**: Which success criterion catches fundamental
  failure earliest? If the first real test is in the final phase, the
  EPIC risks months of work before discovering the approach doesn't
  work. Flag as **HIGH** if the earliest meaningful accuracy test is
  in the last phase.

- **Failure modes specific to this approach**: Generic risks ("might
  be slow", "might not be accurate") don't count. The risks should
  be specific to the approach — failure modes that wouldn't apply to
  alternative approaches. (e.g., "profile blowup" is specific to
  profile-based methods; "might not beat PRANK" is generic.)

#### 5. Reference and attribution accuracy

- **Author attributions**: Are papers correctly attributed? Pay
  special attention to the PI's own papers — don't use generic
  "et al." for first-author work. Check that reference IDs (bip keys)
  are correct if provided.

- **Characterization of related work**: For each method described in
  "Connection to existing work", is the characterization accurate?
  Check for common mischaracterizations: saying a method "doesn't do X"
  when it does, or claiming a method uses technique Y when it uses Z.

#### 6. Structural completeness

Check for the following sections (based on established EPIC patterns
in matsengrp repos). Flag missing sections as **MEDIUM**:

- [ ] Vision
- [ ] What this is NOT (scope boundaries)
- [ ] Background and motivation
- [ ] Design (algorithm description)
- [ ] Clean-room / implementation strategy (if reimplementing)
- [ ] Build-up sequence with phases and gates
- [ ] Risks and mitigations
- [ ] Success criteria (measurable)
- [ ] Status dashboard (can be empty placeholder)
- [ ] Key findings (can be empty placeholder)
- [ ] Open issues (can be empty placeholder)
- [ ] Key references
- [ ] Context (file paths, repo targets)

#### 7. Insertion order / search order (domain-specific)

If the EPIC proposes a greedy or incremental algorithm, check whether
the ordering/scheduling of operations is discussed. Greedy algorithms
are notoriously order-sensitive. Flag as **HIGH** if the EPIC proposes
incremental construction but defers order discussion to the final phase.

### Step 4: Report findings

Present findings grouped by severity:

- **CRITICAL**: Cost claims that don't survive arithmetic; success
  criteria contradicted by the design; fundamental logical gaps
- **HIGH**: Missing scope boundaries; missing risk analysis; phase
  dependencies unclear; kill criterion too late
- **MEDIUM**: Style/structure gaps; imprecise language; missing
  placeholder sections
- **LOW**: Nitpicks, minor attribution issues

For each finding, state:
1. What the problem is (one sentence)
2. Where in the file it occurs (quote the relevant text)
3. What the fix should be (concrete suggestion)

### Step 5: Fix gaps (with approval)

Present findings to the user. Do NOT edit the file without explicit
approval — EPICs represent strategic decisions that require human
judgment.

After the user approves specific fixes, apply them to the file.

### Step 6: Optionally submit

If the user wants to submit the EPIC as a GitHub issue, invoke
`/bip.issue.file` with the file path. EPICs are often kept as local
markdown files and submitted later, so do not assume submission.
