# Workflow Coordination

Bipartite provides visibility across your team's GitHub repositories, Slack channels, and project boards — without checking each one individually.

## Check-ins

`bip checkin` scans GitHub activity across all tracked repos:

```bash
bip checkin                     # Items needing your attention since last checkin
bip checkin --since 7d          # Last week (does not update .last-checkin.json)
bip checkin --since 12h         # Last 12 hours
bip checkin --all               # All activity, not just action-needed
bip checkin --category code     # Only repos in "code" category
bip checkin --repo org/repo     # Single repo
bip checkin --summarize         # Include LLM-generated summaries
```

Each run saves the current timestamp to `.last-checkin.json`, so the next run picks up where you left off (falling back to 3 days if the file doesn't exist). Using `--since` overrides the window without updating the state file.

By default, checkin filters to items where the "ball is in your court" — PRs awaiting your review, issues assigned to you, discussions needing your response. Use `--all` to see everything.

Requires `sources.json` in the working directory (typically your nexus repo).

## Digests

Digests aggregate activity by channel (a group of related repos) and produce LLM-summarized reports:

```bash
bip digest --channel dasm2                  # Preview digest
bip digest --channel dasm2 --verbose        # Include PR/issue body summaries
bip digest --channel dasm2 --post           # Post to Slack
bip digest --channel dasm2 --since 2w       # Custom time range
bip digest --channel dasm2 --post-to other  # Override destination channel
bip digest --repos org/a,org/b --channel x  # Override repos to scan
```

Channels are defined in `sources.json` via the `"channel"` field on repos. The digest organizes work by research theme rather than by repository.

### Narrative Digests

For prose-style summaries organized by research themes, use the Claude Code skill:

```
/bip.narrative dasm2
/bip.narrative dasm2 --since 2w --verbose
```

Output goes to `narrative/{channel}/{YYYY-MM-DD}.md`.

## Project Boards

Sync with GitHub project boards:

```bash
bip board list                    # View board items by status
bip board add org/repo#123        # Add issue to board
bip board move org/repo#123 Done  # Move to different status
bip board remove org/repo#123     # Remove from board
bip board sync                    # Report mismatches with beads
bip board refresh-cache           # Refresh cached board metadata
```

Boards are configured in `sources.json` under the `"boards"` key. Use `--board owner/number` to target a specific board if you have multiple.

## Spawning Sessions

Launch a Claude Code session with issue context pre-loaded:

```bash
bip spawn org/repo#123                          # Open issue in tmux window
bip spawn https://github.com/org/repo/pull/456  # Works with URLs too
bip spawn --prompt "Explore the clamping question"  # Adhoc session without issue
```

Requires tmux. The spawned session gets the issue/PR context so the agent can start working immediately.

## Task Hierarchy

Visualize your beads task tree:

```bash
bip tree --open                       # Generate and open in browser
bip tree --output tasks.html          # Write to file
bip tree --since 2026-01-20           # Highlight recently created beads
```

## Slack Integration

Read and ingest Slack channel history:

```bash
bip slack channels                              # List configured channels
bip slack history fortnight-goals               # Last 14 days of messages
bip slack history fortnight-goals --days 7      # Last week
bip slack history fortnight-goals --since 2026-01-01
bip slack history fortnight-goals --human       # Markdown output
bip slack history fortnight-goals --limit 50    # Cap results
```

Ingest messages into a queryable store:

```bash
bip slack ingest fortnight-goals --store goals
bip slack ingest fortnight-goals --store goals --create-store  # Create store if needed
```

Requires `SLACK_BOT_TOKEN` with `channels:history`, `channels:read`, and `users:read` scopes.

## Claude Code Skills

| Skill | Description |
|-------|-------------|
| `/bip.checkin` | Interactive activity check-in |
| `/bip.narrative <channel>` | Generate themed prose digest |
| `/bip.digest` | Generate and post Slack digest |
| `/bip.spawn` | Launch Claude session with context |
| `/bip.board` | Project board operations |
| `/bip.tree` | Task hierarchy visualization |

Skills are installed by symlinking from the bipartite repo:

```bash
ln -s $(pwd)/.claude/skills/* ~/.claude/skills/
```

## Configuration

Coordination commands read from files in your nexus repo:

- `sources.json` — Repository list, channel mappings, and board config
- `config.json` — Local path configuration
- `context/` — Project context files for narrative generation
- `narrative/preferences.md` — Shared formatting rules for narrative digests
- `narrative/{channel}.md` — Per-channel themes and repo context

### Environment Variables

| Variable | Description |
|----------|-------------|
| `SLACK_BOT_TOKEN` | Slack bot token (requires `channels:history`, `channels:read`, `users:read` scopes) |

## A Workflow in Practice

**Alice**, a graduate student, downloads two new papers to her Paperpile folder. Her coding agent reads both papers, determines relevance, fetches missing references via Semantic Scholar, and adds them to the library with edges linking to the group's concepts.

**Bernadetta**, the PI, pulls the changes and runs `bip rebuild`. Her agent scans the additions against her manuscripts and adds relevant papers to the references.

**Friday afternoon**, an agent runs `/bip.narrative` to generate a themed digest of the week's work across all repos. Bernadetta reviews it and posts to Slack with `bip digest --post`.
