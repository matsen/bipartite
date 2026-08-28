---
name: bip-scout
description: Check remote server CPU, memory, and GPU availability via SSH
---

# /bip-scout

Check remote server availability and resource usage.
Runs `bip scout` to collect metrics, presents a human-readable summary, and answers follow-up questions by reasoning over the data.

## Usage

```
/bip-scout [question]
```

**Arguments:**
- `[question]` — Optional natural-language question (e.g., "which server has free GPUs?", "where should I run a training job?")

If no question is given, show a full summary of all servers.

## Workflow

### Step 1: Collect Server Metrics

Run `bip scout` to get JSON output:

```bash
bip scout
```

**If the command fails**, check for common issues:
- Missing `servers.yml`: "No servers.yml found — see `bip scout --help` for config location."
- SSH authentication failure: Report the error and suggest checking SSH agent and `~/.ssh/config`.
- All servers offline: Report that all servers are unreachable and suggest checking network/VPN.

### Step 2: Parse JSON Output

The JSON output has this structure:

```json
{
  "servers": [
    {
      "name": "servername",
      "status": "online",
      "metrics": {
        "cpu_percent": 45.2,
        "memory_percent": 62.1,
        "load_avg_1min": 2.3,
        "load_avg_5min": 1.8,
        "load_avg_15min": 1.5,
        "gpus": [
          {
            "utilization_percent": 85,
            "memory_used_mb": 30720,
            "memory_total_mb": 49152
          }
        ],
        "top_users": [
          {"user": "alice", "cpu_percent": 45.2},
          {"user": "bob", "cpu_percent": 12.1}
        ]
      }
    },
    {
      "name": "deadserver",
      "status": "offline",
      "error": "connection timed out"
    }
  ]
}
```

### Step 3: Present Results

**If no question was asked**, present a summary table showing:
- All servers grouped by status (online first, offline last)
- For online servers: CPU%, Memory%, per-GPU utilization and memory
- For offline servers: note them as unreachable
- Highlight servers with low utilization (CPU < 20%, no busy GPUs) as good candidates

**If a question was asked**, reason over the JSON data to answer it.
Common question types:

| Question Type | How to Answer |
|---------------|---------------|
| "Which server has free GPUs?" | Find online servers where GPU utilization < 50% and GPU memory has headroom |
| "Where should I run a training job?" | Find servers with lowest GPU utilization and most free GPU memory |
| "Is server X available?" | Check that specific server's status and metrics |
| "What's the GPU memory on Y?" | Report per-GPU memory used/total for that server |
| "Which servers are idle?" | Find servers with CPU < 20% and memory < 50% |
| "Who is using server X?" | Report top_users with their CPU percentages |

**Formatting guidelines:**
- Use a markdown table for multi-server summaries
- Convert GPU memory from MB to GB for readability (divide by 1024)
- Show percentages with one decimal place
- For GPU-focused questions, show per-GPU breakdown (not just averages)

### Step 4: Offer Follow-Up

After presenting results, briefly note that the user can ask follow-up questions like:
- "Which of these has the most free GPU memory?"
- "Run my job on the least-loaded GPU server"

## Running After Scouting

Once you've picked a host, three rules keep remote runs from silently breaking or clobbering a neighbor:

- **Never write `cd` in an ssh command string — pin the directory as a flag.**
  This is a *hard mechanical rule*, not advice: composing `ssh host "cd /long/path && <cmd>"`, the `cd /long/path && ` prefix gets silently dropped from what actually runs, so the command executes in `$HOME` (→ "not a git repository", the wrong project resolved, etc.).
  This recurs *every session* and prose warnings have not stopped it — the fix is structural: make the working directory an un-droppable argument.
  - **Single command → use the tool's dir flag:** `git -C <dir> …`, `uv run --directory <dir> …`, `make -C <dir> …`, `tar -C <dir> …`, `snakemake --directory <dir> …`.
  - **Multi-step → write a script on the remote and run *that*.**
    Inside a here-doc'd file, `cd` sits on its own line where it cannot be dropped:
    ```bash
    ssh host 'cat > ~/run.sh <<"EOF"
    #!/bin/bash -l
    cd /home/you/re/<repo>/experiments/<exp>
    uv run snakemake <rule> --cores N
    EOF'
    ssh host 'tmux new-session -d -s job "bash ~/run.sh > ~/job.log 2>&1"'
    ```
  - A bare inline `cd …` in the ssh argument string is *only* acceptable inside a here-doc'd script body or a `make remote-tmux` window — never as a prefix you hand-compose.
- **Scope the remote checkout to the clone corresponding to *your* repo.**
  Pass `REMOTE_DIR` explicitly on every `make remote-sync`/`remote-run`/`remote-tmux` call (or ssh to an explicit path).
  The config-derived default (`~/.config/dasm2/config.yaml`) is the *same directory for every clone/worktree on the box*, so concurrent runs overwrite each other's checkout — the rule `bip-conductor-spawn` states for EPIC slots applies to any remote run.
  Don't trust that default to be non-empty either: the Makefile resolves it with a bare `python3` that may lack the project's deps and silently return `""` (`cd ` → "not a git repository").
  **Before syncing, check the target clone isn't dirty with unrelated in-progress work** (`git -C <dir> status`); if it is, verify the changes are already in history (`git -C <dir> diff <pushed-HEAD>` empty) *before* any `reset --hard`, and never discard uncommitted work you didn't create.
- **Invoke through a login shell.**
  Use `make remote-tmux` (its tmux window is a login shell) or `ssh <host> 'bash -lc "…"'` — not a bare `ssh <host> "source .venv/bin/activate && …"`.
  A non-login shell skips lmod module init, so a venv Python built against a module-provided libpython dies with `libpython3.XX.so.1.0: cannot open shared object file`.

## Error Handling

- **bip not found**: Report error, suggest building with `go build -o bip ./cmd/bip`
- **servers.yml missing**: Report error, suggest creating config file
- **SSH failures**: Report which servers failed and why
- **No GPU servers**: Note that no servers are configured with `has_gpu: true`
