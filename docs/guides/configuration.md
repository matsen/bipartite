# Configuration Guide

bip uses a global config file and per-repository config files, all in YAML format.

## Configuration Files

1. **Global config file** - `~/.config/bip/config.yml`
2. **Per-repository config** - `.bipartite/config.yml` (for repo-specific settings)

## Global Configuration

The global config file stores settings that apply across all bip commands, regardless of which repository you're in.

### Location

The config file follows the XDG Base Directory specification:
- Default: `~/.config/bip/config.yml`
- Custom: Set `XDG_CONFIG_HOME` to change the base directory

### Creating the Config File

```bash
mkdir -p ~/.config/bip
cat > ~/.config/bip/config.yml << 'EOF'
nexus_path: ~/re/nexus
s2_api_key: your-semantic-scholar-key
asta_api_key: your-asta-key
github_token: ghp_your-github-token
slack_bot_token: xoxb-your-slack-bot-token
slack_webhooks:
  channel-name: https://hooks.slack.com/services/...
EOF
```

### Configuration Options

| Field | Description |
|-------|-------------|
| `nexus_path` | Default bipartite repository path. Allows running bip commands from anywhere. |
| `s2_api_key` | Semantic Scholar API key for higher rate limits |
| `asta_api_key` | ASTA MCP API key |
| `github_token` | GitHub personal access token ([setup guide](#github-authentication)) |
| `slack_bot_token` | Slack bot token for reading channel history |
| `slack_webhooks` | Slack webhook URLs keyed by channel name |

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

## GitHub Authentication

bip uses GitHub through two independent authentication paths. You need **both** configured for full functionality.

### Step 1: Authenticate the `gh` CLI

Most bip commands (`checkin`, `board`, `spawn`, `digest`) call the GitHub API through the [`gh` CLI](https://cli.github.com/). You must authenticate it first:

```bash
gh auth login
```

Follow the interactive prompts. When asked about scopes, the defaults (`repo`, `read:org`) cover most bip features. However, for **project board** commands (`bip board list/add/move/remove`), you also need the `project` scope:

```bash
gh auth refresh --scopes project
```

Verify your authentication:

```bash
gh auth status
```

### Step 2: Add a Personal Access Token to bip config

bip's Go HTTP client uses a separate token from its config file for repository metadata fetching and higher rate limits. To create one:

1. Go to [github.com/settings/tokens](https://github.com/settings/tokens) (Profile photo → Settings → Developer settings → Personal access tokens)
2. Choose **Fine-grained tokens** (recommended) or **Tokens (classic)**

#### Fine-grained token (recommended)

Fine-grained tokens are more secure because you scope them to specific repositories and permissions:

1. Click **Generate new token**
2. Set a descriptive name (e.g., "bip CLI")
3. Set expiration (90 days or custom)
4. Under **Repository access**, choose "All repositories" or select specific repos you track with bip
5. Under **Permissions → Repository permissions**, enable:
   - **Issues**: Read-only
   - **Pull requests**: Read-only
   - **Metadata**: Read-only (automatically selected)
6. Under **Permissions → Organization permissions** (if you use org project boards):
   - **Projects**: Read and write
7. Click **Generate token** and copy the value

#### Classic token

If you prefer classic tokens or need compatibility with older GitHub Enterprise versions:

1. Click **Generate new token (classic)**
2. Set a descriptive name and expiration
3. Select these scopes:
   - **`repo`** — Read access to repositories (issues, PRs, comments)
   - **`read:org`** — Read organization membership (needed for org project boards)
   - **`project`** — Read/write GitHub project boards
4. Click **Generate token** and copy the value

#### Add the token to config

```yaml
# ~/.config/bip/config.yml
github_token: ghp_your-token-here   # classic token
# or
github_token: github_pat_your-token-here  # fine-grained token
```

### Which commands need what

| Command | Auth method | What it does |
|---------|-------------|--------------|
| `bip checkin` | `gh` CLI | Fetches issues, PRs, comments across repos |
| `bip board list/add/move/remove` | `gh` CLI | Reads/writes GitHub project boards (needs `project` scope) |
| `bip spawn` | `gh` CLI | Fetches issue/PR details for tmux sessions |
| `bip digest` | `gh` CLI | Generates activity summaries |
| `bip repo add/refresh` | `github_token` | Fetches repository metadata |

### Troubleshooting

**"gh: not logged in"** — Run `gh auth login`.

**403 on project board commands** — You need the `project` scope: `gh auth refresh --scopes project`.

**Rate limiting on `bip repo` commands** — Add `github_token` to your config file. Without a token, these requests are unauthenticated and limited to 60/hour.

## Per-Repository Configuration

Each bipartite repository has its own config file at `.bipartite/config.yml`. This file stores repository-specific settings that don't belong in the global config.

### Example

```yaml
pdf_root: ~/papers
pdf_reader: skim
papers_repo: ~/re/bip-papers
```

### Options

| Field | Description |
|-------|-------------|
| `pdf_root` | Directory containing PDF files |
| `pdf_reader` | PDF reader to use: `system`, `skim`, `zathura`, `evince`, `okular` |
| `papers_repo` | Path to a linked papers repository |

## Security Considerations

- The config file may contain API keys - ensure proper file permissions:
  ```bash
  chmod 600 ~/.config/bip/config.yml
  ```
- Never commit API keys to version control

## Troubleshooting

### "No bipartite repository found"

If you see this message, bip couldn't find a repository. The message includes setup instructions:

```
No bipartite repository found.

Tip: Create ~/.config/bip/config.yml to set a default nexus:
  mkdir -p ~/.config/bip
  echo 'nexus_path: ~/re/nexus' > ~/.config/bip/config.yml
```

### Checking Current Configuration

To verify your config is being read correctly:

```bash
# Check if config file exists
cat ~/.config/bip/config.yml

# Test by running a simple command
bip --version
```

### Path Expansion

The `~` character is automatically expanded to your home directory in the config file:

```yaml
nexus_path: ~/re/nexus
```

This is equivalent to `/Users/yourname/re/nexus` (or `/home/yourname/re/nexus` on Linux).
