# /digest

Generate and post activity digest to Slack.

## Instructions

```bash
flowc digest --channel dasm2                  # Weekly digest for channel
flowc digest --channel dasm2 --since 2d       # Last 2 days
flowc digest --channel dasm2 --post-to scratch  # Test to scratch channel
```

## Options

- `--channel CHANNEL` — Channel whose repos to scan (required)
- `--since PERIOD` — Time period (e.g., 1w, 2d, 12h). Default: 1w
- `--post-to CHANNEL` — Override destination (e.g., scratch for testing)
- `--repos REPOS` — Override repos (comma-separated)

## What it does

1. Scans repos associated with the channel (from sources.json)
2. Fetches merged PRs, new issues, active discussions
3. Uses LLM to generate summary
4. Posts to Slack webhook (env: `SLACK_WEBHOOK_<CHANNEL>`)

## Testing

Always test with `--post-to scratch` first to avoid spamming real channels.
