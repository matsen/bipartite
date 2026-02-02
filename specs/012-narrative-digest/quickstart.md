# Quickstart: Narrative Digest

## Prerequisites

1. **bipartite CLI** built and in PATH:
   ```bash
   cd ~/re/bipartite
   go build -o bip ./cmd/bip
   ```

2. **nexus directory** with channel config:
   ```bash
   ls ~/re/nexus/narrative/
   # Should contain: preferences.md, {channel}.md
   ```

3. **GitHub CLI** (`gh`) authenticated for fetching activity

4. **Claude CLI** available for verbose mode (optional)

## Basic Usage

### Generate Narrative Digest

```bash
# From nexus directory
cd ~/re/nexus

# Generate narrative for dasm2 channel (last week)
/bip.narrative dasm2

# With custom date range
/bip.narrative dasm2 --since 2w

# With verbose mode (includes PR/issue body summaries)
/bip.narrative dasm2 --verbose
```

Output is written to `narrative/{channel}/{YYYY-MM-DD}.md`.

### Preview Raw Activity

```bash
# Preview activity without posting (new default)
bip digest --channel dasm2

# With custom date range
bip digest --channel dasm2 --since 2w

# Actually post to Slack (requires --post)
bip digest --channel dasm2 --post
```

## Configuration

### Channel Config (`narrative/{channel}.md`)

```markdown
# {channel} Narrative Configuration

Inherits from [preferences.md](preferences.md).

## Themes

1. **Theme Name** - Description of what belongs here
2. **Another Theme** - Another category

## Project-Specific Preferences

- Any channel-specific formatting rules

## Repo Context

- **repo-name**: Description of what this repo contains
```

### Shared Preferences (`narrative/preferences.md`)

```markdown
# Narrative Digest Preferences

## Attribution
- Do NOT use attribution ("Will's PR") - just describe the work

## Format
- Use hybrid format: bullets for lists, prose for connected items
- Mark status: "In progress:", "Open:"
- End with "Looking Ahead" section

## Content
- Stick to available information - do not invent context
```

## Workflow

1. Run `/bip.narrative {channel}` from Claude Code
2. Review generated markdown in `narrative/{channel}/{date}.md`
3. Edit if needed (human review encouraged)
4. Commit when satisfied

## Troubleshooting

### "No config file for channel"

Create `narrative/{channel}.md` following the template above.

### "No repos configured for channel"

Add `"channel": "{channel}"` to repos in `sources.yml`.

### Empty output

Check that repos have activity in the date range:
```bash
bip digest --channel {channel} --since 2w
```
