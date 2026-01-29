# Quickstart: bip scout

## Prerequisites

- SSH agent running with keys loaded (`ssh-add -l` shows keys)
- `~/.ssh/config` configured for target servers (including ProxyJump if needed)
- Go 1.24+ installed

## Setup

### 1. Create `servers.yml` in your nexus directory

```yaml
servers:
  - name: mantis
    has_gpu: true
  - name: cricket
  - pattern: "beetle{01..05}"
    has_gpu: true

ssh:
  proxy_jump: jumphost.example.org
  connect_timeout: 10
```

### 2. Build

```bash
cd ~/re/bipartite
go build -o bip ./cmd/bip
```

### 3. Run

```bash
cd ~/re/nexus

# JSON output (default)
bip scout

# Human-readable table
bip scout --human

# Single server
bip scout --server beetle01
```

## Example Output

### JSON

```json
{
  "servers": [
    {
      "name": "mantis",
      "status": "online",
      "metrics": {
        "cpu_percent": 9.8,
        "memory_percent": 26.1,
        "load_avg_1min": 5.41,
        "load_avg_5min": 5.43,
        "load_avg_15min": 5.20,
        "gpus": [
          {"utilization_percent": 100, "memory_used_mb": 17706, "memory_total_mb": 20480},
          {"utilization_percent": 100, "memory_used_mb": 17706, "memory_total_mb": 20480}
        ]
      }
    }
  ]
}
```

### Human Table

```
Server    Status  CPU     Memory  Load Avg           GPU Usage         GPU Memory
mantis    online  9.8%    26.1%   5.41, 5.43, 5.20   100% (avg of 2)   87% (35412/40960 MB)
cricket    online  0.9%    3.7%    1.23, 1.19, 1.18   -                 -
beetle01    online  0.1%    0.9%    0.00, 0.14, 0.09   0% (avg of 4)     0% (16/184272 MB)
```

## Troubleshooting

- **"SSH agent not running"**: Run `eval $(ssh-agent) && ssh-add`
- **"Cannot reach proxy"**: Check VPN connection and that the proxy host is accessible
- **Server shows "offline"**: Verify `ssh <servername>` works from your terminal
- **"servers.yml not found"**: Run from your nexus directory (contains `sources.json`)
