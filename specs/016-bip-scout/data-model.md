# Data Model: bip scout

## Configuration Entities

### ScoutConfig (servers.yml)

Top-level YAML structure.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| servers | []ServerEntry | yes | List of server definitions |
| ssh | SSHConfig | no | SSH connection parameters |

### ServerEntry

Individual or pattern-based server definition.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | either name or pattern | Server hostname |
| pattern | string | either name or pattern | Brace expansion pattern (e.g. `beetle{01..05}`) |
| has_gpu | bool | no | Whether server has NVIDIA GPUs (default: false) |

**Validation**: Exactly one of `name` or `pattern` must be set. Pattern must match `\w+\{\d+\.\.\d+\}`.

### SSHConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| proxy_jump | string | no | (none) | Jump host for ProxyJump |
| connect_timeout | int | no | 10 | SSH timeout in seconds |

### Server (expanded)

After pattern expansion, each server resolves to:

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Server hostname |
| HasGPU | bool | Whether to run GPU commands |

## Output Entities

### ScoutResult

| Field | Type | Description |
|-------|------|-------------|
| servers | []ServerStatus | One entry per server |

### ServerStatus

| Field | Type | Description |
|-------|------|-------------|
| name | string | Server hostname |
| status | string | "online" or "offline" |
| error | string | Error message (omitted if online) |
| metrics | *ServerMetrics | Metrics (omitted if offline) |

### ServerMetrics

| Field | Type | Description |
|-------|------|-------------|
| cpu_percent | float64 | CPU usage percentage |
| memory_percent | float64 | Memory usage percentage |
| load_avg_1min | float64 | 1-minute load average |
| load_avg_5min | float64 | 5-minute load average |
| load_avg_15min | float64 | 15-minute load average |
| gpus | []GPUInfo | Per-GPU metrics (omitted if no GPUs) |

### GPUInfo

| Field | Type | Description |
|-------|------|-------------|
| utilization_percent | int | GPU utilization (0-100) |
| memory_used_mb | int | GPU memory used in MB |
| memory_total_mb | int | GPU memory total in MB |

## State Transitions

None — `bip scout` is stateless. No persistent data, no state machine.

## Relationships

```
servers.yml → [parse] → []Server → [SSH check] → []ServerStatus → [format] → JSON or table
```
