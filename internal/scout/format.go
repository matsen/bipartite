package scout

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// FormatTable formats a ScoutResult as a human-readable table.
func FormatTable(result ScoutResult) string {
	if len(result.Servers) == 0 {
		return "No servers configured.\n"
	}

	// Sort by availability: most available servers first, offline last
	servers := make([]ServerStatus, len(result.Servers))
	copy(servers, result.Servers)
	sortByAvailability(servers)

	// Find max GPU count and memory sub-column width for alignment
	maxGPUs, memSubWidth := gpuColumnParams(servers)

	// Calculate column widths
	rows := make([][]string, len(servers))
	for i, s := range servers {
		rows[i] = formatRow(s, maxGPUs, memSubWidth)
	}

	headers := []string{"Server", "Status", "CPU", "Mem", "GPUs", "GPU Mem (GB)", "Users"}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Build output
	var sb strings.Builder

	// Header
	for i, h := range headers {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(padRight(h, widths[i]))
	}
	sb.WriteString("\n")

	// Underline
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("-", w))
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteString("  ")
			}
			// Left-align server, status, and users; right-align everything else
			if i <= 1 || i == 6 {
				sb.WriteString(padRight(cell, widths[i]))
			} else {
				sb.WriteString(padLeft(cell, widths[i]))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// gpuColumnParams scans servers to determine GPU sub-column sizing.
// Returns (maxGPUs, memSubWidth) where maxGPUs is the highest GPU count
// across all servers and memSubWidth is the widest individual memory value.
func gpuColumnParams(servers []ServerStatus) (int, int) {
	maxGPUs := 0
	memSubWidth := 0
	for _, s := range servers {
		if s.Metrics == nil {
			continue
		}
		if n := len(s.Metrics.GPUs); n > maxGPUs {
			maxGPUs = n
		}
		for _, g := range s.Metrics.GPUs {
			used := int(math.Round(float64(g.MemoryUsedMB) / 1024))
			total := int(math.Round(float64(g.MemoryTotalMB) / 1024))
			w := len(fmt.Sprintf("%d/%d", used, total))
			if w > memSubWidth {
				memSubWidth = w
			}
		}
	}
	return maxGPUs, memSubWidth
}

// formatRow formats a single server status as table cells.
func formatRow(s ServerStatus, maxGPUs int, memSubWidth int) []string {
	if s.Status == "offline" {
		return []string{s.Name, "offline", "", "", "", "", ""}
	}

	if s.Metrics == nil {
		return []string{s.Name, s.Status, "", "", "", "", ""}
	}

	m := s.Metrics
	cpu := fmt.Sprintf("%.1f%%", m.CPUPercent)
	mem := fmt.Sprintf("%.1f%%", m.MemoryPercent)

	gpuUsage := ""
	gpuMemory := ""
	if len(m.GPUs) > 0 {
		gpuUsage = formatGPUUsage(m.GPUs, maxGPUs)
		gpuMemory = formatGPUMemory(m.GPUs, maxGPUs, memSubWidth)
	}

	return []string{s.Name, "online", cpu, mem, gpuUsage, gpuMemory, formatUsers(m.TopUsers)}
}

const gpuUtilSubWidth = 4 // enough for "100%"

// formatGPUUsage formats per-GPU utilization as fixed-width sub-columns.
// Each slot is right-aligned to gpuUtilSubWidth chars, separated by spaces.
// Servers with fewer GPUs than maxGPUs get leading blank slots (right-justified).
func formatGPUUsage(gpus []GPUInfo, maxGPUs int) string {
	if maxGPUs == 0 {
		return ""
	}
	parts := make([]string, maxGPUs)
	offset := maxGPUs - len(gpus)
	for i := 0; i < maxGPUs; i++ {
		if i < offset {
			parts[i] = strings.Repeat(" ", gpuUtilSubWidth)
		} else {
			parts[i] = padLeft(fmt.Sprintf("%d%%", gpus[i-offset].UtilizationPercent), gpuUtilSubWidth)
		}
	}
	return strings.Join(parts, " ")
}

// formatGPUMemory formats per-GPU memory as fixed-width sub-columns of used/total GB.
// Each slot is right-aligned to subWidth chars, separated by spaces.
// Servers with fewer GPUs than maxGPUs get leading blank slots (right-justified).
func formatGPUMemory(gpus []GPUInfo, maxGPUs int, subWidth int) string {
	if maxGPUs == 0 {
		return ""
	}
	parts := make([]string, maxGPUs)
	offset := maxGPUs - len(gpus)
	for i := 0; i < maxGPUs; i++ {
		if i < offset {
			parts[i] = strings.Repeat(" ", subWidth)
		} else {
			used := int(math.Round(float64(gpus[i-offset].MemoryUsedMB) / 1024))
			total := int(math.Round(float64(gpus[i-offset].MemoryTotalMB) / 1024))
			parts[i] = padLeft(fmt.Sprintf("%d/%d", used, total), subWidth)
		}
	}
	return strings.Join(parts, " ")
}

// formatUsers formats top CPU users as "name(N%) name2(M%)" showing up to 3.
// If more than 3 users, appends "+N" indicating how many are hidden.
func formatUsers(users []UserCPU) string {
	if len(users) == 0 {
		return ""
	}
	show := users
	extra := 0
	if len(users) > 3 {
		show = users[:3]
		extra = len(users) - 3
	}
	parts := make([]string, len(show))
	for i, u := range show {
		parts[i] = fmt.Sprintf("%s(%d%%)", u.User, int(u.CPUPercent+0.5))
	}
	result := strings.Join(parts, " ")
	if extra > 0 {
		result += fmt.Sprintf(" +%d", extra)
	}
	return result
}

// sortByAvailability sorts servers with most available resources first.
// Online servers are sorted by average utilization ascending; offline servers go last.
func sortByAvailability(servers []ServerStatus) {
	sort.SliceStable(servers, func(i, j int) bool {
		si, sj := servers[i], servers[j]
		oi := si.Status == "online"
		oj := sj.Status == "online"
		if oi != oj {
			return oi
		}
		if !oi {
			return false
		}
		return avgUtilization(si) < avgUtilization(sj)
	})
}

// avgUtilization returns average utilization across CPU, memory, and GPUs.
// Returns 100 for servers without metrics (e.g., parse errors) to sort them
// toward the bottom, as they should be treated as "fully utilized" for
// resource allocation purposes.
func avgUtilization(s ServerStatus) float64 {
	if s.Metrics == nil {
		return 100
	}
	m := s.Metrics
	sum := m.CPUPercent + m.MemoryPercent
	count := 2.0
	for _, g := range m.GPUs {
		sum += float64(g.UtilizationPercent)
		count++
	}
	return sum / count
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
