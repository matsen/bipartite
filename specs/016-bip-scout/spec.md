# Feature Specification: bip scout — Remote Server Availability

**Feature Branch**: `016-bip-scout`
**Created**: 2026-01-29
**Status**: Draft
**Input**: GitHub issue #70 — Add `bip scout` command for remote server CPU and GPU availability

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Check All Servers (Priority: P1)

A user (typically an AI agent or researcher) wants to know which remote servers are available and what their current CPU, memory, and GPU utilization looks like, so they can decide where to run a compute job.

**Why this priority**: This is the core use case — without it, nothing else matters.

**Independent Test**: Run `bip scout` from a nexus directory with a valid `servers.yml` and verify structured JSON output containing status, CPU, memory, load, and GPU data for each configured server.

**Acceptance Scenarios**:

1. **Given** a nexus directory with `servers.yml` listing multiple servers, **When** the user runs `bip scout`, **Then** the command outputs pretty-printed JSON with one entry per server containing status, CPU usage, memory usage, load averages, and GPU metrics (if applicable).
2. **Given** a server that is unreachable (SSH timeout), **When** the user runs `bip scout`, **Then** that server appears in the output with status "offline" and null metrics, while other servers report normally.
3. **Given** a server without GPUs, **When** the user runs `bip scout`, **Then** GPU fields are absent or null for that server.

---

### User Story 2 — Human-Readable Table Output (Priority: P2)

A user wants a quick visual summary of server availability in a terminal-friendly table format instead of JSON.

**Why this priority**: Convenience for interactive use — the data is the same as P1, just formatted differently.

**Independent Test**: Run `bip scout --human` and verify a well-aligned table with columns for server name, status, CPU, memory, load averages, GPU usage, and GPU memory.

**Acceptance Scenarios**:

1. **Given** a valid `servers.yml`, **When** the user runs `bip scout --human`, **Then** output is a formatted table with aligned columns matching the example in issue #70.
2. **Given** servers with varying GPU counts, **When** displayed in human format, **Then** GPU columns show aggregated usage (e.g., "100% (avg of 2)") or "-" for servers without GPUs.

---

### User Story 3 — Check a Single Server (Priority: P2)

A user wants to check status for just one specific server rather than all of them.

**Why this priority**: Same tier as human formatting — filters the core data rather than adding new capability.

**Independent Test**: Run `bip scout --server beetle01` and verify JSON output for only that server.

**Acceptance Scenarios**:

1. **Given** a valid `servers.yml` containing "beetle01", **When** the user runs `bip scout --server beetle01`, **Then** output is JSON for only that server.
2. **Given** a server name not in `servers.yml`, **When** the user runs `bip scout --server unknown`, **Then** the command exits with a clear error message.

---

### User Story 4 — Server Configuration via YAML (Priority: P1)

A user defines which servers to check and SSH connection parameters in a `servers.yml` file in their nexus directory.

**Why this priority**: Required for P1 — the command cannot run without knowing which servers to connect to.

**Independent Test**: Create a `servers.yml` with servers including pattern expansion (e.g., `beetle{01..05}`), run `bip scout`, and verify all expanded servers are checked.

**Acceptance Scenarios**:

1. **Given** a `servers.yml` with a `pattern: "beetle{01..05}"` entry, **When** the command loads config, **Then** it expands to servers beetle01, beetle02, beetle03, beetle04, and beetle05.
2. **Given** a `servers.yml` with `ssh.proxy_jump` and `ssh.connect_timeout` set, **When** connecting to servers, **Then** SSH connections route through the specified jump host with the specified timeout.
3. **Given** no `servers.yml` in the nexus directory, **When** the user runs `bip scout`, **Then** the command exits with a clear error explaining the missing config file.

---

### Edge Cases

- What happens when the SSH proxy/jump host itself is unreachable? The command should fail gracefully with a clear error indicating the proxy is down, not hang indefinitely.
- What happens when `nvidia-smi` is not installed on a server marked `has_gpu: true`? The server should report as online but with null/error GPU metrics.
- What happens when a server is reachable but a metric command (e.g., `top`) produces unexpected output? The command should report what it can parse and indicate errors for unparseable fields.
- What happens when all servers are offline? The command should output the full list with all servers marked offline, not produce an empty result.
- What happens when pattern expansion produces zero servers (e.g., malformed pattern)? The command should report a configuration error.
- What happens when SSH authentication fails (e.g., no SSH agent running, no keys in `~/.ssh/`, or `~/.ssh/config` not set up for the target hosts)? The command should produce a helpful error message explaining what's missing and how to fix it (e.g., "SSH authentication failed for beetle01: no keys found. Ensure your SSH agent is running and ~/.ssh/config is configured for this host.").

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `bip scout` CLI subcommand that checks remote server availability.
- **FR-002**: The system MUST output JSON by default (pretty-printed, 2-space indent), with a `--human` flag for table output.
- **FR-003**: The system MUST accept a `--server <name>` flag to check a single server.
- **FR-004**: The system MUST read server definitions from a `servers.yml` file in the nexus directory.
- **FR-005**: The `servers.yml` MUST support individual server entries with `name` and optional `has_gpu` fields.
- **FR-006**: The `servers.yml` MUST support pattern expansion (e.g., `beetle{01..05}` expands to beetle01 through beetle05).
- **FR-007**: The `servers.yml` MUST support SSH configuration including `proxy_jump` and `connect_timeout`.
- **FR-008**: The system MUST connect to servers using a native SSH client (not subprocess), supporting ProxyJump by dialing through the jump host.
- **FR-009**: The system MUST run all metric-gathering commands within a single SSH session per server to avoid connection flooding.
- **FR-010**: The system MUST limit concurrent SSH connections using a bounded semaphore (default: 5 concurrent connections).
- **FR-011**: The system MUST collect CPU usage, memory usage, load averages, and (when applicable) GPU utilization and GPU memory from each server.
- **FR-012**: The system MUST report unreachable servers as "offline" with null metrics rather than failing entirely.
- **FR-013**: The human-readable table MUST include columns: Server, Status, CPU, Memory, Load Avg, GPU Usage, GPU Memory.
- **FR-014**: GPU metrics MUST show aggregated values across all GPUs on a server (average utilization, total memory used/total).
- **FR-015**: The system MUST produce a helpful, actionable error message when SSH authentication fails, indicating what is missing (e.g., no SSH agent, no keys, missing `~/.ssh/config` entry) and how to fix it.
- **FR-016**: The system MUST provide a `/bip.scout` Claude Code slash command skill (`.claude/skills/bip.scout/SKILL.md`) that runs `bip scout` to collect JSON, presents a human-readable summary, and answers follow-up questions (e.g., "which server has free GPUs?") by reasoning over the data.

### Key Entities

- **Server**: A remote machine to check. Attributes: name, has_gpu flag, online/offline status.
- **ServerMetrics**: CPU usage percentage, memory usage percentage, load averages (1/5/15 min), optional GPU utilization and GPU memory.
- **ServerConfig**: The `servers.yml` file defining servers to check and SSH parameters.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `bip scout` returns complete status for all configured servers within 30 seconds (accounting for SSH timeouts on unreachable servers).
- **SC-002**: The command correctly reports metrics for all reachable servers — CPU, memory, and load averages always present; GPU metrics present when server has GPUs.
- **SC-003**: Unreachable servers appear in output with "offline" status and do not block reporting of other servers.
- **SC-004**: The `--human` flag produces a readable, aligned table that matches the format shown in issue #70.
- **SC-005**: The command replaces the existing `mcp-compute-scout` Python MCP server, providing equivalent data with lower operational overhead (no long-running process, no venv management).

## Clarifications

### Session 2026-01-29

- Q: How should the tool discover SSH keys? → A: Use system SSH agent + `~/.ssh/config` (standard SSH key discovery), no additional config fields needed.

## Assumptions

- Users have SSH key-based authentication configured for the jump host and target servers via their system SSH agent and `~/.ssh/config` (no password prompts, no tool-specific key configuration).
- The remote servers run Linux with standard `top`, `free`, `uptime`, and (for GPU servers) `nvidia-smi` commands available.
- The nexus directory path is already known to bip via its existing configuration mechanism.
- The `servers.yml` file is manually maintained by the user (no auto-discovery of servers).
