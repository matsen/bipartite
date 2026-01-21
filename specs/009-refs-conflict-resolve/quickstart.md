# Quickstart: Domain-Aware Conflict Resolution

**Feature Branch**: `009-refs-conflict-resolve`
**Date**: 2026-01-21

## Overview

The `bip resolve` command provides domain-aware conflict resolution for refs.jsonl merge conflicts. Git sees JSON as opaque blobs, but bip understands paper metadata and can intelligently merge or select the better version.

## Basic Usage

### Preview Conflicts (Dry Run)

See what would happen without modifying files:

```bash
bip resolve --dry-run
```

JSON output shows detected conflicts and proposed resolutions:
```json
{
  "resolved": 5,
  "ours_papers": 1,
  "theirs_papers": 2,
  "merged": 2,
  "operations": [
    {"paper_id": "Smith2024-ab", "action": "keep_theirs", "reason": "theirs has abstract, venue"},
    {"paper_id": "Jones2023-cd", "action": "merge", "reason": "complementary metadata"}
  ]
}
```

Human-readable output:
```bash
bip resolve --dry-run --human
```

### Auto-Resolve Conflicts

Resolve conflicts automatically (fails if true conflicts exist):

```bash
bip resolve
```

This will:
1. Parse conflict regions in refs.jsonl
2. Match papers by DOI (primary) or ID (fallback)
3. Auto-select the more complete version when one side has more data
4. Merge complementary metadata (e.g., ours has abstract, theirs has venue)
5. Include papers unique to either side
6. Write the resolved refs.jsonl

### Interactive Resolution

Handle true conflicts interactively:

```bash
bip resolve --interactive
```

For each unresolvable field conflict, you'll be prompted:
```
Conflict in 'abstract' for paper Smith2024-ab:
  [1] ours:   "We present a novel approach to..." (142 chars)
  [2] theirs: "This paper introduces a new..." (189 chars)
Enter choice [1/2]:
```

## Resolution Logic

### Paper Matching

Papers are matched between ours/theirs sides by:
1. **DOI** (primary): If both have the same non-empty DOI
2. **ID** (fallback): If DOIs don't match but IDs do

### Auto-Resolution Rules

| Scenario | Resolution |
|----------|------------|
| One side has more non-empty fields | Keep the more complete version |
| Complementary metadata (non-overlapping fields) | Merge both |
| Paper only on one side | Include it |
| Same field, same value | Use that value |
| Same field, different values | **True conflict** - needs `--interactive` |

### Field Priority

When comparing completeness, fields are weighted by importance:
1. Abstract (most valuable)
2. Authors
3. Venue
4. Publication date
5. DOI

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - conflicts resolved |
| 1 | Error - unresolvable conflicts without `--interactive` |
| 3 | Data error - malformed conflict markers |

## Examples

### Scenario 1: Simple Completeness

**Conflict:**
```
<<<<<<< HEAD
{"id":"paper1","doi":"10.1234/a","title":"Paper One"}
=======
{"id":"paper1","doi":"10.1234/a","title":"Paper One","abstract":"Full text...","venue":"Nature"}
>>>>>>> feature
```

**Resolution:** Keep theirs (has abstract and venue).

### Scenario 2: Complementary Merge

**Conflict:**
```
<<<<<<< HEAD
{"id":"paper2","doi":"10.1234/b","title":"Paper Two","abstract":"Our abstract"}
=======
{"id":"paper2","doi":"10.1234/b","title":"Paper Two","venue":"Science"}
>>>>>>> feature
```

**Resolution:** Merge → `{"id":"paper2","doi":"10.1234/b","title":"Paper Two","abstract":"Our abstract","venue":"Science"}`

### Scenario 3: True Conflict

**Conflict:**
```
<<<<<<< HEAD
{"id":"paper3","doi":"10.1234/c","abstract":"Version A"}
=======
{"id":"paper3","doi":"10.1234/c","abstract":"Version B"}
>>>>>>> feature
```

**Resolution:** Requires `--interactive` - both have different abstracts.

## Workflow Integration

### Typical Git Merge Workflow

```bash
# After git merge with conflicts
git status
# ... refs.jsonl has conflicts ...

# Preview what bip would do
bip resolve --dry-run --human

# Resolve automatically
bip resolve

# If true conflicts exist, use interactive mode
bip resolve --interactive

# Stage the resolved file
git add .bipartite/refs.jsonl

# Continue merge
git commit
```

### CI/CD Integration

```bash
# In CI, use dry-run to detect unresolvable conflicts
bip resolve --dry-run || echo "Manual conflict resolution needed"
```
