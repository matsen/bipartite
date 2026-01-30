# Configuration Guide

bip supports global config files and per-repository config files. Environment variables can override API keys.

## Configuration Priority

1. **Environment variables** - Override API keys only
2. **Global config file** - `~/.config/bip/config.json`
3. **Per-repository config** - `.bipartite/config.json` (for repo-specific settings)

## Global Configuration

The global config file stores settings that apply across all bip commands, regardless of which repository you're in.

### Location

The config file follows the XDG Base Directory specification:
- Default: `~/.config/bip/config.json`
- Custom: Set `XDG_CONFIG_HOME` to change the base directory

### Creating the Config File

```bash
mkdir -p ~/.config/bip
cat > ~/.config/bip/config.json << 'EOF'
{
  "nexus_path": "~/re/nexus",
  "s2_api_key": "your-semantic-scholar-key",
  "asta_api_key": "your-asta-key",
  "github_token": "ghp_your-github-token",
  "slack_bot_token": "xoxb-your-slack-bot-token",
  "slack_webhooks": {
    "channel-name": "https://hooks.slack.com/services/..."
  }
}
EOF
```

### Configuration Options

| Field | Environment Override | Description |
|-------|---------------------|-------------|
| `nexus_path` | — | Default bipartite repository path. Allows running bip commands from anywhere. |
| `s2_api_key` | `S2_API_KEY` | Semantic Scholar API key for higher rate limits |
| `asta_api_key` | `ASTA_API_KEY` | ASTA MCP API key |
| `github_token` | `GITHUB_TOKEN` | GitHub personal access token for API calls |
| `slack_bot_token` | `SLACK_BOT_TOKEN` | Slack bot token for reading channel history |
| `slack_webhooks` | `SLACK_WEBHOOK_<CHANNEL>` | Slack webhook URLs keyed by channel name |

### Example: Running bip from Anywhere

With `nexus_path` configured, you can run bip commands from any directory:

```bash
# Without config - must be in a bipartite repo
cd ~/re/nexus
bip search "phylogenetics"

# With global config - works from anywhere
cd /tmp
bip search "phylogenetics"  # Uses nexus_path from config
```

## Environment Variables

Environment variables override config file values for API keys. This is useful for:
- Overriding config temporarily
- CI/CD environments
- Keeping secrets out of config files

### Supported Variables

```bash
export S2_API_KEY="your-key"
export ASTA_API_KEY="your-key"
export GITHUB_TOKEN="ghp_..."
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_WEBHOOK_CHANNELNAME="https://hooks.slack.com/..."
```

## Per-Repository Configuration

Each bipartite repository has its own config file at `.bipartite/config.json`. This file stores repository-specific settings that don't belong in the global config.

### Example

```json
{
  "pdf_root": "~/papers",
  "pdf_reader": "skim",
  "papers_repo": "~/re/bip-papers"
}
```

### Options

| Field | Description |
|-------|-------------|
| `pdf_root` | Directory containing PDF files |
| `pdf_reader` | PDF reader to use: `system`, `skim`, `zathura`, `evince`, `okular` |
| `papers_repo` | Path to a linked papers repository |

## Migration from Environment Variables

If you're currently using environment variables exclusively, you can migrate to the config file approach:

1. Create the config directory:
   ```bash
   mkdir -p ~/.config/bip
   ```

2. Create the config file with your current values:
   ```bash
   cat > ~/.config/bip/config.json << 'EOF'
   {
     "nexus_path": "~/re/nexus",
     "s2_api_key": "your-current-s2-key",
     "github_token": "your-current-github-token"
   }
   EOF
   ```

3. Optionally remove the environment variables from your shell config (they'll still work as overrides).

## Security Considerations

- The config file may contain API keys - ensure proper file permissions:
  ```bash
  chmod 600 ~/.config/bip/config.json
  ```
- For shared machines, prefer environment variables for sensitive values
- Never commit API keys to version control

## Troubleshooting

### "No bipartite repository found"

If you see this message, bip couldn't find a repository. The message includes setup instructions:

```
No bipartite repository found.

Tip: Create ~/.config/bip/config.json to set a default nexus:
  mkdir -p ~/.config/bip
  echo '{"nexus_path": "~/re/nexus"}' > ~/.config/bip/config.json
```

### Checking Current Configuration

To verify your config is being read correctly:

```bash
# Check if config file exists
cat ~/.config/bip/config.json

# Test by running a simple command
bip --version
```

### Path Expansion

The `~` character is automatically expanded to your home directory in the config file:

```json
{
  "nexus_path": "~/re/nexus"
}
```

This is equivalent to `/Users/yourname/re/nexus` (or `/home/yourname/re/nexus` on Linux).
