# CLI Contract: bip scout

## Command

```
bip scout [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--server` | string | (all) | Check a specific server by name |
| `--human` | bool | false | Human-readable table output (inherited from root) |

## Exit Codes

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | ExitSuccess | All checks completed (some servers may be offline) |
| 1 | ExitError | System-level failure (SSH agent missing, etc.) |
| 2 | ExitConfigError | Configuration error (missing servers.yml, malformed YAML, unknown server name) |

## JSON Output Schema

### Success (exit 0)

```json
{
  "servers": [
    {
      "name": "string",
      "status": "online | offline",
      "error": "string (omitted if online)",
      "metrics": {
        "cpu_percent": 0.0,
        "memory_percent": 0.0,
        "load_avg_1min": 0.0,
        "load_avg_5min": 0.0,
        "load_avg_15min": 0.0,
        "gpus": [
          {
            "utilization_percent": 0,
            "memory_used_mb": 0,
            "memory_total_mb": 0
          }
        ]
      }
    }
  ]
}
```

Field rules:
- `status`: Always present. One of `"online"` or `"offline"`.
- `error`: Present only when `status` is `"offline"`. Contains human-readable error message.
- `metrics`: Present only when `status` is `"online"`. Contains all metric fields.
- `gpus`: Present only when server `has_gpu: true` AND GPU commands succeeded. Array with one entry per GPU.
- `metrics` fields `cpu_percent`, `memory_percent`, `load_avg_*`: Always present when `metrics` is present.

### Error (exit 1 or 2)

```json
{
  "error": "descriptive error message"
}
```

## Human Output Format

```
Server    Status  CPU     Memory  Load Avg           GPU Usage         GPU Memory
<name>    online  <n>%    <n>%    <n>, <n>, <n>      <n>% (avg of <k>) <n>% (<used>/<total> MB)
<name>    offline -       -       -                   -                 -
```

Column rules:
- `GPU Usage`: Average utilization across all GPUs, with count. `-` if no GPUs.
- `GPU Memory`: Total used / total capacity across all GPUs, as percentage and absolute. `-` if no GPUs.
- Offline servers show `-` for all metric columns.

## Configuration

### servers.yml (nexus directory)

```yaml
servers:
  - name: <hostname>          # Individual server
    has_gpu: <bool>           # Optional, default false
  - pattern: "<prefix>{<NN>..<MM>}"  # Pattern expansion
    has_gpu: <bool>

ssh:
  proxy_jump: <jump_host>    # Optional ProxyJump host
  connect_timeout: <seconds> # Optional, default 10
```

### Validation Rules

- `servers.yml` must exist in the current working directory (nexus directory)
- Each server entry must have exactly one of `name` or `pattern`
- Pattern must match format `<prefix>{<start>..<end>}` where start ≤ end
- `--server` value must match an expanded server name
- SSH agent must be running with keys loaded
