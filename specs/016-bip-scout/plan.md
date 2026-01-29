# Implementation Plan: bip scout — Remote Server Availability

**Branch**: `016-bip-scout` | **Date**: 2026-01-29 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/016-bip-scout/spec.md`

## Summary

Add a `bip scout` CLI subcommand that checks remote server CPU, memory, load, and GPU availability via native SSH connections. Replaces the existing `mcp-compute-scout` Python MCP server with a single Go binary command. Reads server definitions from `servers.yml` in the nexus directory, connects in parallel with bounded concurrency, and outputs JSON (default) or a human-readable table (`--human`).

## Technical Context

**Language/Version**: Go 1.24.1 (from go.mod)
**Primary Dependencies**: spf13/cobra (CLI), golang.org/x/crypto/ssh (native SSH), gopkg.in/yaml.v3 (config parsing)
**Storage**: N/A — stateless command, no persistence
**Testing**: `go test ./...` with unit tests for parsing/config and integration tests for SSH (using testable interfaces)
**Target Platform**: macOS + Linux (same as existing bip)
**Project Type**: Single — extends existing Go CLI
**Performance Goals**: Complete all server checks within 30 seconds (SC-001), including SSH timeouts for unreachable servers
**Constraints**: Max 5 concurrent SSH connections through proxy (FR-010); single SSH session per server (FR-009)
**Scale/Scope**: ~7 servers (2 individual + 5 from pattern expansion) — small scale, no need for sophisticated scheduling

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Agent-First Design | **PASS** | JSON default output, `--human` flag for table, CLI-only interface, composable |
| II. Git-Versionable Architecture | **PASS** | No persistent state — `servers.yml` is user-maintained config. No database files |
| III. Fail-Fast Philosophy | **PASS** | Missing config → immediate error; SSH failures → "offline" status per server; no silent defaults |
| IV. Real Testing | **PASS** | Unit tests with real parsing fixtures; SSH integration via testable interface (not mocks — real parsing of actual command output) |
| V. Clean Architecture | **PASS** | Separate packages: config loading, SSH connection, metrics parsing, table formatting. Clear entity names |
| VI. Simplicity | **PASS** | Minimal deps (ssh + yaml only). No caching, no daemon, no abstraction layers. Hardcoded commands per spec |
| CLI Responsiveness | **PASS** | Compiled Go binary, no startup overhead |
| Embeddable Over Client-Server | **PASS** | No server process — direct CLI command |
| Data Portability | **PASS** | YAML config, JSON output — both human-readable |
| Platform Support | **PASS** | `golang.org/x/crypto/ssh` works on macOS + Linux |

**Post-Phase-1 Re-check**: No violations. Design stays within constitution bounds.

## Project Structure

### Documentation (this feature)

```text
specs/016-bip-scout/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── cli.md           # CLI contract (flags, exit codes, output schemas)
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/bip/
└── scout.go             # Cobra command: flags, validation, output dispatch

internal/scout/
├── config.go            # servers.yml loading, pattern expansion, validation
├── config_test.go       # Config parsing tests with fixture YAML
├── ssh.go               # Native SSH client: dial, ProxyJump, session management
├── ssh_test.go          # SSH unit tests (agent auth, key discovery)
├── metrics.go           # Remote command execution + output parsing
├── metrics_test.go      # Parser tests with real command output fixtures
├── format.go            # Human-readable table formatting
├── format_test.go       # Table formatting tests
└── types.go             # ServerConfig, ServerMetrics, ScoutResult types
```

**Structure Decision**: Follows existing pattern — Cobra command in `cmd/bip/`, business logic in `internal/scout/`. No new top-level directories.

## Complexity Tracking

No violations to justify. The design is straightforward:
- One new Cobra subcommand
- One new internal package with clear separation of concerns
- Two new dependencies (`golang.org/x/crypto/ssh`, `gopkg.in/yaml.v3`) — both standard Go ecosystem libraries

---

## Phase 0: Research

### Research 1: Native SSH with ProxyJump in Go

**Decision**: Use `golang.org/x/crypto/ssh` with manual ProxyJump implementation.

**Rationale**: The spec explicitly requires native SSH (FR-008) to avoid subprocess argument-splitting bugs and dependency on the system `ssh` binary. ProxyJump is implemented by first dialing the jump host, then dialing the target through the established connection.

**Implementation Pattern**:
```go
// 1. Connect to jump host
jumpConn, err := ssh.Dial("tcp", jumpHost+":22", jumpConfig)

// 2. Dial target through jump host
targetConn, err := jumpConn.Dial("tcp", targetHost+":22")

// 3. Create SSH client on target connection
targetClient, err := ssh.NewClientConn(targetConn, targetHost+":22", targetConfig)
```

**SSH Key Discovery**: Use `ssh-agent` via `golang.org/x/crypto/ssh/agent` — connect to `SSH_AUTH_SOCK` Unix socket. This matches the spec assumption that users have SSH agent configured. Also parse `~/.ssh/config` for per-host settings using a lightweight parser or by reading key files directly from `~/.ssh/`.

**Alternatives considered**:
- `os/exec` subprocess (`ssh` command): Rejected per FR-008; prone to argument injection, no ProxyJump control
- `github.com/kevinburke/ssh_config`: Lightweight SSH config parser — considered for `~/.ssh/config` parsing but adds a dependency for something we can handle with agent auth alone

### Research 2: YAML Parsing in Go

**Decision**: Use `gopkg.in/yaml.v3` for `servers.yml` parsing.

**Rationale**: The project currently uses JSON exclusively, but the spec and issue #70 explicitly define `servers.yml` as YAML. `gopkg.in/yaml.v3` is the standard Go YAML library — stable, well-maintained, no CGO.

**Alternatives considered**:
- JSON config: Would conflict with the spec's YAML requirement and future consistency plans (#69)
- `github.com/goccy/go-yaml`: Faster but less standard; premature optimization for a tiny config file

### Research 3: Remote Command Parsing Strategy

**Decision**: Hardcode the five metric commands in Go and parse their output directly. No user-configurable commands.

**Rationale**: The issue explicitly states: "The remote commands (`top`, `free`, `uptime`, `nvidia-smi`) are hardcoded in Go — they're standard Linux tooling with no reason to be user-configurable." The existing Python version has configurable commands in YAML, but the Go version simplifies by removing this indirection.

**Commands**:
| Metric | Command | Parse Strategy |
|--------|---------|---------------|
| CPU usage | `top -bn1 \| grep -i "cpu(s)" \| awk '{print $2}' \| cut -d'%' -f1` | Parse single float |
| Memory usage | `free -m \| awk '/^Mem:/ {printf "%.1f", ($3/$2) * 100}'` | Parse single float |
| Load average | `uptime \| awk -F'load average:' '{print $2}' \| sed 's/^[[:space:]]*//'` | Parse 3 comma-separated floats |
| GPU utilization | `nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits` | Parse one int per line |
| GPU memory | `nvidia-smi --query-gpu=memory.used,memory.total --format=csv,noheader,nounits` | Parse two ints per line |

**Execution strategy**: Combine all commands for a server into a single SSH session using `___SCOUT_DELIM___` separators (matching the existing Python pattern). This satisfies FR-009 (single session per server).

### Research 4: Pattern Expansion

**Decision**: Implement brace expansion for `{NN..MM}` patterns in Go.

**Rationale**: The spec requires expanding patterns like `beetle{01..05}` → `beetle01, beetle02, ..., beetle05`. The existing Python implementation uses a regex `(.+)\{(\d+)\.\.(\d+)\}` and generates zero-padded names based on the start value's padding.

**Implementation**: Simple regex match → extract prefix, start, end → iterate with zero-padded format string. No need for full bash brace expansion — only numeric ranges with zero-padding.

### Research 5: SSH Authentication Error Reporting

**Decision**: Detect specific SSH error types and produce actionable error messages per FR-015.

**Error scenarios**:
| Condition | Detection | Message |
|-----------|-----------|---------|
| No SSH agent | `SSH_AUTH_SOCK` not set or socket missing | "SSH agent not running. Start with `eval $(ssh-agent)` and add keys with `ssh-add`" |
| Agent has no keys | Agent returns empty key list | "SSH agent has no keys. Add keys with `ssh-add`" |
| Auth rejected | `ssh.Dial` returns auth error | "SSH authentication failed for {server}. Check ~/.ssh/config and ensure your key is authorized" |
| Proxy unreachable | `ssh.Dial` to jump host times out | "Cannot reach proxy {proxy}: connection timed out" |
| Server unreachable | Target dial through proxy times out | Server marked "offline" with null metrics |

---

## Phase 1: Design

### Data Model

#### ServerConfig (from servers.yml)

```go
// ScoutConfig represents the top-level servers.yml structure.
type ScoutConfig struct {
    Servers []ServerEntry `yaml:"servers"`
    SSH     SSHConfig     `yaml:"ssh"`
}

// ServerEntry is a single entry in servers.yml (either name or pattern).
type ServerEntry struct {
    Name    string `yaml:"name,omitempty"`
    Pattern string `yaml:"pattern,omitempty"`
    HasGPU  bool   `yaml:"has_gpu,omitempty"`
}

// SSHConfig holds SSH connection parameters.
type SSHConfig struct {
    ProxyJump      string `yaml:"proxy_jump,omitempty"`
    ConnectTimeout int    `yaml:"connect_timeout,omitempty"` // seconds, default 10
}

// Server is an expanded, resolved server ready to check.
type Server struct {
    Name   string
    HasGPU bool
}
```

#### ServerResult (output)

```go
// ScoutResult is the top-level JSON output.
type ScoutResult struct {
    Servers []ServerStatus `json:"servers"`
}

// ServerStatus is one server's check result.
type ServerStatus struct {
    Name    string         `json:"name"`
    Status  string         `json:"status"` // "online" or "offline"
    Error   string         `json:"error,omitempty"`
    Metrics *ServerMetrics `json:"metrics,omitempty"`
}

// ServerMetrics holds parsed metric values.
type ServerMetrics struct {
    CPUPercent    float64    `json:"cpu_percent"`
    MemoryPercent float64    `json:"memory_percent"`
    LoadAvg1      float64    `json:"load_avg_1min"`
    LoadAvg5      float64    `json:"load_avg_5min"`
    LoadAvg15     float64    `json:"load_avg_15min"`
    GPUs          []GPUInfo  `json:"gpus,omitempty"`
}

// GPUInfo holds per-GPU metrics.
type GPUInfo struct {
    UtilizationPercent int `json:"utilization_percent"`
    MemoryUsedMB       int `json:"memory_used_mb"`
    MemoryTotalMB      int `json:"memory_total_mb"`
}
```

### CLI Contract

```
bip scout [flags]

Flags:
  --server string   Check a specific server (must match a name in servers.yml)
  --human           Use human-readable table output (inherited from root)

Exit codes:
  0  Success
  1  General error (SSH system failure, etc.)
  2  Config error (missing servers.yml, malformed YAML, unknown --server name)

JSON output (default):
  {
    "servers": [
      {
        "name": "beetle01",
        "status": "online",
        "metrics": {
          "cpu_percent": 12.3,
          "memory_percent": 45.6,
          "load_avg_1min": 0.52,
          "load_avg_5min": 0.48,
          "load_avg_15min": 0.41,
          "gpus": [
            {"utilization_percent": 75, "memory_used_mb": 8192, "memory_total_mb": 16384}
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

Human output (--human):
  Server    Status  CPU     Memory  Load Avg           GPU Usage         GPU Memory
  mantis    online  9.8%    26.1%   5.41, 5.43, 5.20   100% (avg of 2)   87% (35412/40960 MB)
  cricket    online  0.9%    3.7%    1.23, 1.19, 1.18   -                 -
  beetle01    online  0.1%    0.9%    0.00, 0.14, 0.09   0% (avg of 4)     0% (16/184272 MB)
  beetle03    offline -       -       -                   -                 -
```

### Quickstart

After implementation:

```bash
# 1. Create servers.yml in nexus directory
cat > ~/re/nexus/servers.yml << 'EOF'
servers:
  - name: mantis
    has_gpu: true
  - name: cricket
  - pattern: "beetle{01..05}"
    has_gpu: true

ssh:
  proxy_jump: jumphost.example.org
  connect_timeout: 10
EOF

# 2. Build bip
cd ~/re/bipartite
go build -o bip ./cmd/bip

# 3. Run from nexus directory
cd ~/re/nexus
bip scout              # JSON output
bip scout --human      # Table output
bip scout --server beetle01  # Single server
```

### Key Design Decisions

1. **No caching**: The Python MCP server had a 30-second TTL cache because it was a long-running process. `bip scout` is a one-shot CLI command — caching is meaningless.

2. **No configurable commands**: Commands are hardcoded in Go. The Python version allowed YAML-defined commands, but the issue explicitly says to drop this.

3. **No `--find`/`--free` flags**: The issue explicitly states filtering logic belongs in the skill layer, not in Go. The CLI is a data reporter only.

4. **Per-GPU detail in JSON, aggregated in human table**: JSON preserves per-GPU granularity (array of GPUInfo). The human table aggregates (average utilization, summed memory) for readability.

5. **`servers.yml` in nexus directory**: The config lives alongside `sources.json`, `config.json`, and other nexus-level configuration. Discovered via the same nexus directory validation pattern used by `checkin`, `board`, etc.

6. **SSH agent only — no key_file config**: The spec and clarifications say to use standard SSH agent + `~/.ssh/config`. We don't add `username`, `key_file`, or `options` fields to `servers.yml` (simplification from the Python version). The native Go SSH client connects to the SSH agent via `SSH_AUTH_SOCK`.

7. **No `host` field in server config**: Server name IS the hostname (matching SSH config entries). The Python version had separate `name` and `host` fields; we simplify since `~/.ssh/config` handles hostname mapping.
