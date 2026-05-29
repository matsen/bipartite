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

// procSampleSeconds is the gap between the two /proc snapshots. It must match the
// `sleep` in procSampleCmd (both derive from it).
const procSampleSeconds = 0.5

// userCPUThreshold is the minimum per-user CPU percent (100 == one full core) to
// report, dropping the long tail of near-idle accounts.
const userCPUThreshold = 1.0

// procSnapMarker separates the three blocks of procSampleCmd's output: the header
// + uid→name map, then the two /proc snapshots. It must not contain the section
// delimiter, and parseProcSample splits the section on it.
const procSnapMarker = "@@SCOUT_SNAP@@"

// procSampleCmd dumps the raw data needed to compute both per-user CPU and overall
// CPU, doing no arithmetic on the remote host — parseProcSample does that in Go.
// It emits, in one section:
//
//	CLK <clk_tck> SID <our_session> SSHD <serving_sshd_pid>
//	<user> /proc/<pid>        (one per process, full LDAP names via stat(1))
//	@@SCOUT_SNAP@@
//	<contents of /proc/stat and every /proc/<pid>/stat>   (snapshot 1)
//	@@SCOUT_SNAP@@
//	<contents of /proc/stat and every /proc/<pid>/stat>   (snapshot 2, sleep later)
//
// Two snapshots differenced over one window give both numbers:
//   - Per-user: sum each process's (utime+stime) jiffy delta by user — a current
//     reading (100% == one core, matching ps's %cpu unit) without ps's two failure
//     modes: ps %cpu is a lifetime average, so the short-lived sshd serving the SSH
//     session reports a large % from a tiny denominator (a spurious per-user floor),
//     and a process that ran hot hours ago but is now idle keeps over-reporting.
//   - Overall: from /proc/stat's aggregate "cpu" line, busy% = (Δtotal − Δidle) /
//     Δtotal, where idle includes iowait. This replaces `top`, which both needs its
//     own second sample (so we'd sleep twice) and, as we parsed it, reported only
//     the user fraction — undercounting system/IO-heavy machines.
//
// `stat -c %U` is used for names rather than top's USER column because top truncates
// to 8 chars + '+', merging distinct users (systemd-resolve/systemd-timesync both
// become "systemd+"). SID and SSHD identify scout's own footprint so parseProcSample
// can exclude it: SID (our login session — field 6 of the shell's own /proc/$$/stat)
// covers the shell and the cat/stat pipeline, and SSHD ($PPID) is the sshd serving
// the connection — it sits in a *different* session, so SID alone misses it, yet the
// CPU it spends streaming this (sizeable) output back would otherwise be mis-charged
// to the ssh user.
var procSampleCmd = fmt.Sprintf(
	`echo "CLK $(getconf CLK_TCK) SID $(cut -d' ' -f6 /proc/$$/stat) SSHD $PPID"; `+
		`stat -c '%%U %%n' /proc/[0-9]* 2>/dev/null; echo '%s'; `+
		`cat /proc/stat /proc/[0-9]*/stat 2>/dev/null; echo '%s'; `+
		`sleep %.1f; cat /proc/stat /proc/[0-9]*/stat 2>/dev/null`,
	procSnapMarker, procSnapMarker, procSampleSeconds,
)

// Section indices for ParseMetrics output splitting.
// These must match the command order in BuildCommand.
const (
	sectionSample  = 0 // per-user CPU and overall CPU (parseProcSample)
	sectionMemory  = 1
	sectionLoadAvg = 2
	sectionGPUUtil = 3
	sectionGPUMem  = 4
)

// BuildCommand constructs the combined command string for a server.
// All metric commands are joined with delimiters for single-session execution.
func BuildCommand(server Server) string {
	cmds := []string{
		// Per-user and overall CPU — raw /proc snapshots; parseProcSample does the math.
		procSampleCmd,
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

	expectedSections := 3
	if hasGPU {
		expectedSections = 5
	}
	if len(sections) < expectedSections {
		return nil, fmt.Errorf("expected %d metric sections, got %d", expectedSections, len(sections))
	}

	metrics := &ServerMetrics{}

	// Parse per-user and overall CPU from the /proc snapshots (one sample window).
	// The snapshot data identifies and excludes scout's own session and sshd.
	users, cpu, err := parseProcSample(sections[sectionSample])
	if err != nil {
		return nil, err
	}
	metrics.TopUsers = users
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

// parseProcSample turns procSampleCmd's output (header + uid→name map + two
// snapshots of /proc/stat and every /proc/<pid>/stat, separated by procSnapMarker)
// into per-user CPU and overall CPU, both measured over the single sample window.
//
// Per-user: for each process present in both snapshots and not part of scout's own
// session or serving sshd, difference the (utime+stime) jiffies; the per-user
// totals become a percentage where 100 == one fully-used core, matching ps's %cpu
// unit. Users at or below userCPUThreshold are dropped, sorted by CPU descending.
//
// Overall: from /proc/stat's aggregate "cpu" line, busy% = (Δtotal − Δidle) /
// Δtotal * 100, where idle includes iowait.
func parseProcSample(section string) ([]UserCPU, float64, error) {
	blocks := strings.Split(section, procSnapMarker)
	if len(blocks) != 3 {
		return nil, 0, fmt.Errorf("expected 3 blocks separated by %q, got %d", procSnapMarker, len(blocks))
	}

	clkTck, ownSession, sshdPID, pidUser, err := parseProcHeader(blocks[0])
	if err != nil {
		return nil, 0, err
	}
	before := parseProcSnapshot(blocks[1])
	after := parseProcSnapshot(blocks[2])

	cpuPercent, err := overallCPU(blocks[1], blocks[2])
	if err != nil {
		return nil, 0, err
	}

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
	return users, cpuPercent, nil
}

// overallCPU computes system-wide busy percent from the aggregate "cpu" line of
// /proc/stat in the before and after snapshot blocks.
func overallCPU(beforeBlock, afterBlock string) (float64, error) {
	totalBefore, idleBefore, okBefore := parseCPUStat(beforeBlock)
	totalAfter, idleAfter, okAfter := parseCPUStat(afterBlock)
	if !okBefore || !okAfter {
		return 0, fmt.Errorf("missing /proc/stat cpu line in snapshot")
	}
	deltaTotal := totalAfter - totalBefore
	if deltaTotal <= 0 {
		return 0, fmt.Errorf("non-positive /proc/stat total delta: %d", deltaTotal)
	}
	deltaIdle := idleAfter - idleBefore
	return float64(deltaTotal-deltaIdle) * 100 / float64(deltaTotal), nil
}

// parseCPUStat reads the aggregate "cpu" line from a snapshot block (the rest of
// /proc/stat and the per-pid lines are ignored) and returns total and idle jiffies.
// Fields after "cpu": user nice system idle iowait irq softirq steal ...; idle
// counts idle+iowait.
func parseCPUStat(block string) (total, idle int64, ok bool) {
	for _, line := range splitNonEmpty(block) {
		f := strings.Fields(line)
		if len(f) < 6 || f[0] != "cpu" {
			continue
		}
		// Sum user..steal (fields 1–8). guest/guest_nice (9–10) are already
		// included in user/nice, so adding them would double-count guest time.
		end := len(f)
		if end > 9 {
			end = 9
		}
		for _, field := range f[1:end] {
			v, err := strconv.ParseInt(field, 10, 64)
			if err != nil {
				return 0, 0, false
			}
			total += v
		}
		idle = mustInt(f[4]) + mustInt(f[5]) // idle + iowait
		return total, idle, true
	}
	return 0, 0, false
}

// mustInt parses a base-10 int that parseCPUStat has already validated.
func mustInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// parseProcHeader parses the first block of procSampleCmd's output: a
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
