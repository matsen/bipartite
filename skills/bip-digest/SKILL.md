---
name: bip-digest
description: "Generate Slack activity digests from GitHub activity across a channel's repos. Preview by default; --post to send."
---

# /bip-digest

Generate activity digest (preview only by default).

## Instructions

```bash
bip digest --channel dasm2                  # Preview digest for channel
bip digest --channel dasm2 --since 2d       # Last 2 days
bip digest --channel dasm2 --post           # Actually post to Slack
bip digest --channel dasm2 --post-to scratch --post  # Post to scratch channel
```

`bip digest --help` has the rest of the flags.

## What it does

1. Scans repos associated with the channel (from sources.yml)
2. Fetches merged PRs, new issues, active discussions
3. Uses LLM to generate summary
4. Shows preview (default) or posts to Slack (if --post)
