# /bip.digest

Generate activity digest (preview only by default).

## Instructions

```bash
bip digest --channel dasm2                  # Preview digest for channel
bip digest --channel dasm2 --since 2d       # Last 2 days
bip digest --channel dasm2 --post           # Actually post to Slack
bip digest --channel dasm2 --post-to scratch --post  # Post to scratch channel
```

## Options

- `--channel CHANNEL` — Channel whose repos to scan (required)
- `--since PERIOD` — Time period (e.g., 1w, 2d, 12h). Default: 1w
- `--post` — Actually post to Slack (default: preview only)
- `--post-to CHANNEL` — Override destination (e.g., scratch for testing)
- `--repos REPOS` — Override repos (comma-separated)

## What it does

1. Scans repos associated with the channel (from sources.json)
2. Fetches merged PRs, new issues, active discussions
3. Uses LLM to generate summary
4. Shows preview (default) or posts to Slack (if --post)

## Safe by Default

The digest command previews output by default. Use `--post` to actually send to Slack.
This prevents accidental posts to real channels.
