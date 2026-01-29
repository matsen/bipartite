package scout

import (
	"strings"
	"testing"
)

func TestFormatTable_OnlineWithGPU(t *testing.T) {
	result := ScoutResult{
		Servers: []ServerStatus{
			{
				Name:   "mantis",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    9.8,
					MemoryPercent: 26.1,
					LoadAvg1:      5.41,
					LoadAvg5:      5.43,
					LoadAvg15:     5.20,
					GPUs: []GPUInfo{
						{UtilizationPercent: 100, MemoryUsedMB: 17706, MemoryTotalMB: 20480},
						{UtilizationPercent: 100, MemoryUsedMB: 17706, MemoryTotalMB: 20480},
					},
					TopUsers: []UserCPU{
						{User: "alice", CPUPercent: 42.1},
						{User: "bob", CPUPercent: 10.3},
					},
				},
			},
		},
	}

	output := FormatTable(result)

	if !strings.Contains(output, "mantis") {
		t.Error("expected server name in output")
	}
	if !strings.Contains(output, "online") {
		t.Error("expected online status in output")
	}
	if !strings.Contains(output, "9.8%") {
		t.Error("expected CPU percentage in output")
	}
	if !strings.Contains(output, "26.1%") {
		t.Error("expected memory percentage in output")
	}
	if !strings.Contains(output, "100% 100%") {
		t.Error("expected per-GPU utilization in output")
	}
	if !strings.Contains(output, "17/20 17/20") {
		t.Error("expected per-GPU memory in output")
	}
	if !strings.Contains(output, "alice(42%)") {
		t.Error("expected alice user in output")
	}
	if !strings.Contains(output, "bob(10%)") {
		t.Error("expected bob user in output")
	}
	if !strings.Contains(output, "Users") {
		t.Error("expected Users header in output")
	}
}

func TestFormatTable_OnlineWithoutGPU(t *testing.T) {
	result := ScoutResult{
		Servers: []ServerStatus{
			{
				Name:   "cricket",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    0.9,
					MemoryPercent: 3.7,
					LoadAvg1:      1.23,
					LoadAvg5:      1.19,
					LoadAvg15:     1.18,
				},
			},
		},
	}

	output := FormatTable(result)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Data line should have no GPU data (blank GPU columns)
	dataLine := lines[1]
	if strings.Contains(dataLine, "%") && strings.Contains(dataLine, "/") {
		t.Errorf("expected no GPU data for non-GPU server, got: %s", dataLine)
	}
}

func TestFormatTable_Offline(t *testing.T) {
	result := ScoutResult{
		Servers: []ServerStatus{
			{
				Name:   "beetle03",
				Status: "offline",
				Error:  "connection timed out",
			},
		},
	}

	output := FormatTable(result)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 3 {
		t.Fatal("expected at least 3 lines (header + underline + data)")
	}

	dataLine := lines[2]
	if !strings.Contains(dataLine, "offline") {
		t.Error("expected offline status")
	}

	// Offline server should only have name and status, no metric data
	fields := strings.Fields(dataLine)
	if len(fields) != 2 {
		t.Errorf("expected only name and status for offline server, got %d fields: %v", len(fields), fields)
	}
}

func TestFormatTable_ColumnAlignment(t *testing.T) {
	result := ScoutResult{
		Servers: []ServerStatus{
			{
				Name:   "short",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    1.0,
					MemoryPercent: 2.0,
					LoadAvg1:      0.1,
					LoadAvg5:      0.2,
					LoadAvg15:     0.3,
				},
			},
			{
				Name:   "verylongservername",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    99.9,
					MemoryPercent: 88.8,
					LoadAvg1:      10.0,
					LoadAvg5:      20.0,
					LoadAvg15:     30.0,
				},
			},
		},
	}

	output := FormatTable(result)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (header + underline + 2 data), got %d", len(lines))
	}

	// Header line and data lines should have consistent column positions
	if !strings.Contains(lines[0], "Server") {
		t.Error("expected header row")
	}
	if !strings.Contains(lines[1], "------") {
		t.Error("expected underline row")
	}
	if !strings.Contains(lines[2], "short") || !strings.Contains(lines[3], "verylongservername") {
		// Sort order: short (1.5% avg) before verylongservername (94.35% avg)
		t.Error("expected data rows for short and verylongservername")
	}
}

func TestFormatTable_GPUColumnAlignment(t *testing.T) {
	result := ScoutResult{
		Servers: []ServerStatus{
			{
				Name:   "ermine",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    23.2,
					MemoryPercent: 39.2,
					GPUs: []GPUInfo{
						{UtilizationPercent: 100, MemoryUsedMB: 67584, MemoryTotalMB: 81920},
						{UtilizationPercent: 100, MemoryUsedMB: 39936, MemoryTotalMB: 81920},
					},
				},
			},
			{
				Name:   "orca01",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    0.0,
					MemoryPercent: 0.9,
					GPUs: []GPUInfo{
						{UtilizationPercent: 0, MemoryUsedMB: 0, MemoryTotalMB: 45056},
						{UtilizationPercent: 0, MemoryUsedMB: 0, MemoryTotalMB: 45056},
						{UtilizationPercent: 0, MemoryUsedMB: 0, MemoryTotalMB: 45056},
						{UtilizationPercent: 0, MemoryUsedMB: 0, MemoryTotalMB: 45056},
					},
				},
			},
			{
				Name:   "quokka",
				Status: "online",
				Metrics: &ServerMetrics{
					CPUPercent:    5.0,
					MemoryPercent: 12.0,
				},
			},
		},
	}

	output := FormatTable(result)
	// Use TrimRight to only strip the final newline, preserving trailing spaces on lines
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines (header + underline + 3 data), got %d:\n%s", len(lines), output)
	}

	// Find each server's line by name (order depends on sort)
	var ermineLine, orcaLine, quokkaLine string
	for _, line := range lines[2:] {
		switch {
		case strings.Contains(line, "ermine"):
			ermineLine = line
		case strings.Contains(line, "orca01"):
			orcaLine = line
		case strings.Contains(line, "quokka"):
			quokkaLine = line
		}
	}

	// All data lines should be the same length (columns are padded to align)
	if len(ermineLine) != len(orcaLine) {
		t.Errorf("ermine and orca01 lines differ in length (%d vs %d):\n  %q\n  %q",
			len(ermineLine), len(orcaLine), ermineLine, orcaLine)
	}
	if len(ermineLine) != len(quokkaLine) {
		t.Errorf("ermine and quokka lines differ in length (%d vs %d):\n  %q\n  %q",
			len(ermineLine), len(quokkaLine), ermineLine, quokkaLine)
	}

	// Verify the 4-GPU server has right-aligned GPU util sub-columns
	if !strings.Contains(orcaLine, "  0%") {
		t.Errorf("expected right-aligned GPU util sub-columns, got: %s", orcaLine)
	}

	// Verify sort order: orca01 (0.15% avg) before quokka (8.5%) before ermine (65.6%)
	orcaIdx, quokkaIdx, ermineIdx := -1, -1, -1
	for i, line := range lines[2:] {
		switch {
		case strings.Contains(line, "orca01"):
			orcaIdx = i
		case strings.Contains(line, "quokka"):
			quokkaIdx = i
		case strings.Contains(line, "ermine"):
			ermineIdx = i
		}
	}
	if orcaIdx > quokkaIdx || quokkaIdx > ermineIdx {
		t.Errorf("expected sort order orca01, quokka, ermine; got indices %d, %d, %d",
			orcaIdx, quokkaIdx, ermineIdx)
	}
}

func TestSortByAvailability(t *testing.T) {
	servers := []ServerStatus{
		{Name: "offline1", Status: "offline"},
		{Name: "busy", Status: "online", Metrics: &ServerMetrics{CPUPercent: 90, MemoryPercent: 80}},
		{Name: "idle", Status: "online", Metrics: &ServerMetrics{CPUPercent: 1, MemoryPercent: 5}},
		{Name: "offline2", Status: "offline"},
		{Name: "gpu-free", Status: "online", Metrics: &ServerMetrics{
			CPUPercent: 2, MemoryPercent: 10,
			GPUs: []GPUInfo{{UtilizationPercent: 0}},
		}},
	}

	sortByAvailability(servers)

	names := make([]string, len(servers))
	for i, s := range servers {
		names[i] = s.Name
	}

	// gpu-free: (2+10+0)/3 = 4.0
	// idle: (1+5)/2 = 3.0
	// busy: (90+80)/2 = 85.0
	// offline servers last, stable order preserved
	expected := []string{"idle", "gpu-free", "busy", "offline1", "offline2"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("position %d: expected %s, got %s (order: %v)", i, want, names[i], names)
			break
		}
	}
}

func TestFormatTable_Empty(t *testing.T) {
	result := ScoutResult{}
	output := FormatTable(result)
	if !strings.Contains(output, "No servers") {
		t.Error("expected empty message")
	}
}

func TestFormatGPUUsage(t *testing.T) {
	tests := []struct {
		name     string
		gpus     []GPUInfo
		maxGPUs  int
		expected string
	}{
		{
			name:     "no GPUs",
			gpus:     nil,
			maxGPUs:  0,
			expected: "",
		},
		{
			name:     "single GPU",
			gpus:     []GPUInfo{{UtilizationPercent: 75}},
			maxGPUs:  1,
			expected: " 75%",
		},
		{
			name: "multiple GPUs",
			gpus: []GPUInfo{
				{UtilizationPercent: 100},
				{UtilizationPercent: 50},
			},
			maxGPUs:  2,
			expected: "100%  50%",
		},
		{
			name:     "fewer GPUs than max pads with leading blanks",
			gpus:     []GPUInfo{{UtilizationPercent: 0}},
			maxGPUs:  3,
			expected: "            0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatGPUUsage(tt.gpus, tt.maxGPUs)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFormatGPUMemory(t *testing.T) {
	tests := []struct {
		name     string
		gpus     []GPUInfo
		maxGPUs  int
		subWidth int
		expected string
	}{
		{
			name:     "no GPUs",
			gpus:     nil,
			maxGPUs:  0,
			subWidth: 0,
			expected: "",
		},
		{
			name:     "single GPU",
			gpus:     []GPUInfo{{MemoryUsedMB: 8192, MemoryTotalMB: 16384}},
			maxGPUs:  1,
			subWidth: 4,
			expected: "8/16",
		},
		{
			name: "multiple GPUs",
			gpus: []GPUInfo{
				{MemoryUsedMB: 17706, MemoryTotalMB: 20480},
				{MemoryUsedMB: 17706, MemoryTotalMB: 20480},
			},
			maxGPUs:  2,
			subWidth: 5,
			expected: "17/20 17/20",
		},
		{
			name:     "fewer GPUs than max pads with leading blanks",
			gpus:     []GPUInfo{{MemoryUsedMB: 0, MemoryTotalMB: 45056}},
			maxGPUs:  3,
			subWidth: 5,
			expected: "             0/44",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatGPUMemory(tt.gpus, tt.maxGPUs, tt.subWidth)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFormatUsers(t *testing.T) {
	tests := []struct {
		name     string
		users    []UserCPU
		expected string
	}{
		{
			name:     "nil users",
			users:    nil,
			expected: "",
		},
		{
			name:     "empty users",
			users:    []UserCPU{},
			expected: "",
		},
		{
			name:     "single user",
			users:    []UserCPU{{User: "alice", CPUPercent: 42.1}},
			expected: "alice(42%)",
		},
		{
			name: "three users",
			users: []UserCPU{
				{User: "alice", CPUPercent: 42.1},
				{User: "bob", CPUPercent: 10.3},
				{User: "charlie", CPUPercent: 5.0},
			},
			expected: "alice(42%) bob(10%) charlie(5%)",
		},
		{
			name: "more than three users truncates with +N",
			users: []UserCPU{
				{User: "alice", CPUPercent: 42.1},
				{User: "bob", CPUPercent: 10.3},
				{User: "charlie", CPUPercent: 5.0},
				{User: "dave", CPUPercent: 3.0},
				{User: "eve", CPUPercent: 2.0},
			},
			expected: "alice(42%) bob(10%) charlie(5%) +2",
		},
		{
			name:     "rounds to nearest integer",
			users:    []UserCPU{{User: "alice", CPUPercent: 99.5}},
			expected: "alice(100%)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUsers(tt.users)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
