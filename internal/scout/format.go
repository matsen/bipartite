package scout

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// FormatOptions controls the human-readable table.
type FormatOptions struct {
	Color bool   // add ANSI colors (callers set it only for a terminal)
	Me    string // the user whose GPUs and CPU usage are highlighted
}

// ANSI styles. In the GPU columns the text color says how busy each GPU is.
// Util %: white below halfUsed. Memory: white below halfUsed, yellow below
// nearlyFull, red above. A white background marks GPUs running a process of
// opts.Me in the util % column, and opts.Me in the Users column; text that would
// be white or uncolored is black there. "offline" and "metrics error" are red.
const (
	ansiOffline = "\x1b[31m"
	ansiReset   = "\x1b[0m"

	fgLow    = "37" // white
	fgHalf   = "33" // yellow
	fgFull   = "31" // red
	fgOnMine = "30" // black
	bgMine   = "47" // white
	boldCode = "1"
)

// Fractions at which GPU text stops being white (util % and memory), and at
// which memory text turns red.
const (
	halfUsed   = 0.5
	nearlyFull = 0.9
)

// styler applies FormatOptions' colors to individual table values.
type styler struct{ FormatOptions }

func (st styler) paint(code, s string) string {
	if !st.Color {
		return s
	}
	return code + s + ansiReset
}

// gpuCell styles one GPU sub-column value: bold text in color fg (none if empty),
// on a white background if the GPU runs a process of opts.Me.
func (st styler) gpuCell(g GPUInfo, s string, fg string) string {
	if (fg == "" || fg == fgLow) && st.isMine(g) {
		fg = fgOnMine
	}
	codes := []string{}
	if fg != "" {
		codes = append(codes, fg)
	}
	if st.isMine(g) {
		codes = append(codes, bgMine)
	}
	if len(codes) == 0 {
		return s
	}
	return st.paint("\x1b["+boldCode+";"+strings.Join(codes, ";")+"m", s)
}

// memColor is the GPU Mem text color for usedGB of totalGB.
func memColor(usedGB, totalGB int) string {
	switch {
	case float64(usedGB) >= nearlyFull*float64(totalGB):
		return fgFull
	case float64(usedGB) >= halfUsed*float64(totalGB):
		return fgHalf
	}
	return fgLow
}

func (st styler) isMine(g GPUInfo) bool {
	if st.Me == "" {
		return false
	}
	for _, u := range g.Users {
		if u == st.Me {
			return true
		}
	}
	return false
}

// FormatTable formats a ScoutResult as a human-readable table.
func FormatTable(result ScoutResult, opts FormatOptions) string {
	st := styler{opts}

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
		rows[i] = formatRow(s, maxGPUs, memSubWidth, st)
	}

	headers := []string{"Server", "GPUs", "GPU Mem (GB)", "Users (%CPU)", "CPU", "Mem"}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if visibleLen(cell) > widths[i] {
				widths[i] = visibleLen(cell)
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
	for r, row := range rows {
		online := servers[r].Metrics != nil
		for i, cell := range row {
			if i > 0 {
				sb.WriteString("  ")
			}
			// Left-align server, users, and the note on a row without metrics;
			// right-align everything else.
			if i == 0 || i == 3 || !online {
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
func formatRow(s ServerStatus, maxGPUs int, memSubWidth int, st styler) []string {
	if s.Status == "offline" {
		return []string{s.Name, st.paint(ansiOffline, "offline"), "", "", "", ""}
	}

	if s.Metrics == nil {
		return []string{s.Name, st.paint(ansiOffline, "metrics error"), "", "", "", ""}
	}

	m := s.Metrics
	cpu := fmt.Sprintf("%.1f%%", m.CPUPercent)
	mem := fmt.Sprintf("%.1f%%", m.MemoryPercent)

	gpuUsage := ""
	gpuMemory := ""
	if len(m.GPUs) > 0 {
		gpuUsage = formatGPUUsage(m.GPUs, maxGPUs, st)
		gpuMemory = formatGPUMemory(m.GPUs, maxGPUs, memSubWidth, st)
	}

	return []string{s.Name, gpuUsage, gpuMemory, formatUsers(m.TopUsers, st), cpu, mem}
}

const gpuUtilSubWidth = 4 // enough for "100%"

// formatGPUUsage formats per-GPU utilization as fixed-width sub-columns.
// Each slot is right-aligned to gpuUtilSubWidth chars, separated by spaces.
// Servers with fewer GPUs than maxGPUs get leading blank slots (right-justified).
func formatGPUUsage(gpus []GPUInfo, maxGPUs int, st styler) string {
	if maxGPUs == 0 {
		return ""
	}
	parts := make([]string, maxGPUs)
	offset := maxGPUs - len(gpus)
	for i := 0; i < maxGPUs; i++ {
		if i < offset {
			parts[i] = strings.Repeat(" ", gpuUtilSubWidth)
		} else {
			g := gpus[i-offset]
			fg := ""
			if float64(g.UtilizationPercent) < halfUsed*100 {
				fg = fgLow
			}
			parts[i] = st.gpuCell(g, padLeft(fmt.Sprintf("%d%%", g.UtilizationPercent), gpuUtilSubWidth), fg)
		}
	}
	return strings.Join(parts, " ")
}

// formatGPUMemory formats per-GPU memory as fixed-width sub-columns of used/total GB.
// Each slot is right-aligned to subWidth chars, separated by spaces.
// Servers with fewer GPUs than maxGPUs get leading blank slots (right-justified).
func formatGPUMemory(gpus []GPUInfo, maxGPUs int, subWidth int, st styler) string {
	if maxGPUs == 0 {
		return ""
	}
	parts := make([]string, maxGPUs)
	offset := maxGPUs - len(gpus)
	for i := 0; i < maxGPUs; i++ {
		if i < offset {
			parts[i] = strings.Repeat(" ", subWidth)
		} else {
			g := gpus[i-offset]
			used := int(math.Round(float64(g.MemoryUsedMB) / 1024))
			total := int(math.Round(float64(g.MemoryTotalMB) / 1024))
			parts[i] = st.paint("\x1b["+boldCode+";"+memColor(used, total)+"m", padLeft(fmt.Sprintf("%d/%d", used, total), subWidth))
		}
	}
	return strings.Join(parts, " ")
}

// formatUsers formats top CPU users as "name(N%) name2(M%)" showing up to 3.
// If more than 3 users, appends "+N" indicating how many are hidden.
func formatUsers(users []UserCPU, st styler) string {
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
		if u.User == st.Me {
			parts[i] = st.paint("\x1b["+boldCode+";"+fgOnMine+";"+bgMine+"m", parts[i])
		}
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

// visibleLen is the printed width of s: its length without ANSI color sequences.
func visibleLen(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		n++
	}
	return n
}

func padRight(s string, width int) string {
	if visibleLen(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visibleLen(s))
}

func padLeft(s string, width int) string {
	if visibleLen(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-visibleLen(s)) + s
}
