package scout

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// delimiter separates command outputs in a single SSH session.
const delimiter = "___SCOUT_DELIM___"

// maxConcurrent is the bounded semaphore size for parallel server checks.
const maxConcurrent = 5

// procSampleSeconds is the gap between the two /proc snapshots taken to measure
// per-user CPU. It must match the `sleep` in topUsersCmd (both derive from it).
const procSampleSeconds = 0.5

// userCPUThreshold is the minimum per-user CPU percent (100 == one full core) to
// report, dropping the long tail of near-idle accounts.
const userCPUThreshold = 1.0

// procSnapMarker separates the three blocks of topUsersCmd's output: the header
// + uid→name map, then the two /proc snapshots. It must not contain the section
// delimiter, and parseProcUserCPU splits the section on it.
const procSnapMarker = "@@SCOUT_SNAP@@"

// topUsersCmd dumps the raw data needed to compute true instantaneous per-user
// CPU, doing no arithmetic on the remote host — parseProcUserCPU does that in Go.
// It emits, in one section:
//
//	CLK <clk_tck> SID <our_session> SSHD <serving_sshd_pid>
//	<user> /proc/<pid>        (one per process, full LDAP names via stat(1))
//	@@SCOUT_SNAP@@
//	<contents of every /proc/<pid>/stat>     (snapshot 1)
//	@@SCOUT_SNAP@@
//	<contents of every /proc/<pid>/stat>     (snapshot 2, taken sleep later)
//
// Sampling /proc/<pid>/stat twice and differencing (utime+stime) jiffies gives a
// current reading (100% == one core, matching ps's %cpu unit) without ps's two
// failure modes: ps %cpu is a lifetime average, so (a) the short-lived sshd the
// SSH session spawns reports a large % from a tiny elapsed-time denominator,
// summing to a spurious per-user floor, and (b) a process that ran hot hours ago
// but is now idle keeps over-reporting. `stat -c %U` is used for names rather than
// top's USER column because top truncates to 8 chars + '+', merging distinct users
// (systemd-resolve/systemd-timesync both become "systemd+").
//
// SID and SSHD identify scout's own footprint so parseProcUserCPU can exclude it:
// SID (our login session — field 6 of the shell's own /proc/$$/stat) covers the
// shell and the cat/stat pipeline, and SSHD ($PPID) is the sshd serving the
// connection. That sshd sits in a *different* session, so SID alone misses it, yet
// the CPU it spends streaming this (sizeable) output back would otherwise be
// mis-charged to the ssh user.
var topUsersCmd = fmt.Sprintf(
	`echo "CLK $(getconf CLK_TCK) SID $(cut -d' ' -f6 /proc/$$/stat) SSHD $PPID"; `+
		`stat -c '%%U %%n' /proc/[0-9]* 2>/dev/null; echo '%s'; `+
		`cat /proc/[0-9]*/stat 2>/dev/null; echo '%s'; `+
		`sleep %.1f; cat /proc/[0-9]*/stat 2>/dev/null`,
	procSnapMarker, procSnapMarker, procSampleSeconds,
)

// Section indices for ParseMetrics output splitting.
// These must match the command order in BuildCommand.
const (
	sectionTopUsers = 0
	sectionCPU      = 1
	sectionMemory   = 2
	sectionLoadAvg  = 3
	sectionGPUUtil  = 4
	sectionGPUMem   = 5
)

// BuildCommand constructs the combined command string for a server.
// All metric commands are joined with delimiters for single-session execution.
func BuildCommand(server Server) string {
	cmds := []string{
		// Top CPU users — raw /proc snapshots; parseProcUserCPU does the math.
		topUsersCmd,
		// CPU usage. Two iterations 0.5s apart so top reports a real delta; its
		// first iteration has no prior sample and would report since-boot stats.
		`top -bn2 -d 0.5 | grep -i "cpu(s)" | tail -1 | awk '{print $2}' | cut -d'%' -f1`,
		// Memory usage
		`free -m | awk '/^Mem:/ {printf "%.1f", ($3/$2) * 100}'`,
		// Load average
		`uptime | awk -F'load average:' '{print $2}' | sed 's/^[[:space:]]*//'`,
	}

	if server.HasGPU {
		cmds = append(cmds,
			// GPU utilization
			`nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits`,
			// GPU memory
			`nvidia-smi --query-gpu=memory.used,memory.total --format=csv,noheader,nounits`,
		)
	}

	// Join with echo delimiter between each command
	parts := make([]string, 0, len(cmds)*2-1)
	for i, cmd := range cmds {
		if i > 0 {
			parts = append(parts, fmt.Sprintf("echo '%s'", delimiter))
		}
		parts = append(parts, cmd)
	}
	return strings.Join(parts, " ; ")
}

// parseFloatMetric parses a float value with a descriptive error message.
func parseFloatMetric(value, metricName string) (float64, error) {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w (raw: %q)", metricName, err, value)
	}
	return result, nil
}

// ParseMetrics parses the combined output of all metric commands for a server.
func ParseMetrics(output string, hasGPU bool) (*ServerMetrics, error) {
	sections := strings.Split(output, delimiter)

	// Clean up whitespace in each section
	for i := range sections {
		sections[i] = strings.TrimSpace(sections[i])
	}

	expectedSections := 4
	if hasGPU {
		expectedSections = 6
	}
	if len(sections) < expectedSections {
		return nil, fmt.Errorf("expected %d metric sections, got %d", expectedSections, len(sections))
	}

	metrics := &ServerMetrics{}

	// Parse top users — non-fatal on failure
	// (the snapshot data identifies and excludes scout's own process group)
	if users, err := parseProcUserCPU(sections[sectionTopUsers]); err == nil {
		metrics.TopUsers = users
	}

	// Parse CPU
	cpu, err := parseFloatMetric(sections[sectionCPU], "CPU")
	if err != nil {
		return nil, err
	}
	metrics.CPUPercent = cpu

	// Parse Memory
	mem, err := parseFloatMetric(sections[sectionMemory], "memory")
	if err != nil {
		return nil, err
	}
	metrics.MemoryPercent = mem

	// Parse Load Average (format: "0.52, 0.48, 0.41")
	loadParts := strings.Split(sections[sectionLoadAvg], ",")
	if len(loadParts) < 3 {
		return nil, fmt.Errorf("parsing load average: expected 3 values, got %d (raw: %q)", len(loadParts), sections[sectionLoadAvg])
	}
	metrics.LoadAvg1, err = parseFloatMetric(strings.TrimSpace(loadParts[0]), "load avg 1min")
	if err != nil {
		return nil, err
	}
	metrics.LoadAvg5, err = parseFloatMetric(strings.TrimSpace(loadParts[1]), "load avg 5min")
	if err != nil {
		return nil, err
	}
	metrics.LoadAvg15, err = parseFloatMetric(strings.TrimSpace(loadParts[2]), "load avg 15min")
	if err != nil {
		return nil, err
	}

	// Parse GPU metrics if applicable.
	// GPU parse failures are non-fatal per spec: "server should report as online
	// but with null/error GPU metrics" when nvidia-smi is unavailable.
	if hasGPU {
		gpuUtils, err := parseGPUUtilization(sections[sectionGPUUtil])
		if err != nil {
			// Return metrics without GPU data
			return metrics, nil
		}
		gpuMems, err := parseGPUMemory(sections[sectionGPUMem])
		if err != nil {
			return metrics, nil
		}

		if len(gpuUtils) != len(gpuMems) {
			// Mismatch — return metrics without GPU data
			return metrics, nil
		}

		gpus := make([]GPUInfo, len(gpuUtils))
		for i := range gpuUtils {
			gpus[i] = GPUInfo{
				UtilizationPercent: gpuUtils[i],
				MemoryUsedMB:       gpuMems[i][0],
				MemoryTotalMB:      gpuMems[i][1],
			}
		}
		metrics.GPUs = gpus
	}

	return metrics, nil
}

// parseGPUUtilization parses nvidia-smi utilization output (one int per line).
func parseGPUUtilization(output string) ([]int, error) {
	lines := splitNonEmpty(output)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no GPU utilization data")
	}

	vals := make([]int, len(lines))
	for i, line := range lines {
		v, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return nil, fmt.Errorf("parsing GPU utilization line %d: %w (raw: %q)", i, err, line)
		}
		vals[i] = v
	}
	return vals, nil
}

// parseGPUMemory parses nvidia-smi memory output (two ints per line: used, total).
func parseGPUMemory(output string) ([][2]int, error) {
	lines := splitNonEmpty(output)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no GPU memory data")
	}

	vals := make([][2]int, len(lines))
	for i, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("parsing GPU memory line %d: expected 2 values, got %d (raw: %q)", i, len(parts), line)
		}
		used, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("parsing GPU memory used line %d: %w (raw: %q)", i, err, parts[0])
		}
		total, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("parsing GPU memory total line %d: %w (raw: %q)", i, err, parts[1])
		}
		vals[i] = [2]int{used, total}
	}
	return vals, nil
}

// procStat is the subset of a /proc/<pid>/stat line we care about.
type procStat struct {
	session string
	jiffies int64 // utime + stime, in clock ticks
}

// parseProcUserCPU turns topUsersCmd's output (header + uid→name map + two
// /proc/<pid>/stat snapshots, separated by procSnapMarker) into per-user CPU.
// For each process present in both snapshots and not part of scout's own session
// or serving sshd, it differences the (utime+stime) jiffies; the per-user totals
// become a percentage where 100 == one fully-used core (delta / (clkTck * dt) *
// 100), matching ps's %cpu unit. Users at or below userCPUThreshold are dropped,
// and results are sorted by CPU descending.
func parseProcUserCPU(section string) ([]UserCPU, error) {
	blocks := strings.Split(section, procSnapMarker)
	if len(blocks) != 3 {
		return nil, fmt.Errorf("expected 3 blocks separated by %q, got %d", procSnapMarker, len(blocks))
	}

	clkTck, ownSession, sshdPID, pidUser, err := parseProcHeader(blocks[0])
	if err != nil {
		return nil, err
	}
	before := parseProcSnapshot(blocks[1])
	after := parseProcSnapshot(blocks[2])

	jiffiesByUser := make(map[string]int64)
	for pid, a := range after {
		b, seenBefore := before[pid]
		if !seenBefore || a.session == ownSession || pid == sshdPID {
			continue
		}
		delta := a.jiffies - b.jiffies
		if delta <= 0 {
			continue
		}
		user := pidUser[pid]
		if user == "" {
			user = "?"
		}
		jiffiesByUser[user] += delta
	}

	denom := float64(clkTck) * procSampleSeconds
	var users []UserCPU
	for user, jiffies := range jiffiesByUser {
		pct := float64(jiffies) * 100 / denom
		if pct > userCPUThreshold {
			users = append(users, UserCPU{User: user, CPUPercent: pct})
		}
	}
	sort.Slice(users, func(i, j int) bool { return users[i].CPUPercent > users[j].CPUPercent })
	return users, nil
}

// parseProcHeader parses the first block of topUsersCmd's output: a
// "CLK <n> SID <n> SSHD <n>" line followed by "<user> /proc/<pid>" mapping lines.
func parseProcHeader(block string) (clkTck int64, ownSession, sshdPID string, pidUser map[string]string, err error) {
	lines := splitNonEmpty(block)
	if len(lines) == 0 {
		return 0, "", "", nil, fmt.Errorf("empty per-user CPU header")
	}
	hdr := strings.Fields(lines[0])
	if len(hdr) != 6 || hdr[0] != "CLK" || hdr[2] != "SID" || hdr[4] != "SSHD" {
		return 0, "", "", nil, fmt.Errorf("malformed per-user CPU header (raw: %q)", lines[0])
	}
	clkTck, err = strconv.ParseInt(hdr[1], 10, 64)
	if err != nil {
		return 0, "", "", nil, fmt.Errorf("parsing CLK_TCK: %w (raw: %q)", err, hdr[1])
	}
	if clkTck <= 0 {
		return 0, "", "", nil, fmt.Errorf("non-positive CLK_TCK: %d", clkTck)
	}
	ownSession, sshdPID = hdr[3], hdr[5]

	pidUser = make(map[string]string)
	for _, line := range lines[1:] {
		// "<user> /proc/<pid>" — usernames have no spaces, so two fields.
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid := fields[1][strings.LastIndexByte(fields[1], '/')+1:]
		pidUser[pid] = fields[0]
	}
	return clkTck, ownSession, sshdPID, pidUser, nil
}

// parseProcSnapshot parses a block of /proc/<pid>/stat lines into pid → procStat.
// Malformed lines are skipped (processes come and go between snapshots).
func parseProcSnapshot(block string) map[string]procStat {
	out := make(map[string]procStat)
	for _, line := range splitNonEmpty(block) {
		// Format: "<pid> (comm) <state> <ppid> <pgrp> ... <utime> <stime> ...".
		// comm can contain spaces and ')', so the pid is the text before the
		// first space and the remaining fields start after the *last* ')'.
		sp := strings.IndexByte(line, ' ')
		rp := strings.LastIndexByte(line, ')')
		if sp < 0 || rp < 0 || rp+1 >= len(line) {
			continue
		}
		pid := line[:sp]
		// Fields after comm (0-indexed): state(0) ppid(1) pgrp(2) session(3) ...
		// utime(11) stime(12).
		f := strings.Fields(line[rp+1:])
		if len(f) < 13 {
			continue
		}
		utime, err1 := strconv.ParseInt(f[11], 10, 64)
		stime, err2 := strconv.ParseInt(f[12], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out[pid] = procStat{session: f[3], jiffies: utime + stime}
	}
	return out
}

// splitNonEmpty splits a string by newlines and returns only non-empty lines.
func splitNonEmpty(s string) []string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// CheckServer checks a single server's metrics via SSH.
func CheckServer(client SSHClient, server Server) ServerStatus {
	command := BuildCommand(server)
	output, err := client.RunCommand(server, command)
	if err != nil {
		return ServerStatus{
			Name:   server.Name,
			Status: "offline",
			Error:  err.Error(),
		}
	}

	metrics, err := ParseMetrics(output, server.HasGPU)
	if err != nil {
		return ServerStatus{
			Name:   server.Name,
			Status: "online",
			Error:  fmt.Sprintf("metrics parse error: %s", err),
		}
	}

	return ServerStatus{
		Name:    server.Name,
		Status:  "online",
		Metrics: metrics,
	}
}

// CheckAllServers checks all servers in parallel with bounded concurrency.
func CheckAllServers(client SSHClient, servers []Server) ScoutResult {
	results := make([]ServerStatus, len(servers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)

	for i, server := range servers {
		wg.Add(1)
		go func(idx int, srv Server) {
			defer wg.Done()
			sem <- struct{}{}        // acquire semaphore
			defer func() { <-sem }() // release semaphore
			results[idx] = CheckServer(client, srv)
		}(i, server)
	}

	wg.Wait()
	return ScoutResult{Servers: results}
}
