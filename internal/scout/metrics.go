package scout

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// delimiter separates command outputs in a single SSH session.
const delimiter = "___SCOUT_DELIM___"

// maxConcurrent is the bounded semaphore size for parallel server checks.
const maxConcurrent = 5

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
		// Top CPU users — runs first to avoid counting scout's own commands
		`ps -eo user:20,%cpu --no-headers | awk '{cpu[$1]+=$2} END {for (u in cpu) if (cpu[u]>1.0) printf "%s %.1f\n",u,cpu[u]}' | sort -k2 -rn`,
		// CPU usage
		`top -bn1 | grep -i "cpu(s)" | awk '{print $2}' | cut -d'%' -f1`,
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
	// (runs first on the server to avoid counting scout's own commands)
	if users, err := parseTopUsers(sections[sectionTopUsers]); err == nil {
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

// parseTopUsers parses "username 42.1" lines into a slice of UserCPU.
// Returns nil slice for empty input (no users above threshold).
func parseTopUsers(output string) ([]UserCPU, error) {
	lines := splitNonEmpty(output)
	if len(lines) == 0 {
		return nil, nil
	}

	var users []UserCPU
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return nil, fmt.Errorf("parsing user CPU line: expected 2 fields, got %d (raw: %q)", len(fields), line)
		}
		pct, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parsing user CPU percent: %w (raw: %q)", err, fields[1])
		}
		users = append(users, UserCPU{User: fields[0], CPUPercent: pct})
	}
	return users, nil
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
