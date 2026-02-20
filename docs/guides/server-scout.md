# Server Scout

Check remote server CPU, memory, load, and GPU availability via SSH.

## Quick Start

```bash
bip scout --human          # Human-readable table
bip scout                  # JSON output (default)
bip scout --server orca42  # Check single server
```

Requires `nexus_path` configured in `~/.config/bip/config.yml` and `servers.yml` in your nexus directory.

## Configuration

Create `servers.yml` to define your servers:

```yaml
servers:
  # Individual servers
  - name: ermine
  - name: orca42
    has_gpu: true

  # Brace expansion for numbered servers
  - pattern: "beetle{01..05}"
    has_gpu: true

ssh:
  proxy_jump: login.example.com   # Optional jump host
  connect_timeout: 10             # Seconds (default: 10)
```

### Server Entries

Each entry must have either `name` or `pattern` (not both):

| Field | Description |
|-------|-------------|
| `name` | Single server hostname |
| `pattern` | Brace expansion like `node{01..10}` |
| `has_gpu` | Set `true` to collect GPU metrics via `nvidia-smi` |

Patterns preserve zero-padding: `gpu{01..12}` expands to `gpu01`, `gpu02`, ..., `gpu12`.

### SSH Settings

| Field | Default | Description |
|-------|---------|-------------|
| `proxy_jump` | (none) | Jump host for servers behind a bastion |
| `connect_timeout` | 10 | Connection timeout in seconds |

Authentication uses your SSH agent (`SSH_AUTH_SOCK`). Ensure keys are loaded with `ssh-add`.

#### SSH Username Resolution

`bip scout` reads `~/.ssh/config` to determine the SSH username for each connection. This is important when your remote username differs from your local OS username. The resolution order is:

1. **Per-server**: If `~/.ssh/config` has a `Host` block matching the server name with a `User` directive, that username is used.
2. **Proxy host**: If no per-server match, the `User` from the `Host` block matching the `proxy_jump` hostname is used.
3. **OS user**: Falls back to your local OS username.

**Important:** The `Host` value in `~/.ssh/config` must match the hostname used in `servers.yml` exactly. SSH aliases won't resolve — for example, if `servers.yml` has `proxy_jump: snail.fhcrc.org`, then `~/.ssh/config` needs `Host snail.fhcrc.org` (not `Host snail`). To support both, use multiple patterns:

```
Host snail snail.fhcrc.org
    User jdoe
    HostName snail.fhcrc.org
```

## Output

### Human-Readable (`--human`)

```
Server    Status  CPU    Mem    GPUs              GPU Mem (GB)      Users
--------  ------  -----  -----  ----------------  ----------------  ----------------
orca42    online  12.3%  45.6%  10%  5%  0%  0%   8/48 8/48 0/48 0/48  alice(45%) bob(12%)
beetle01  online  89.2%  78.1%  95% 92%           40/48 38/48       charlie(156%)
ermine    online   5.1%  22.0%
beetle02  offline
```

- Sorted by availability (least-loaded first, offline last)
- GPU columns right-aligned, one sub-column per GPU
- Top CPU users shown with aggregated CPU percentage

### JSON (default)

```json
{
  "servers": [
    {
      "name": "orca42",
      "status": "online",
      "metrics": {
        "cpu_percent": 12.3,
        "memory_percent": 45.6,
        "load_avg_1min": 2.1,
        "load_avg_5min": 1.8,
        "load_avg_15min": 1.5,
        "gpus": [
          {"utilization_percent": 10, "memory_used_mb": 8192, "memory_total_mb": 49152}
        ],
        "top_users": [
          {"user": "alice", "cpu_percent": 45.2}
        ]
      }
    },
    {
      "name": "beetle02",
      "status": "offline",
      "error": "connection timed out"
    }
  ]
}
```

## Claude Code Skill

Use `/bip.scout` for interactive server queries:

```
/bip.scout                           # Full summary
/bip.scout which server has free GPUs?
/bip.scout where should I run a training job?
```

The skill runs `bip scout`, parses the JSON, and answers questions by reasoning over the data.

## Troubleshooting

| Error | Solution |
|-------|----------|
| "SSH agent not running" | Run `eval $(ssh-agent)` and `ssh-add` |
| "SSH agent has no keys" | Run `ssh-add` to load your keys |
| "connection timed out" | Check network/VPN, increase `connect_timeout` |
| "SSH authentication failed for X (as user Y)" | The error shows which username was attempted. If wrong, add `User yourname` to the matching `Host` block in `~/.ssh/config` |
| "connection refused" | Ensure SSH is running on the server |

## How It Works

Scout connects to each server in parallel (max 5 concurrent) and runs:

```bash
# Top CPU users (>1% usage)
ps -eo user:20,%cpu --no-headers | awk '{cpu[$1]+=$2} END {for (u in cpu) if (cpu[u]>1.0) printf "%s %.1f\n",u,cpu[u]}' | sort -k2 -rn

# CPU usage
top -bn1 | grep -i "cpu(s)" | awk '{print $2}' | cut -d'%' -f1

# Memory usage
free -m | awk '/^Mem:/ {printf "%.1f", ($3/$2) * 100}'

# Load average
uptime | awk -F'load average:' '{print $2}' | sed 's/^[[:space:]]*//'

# GPU metrics (only for has_gpu: true servers)
nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits
nvidia-smi --query-gpu=memory.used,memory.total --format=csv,noheader,nounits
```

GPU metrics are only collected for servers with `has_gpu: true`. Failed commands are non-fatal — partial output is still parsed.
