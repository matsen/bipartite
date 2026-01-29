package scout

import (
	"fmt"
	"sync"
	"testing"
)

func TestBuildCommand_NoGPU(t *testing.T) {
	cmd := BuildCommand(Server{Name: "test", HasGPU: false})
	if containsStr(cmd, "nvidia-smi") {
		t.Error("command for non-GPU server should not contain nvidia-smi")
	}
	if !containsStr(cmd, "top -bn1") {
		t.Error("command should contain top")
	}
	if !containsStr(cmd, "free -m") {
		t.Error("command should contain free")
	}
	if !containsStr(cmd, "uptime") {
		t.Error("command should contain uptime")
	}
	if !containsStr(cmd, "ps -eo") {
		t.Error("command should contain ps")
	}
}

func TestBuildCommand_WithGPU(t *testing.T) {
	cmd := BuildCommand(Server{Name: "test", HasGPU: true})
	if !containsStr(cmd, "nvidia-smi") {
		t.Error("command for GPU server should contain nvidia-smi")
	}
}

func TestParseMetrics_NoGPU(t *testing.T) {
	output := "alice 42.1\nbob 10.3" + delimiter +
		"12.5" + delimiter +
		"45.3" + delimiter +
		" 0.52, 0.48, 0.41"

	metrics, err := ParseMetrics(output, false)
	if err != nil {
		t.Fatal(err)
	}

	if metrics.CPUPercent != 12.5 {
		t.Errorf("CPU: expected 12.5, got %f", metrics.CPUPercent)
	}
	if metrics.MemoryPercent != 45.3 {
		t.Errorf("Memory: expected 45.3, got %f", metrics.MemoryPercent)
	}
	if metrics.LoadAvg1 != 0.52 {
		t.Errorf("Load1: expected 0.52, got %f", metrics.LoadAvg1)
	}
	if metrics.LoadAvg5 != 0.48 {
		t.Errorf("Load5: expected 0.48, got %f", metrics.LoadAvg5)
	}
	if metrics.LoadAvg15 != 0.41 {
		t.Errorf("Load15: expected 0.41, got %f", metrics.LoadAvg15)
	}
	if metrics.GPUs != nil {
		t.Error("expected no GPU data")
	}
	if len(metrics.TopUsers) != 2 {
		t.Fatalf("expected 2 top users, got %d", len(metrics.TopUsers))
	}
	if metrics.TopUsers[0].User != "alice" || metrics.TopUsers[0].CPUPercent != 42.1 {
		t.Errorf("expected alice 42.1, got %s %f", metrics.TopUsers[0].User, metrics.TopUsers[0].CPUPercent)
	}
}

func TestParseMetrics_WithGPU(t *testing.T) {
	output := "charlie 88.5" + delimiter +
		"9.8" + delimiter +
		"26.1" + delimiter +
		" 5.41, 5.43, 5.20" + delimiter +
		"100\n100" + delimiter +
		"17706, 20480\n17706, 20480"

	metrics, err := ParseMetrics(output, true)
	if err != nil {
		t.Fatal(err)
	}

	if metrics.CPUPercent != 9.8 {
		t.Errorf("CPU: expected 9.8, got %f", metrics.CPUPercent)
	}
	if len(metrics.GPUs) != 2 {
		t.Fatalf("expected 2 GPUs, got %d", len(metrics.GPUs))
	}
	if metrics.GPUs[0].UtilizationPercent != 100 {
		t.Errorf("GPU0 util: expected 100, got %d", metrics.GPUs[0].UtilizationPercent)
	}
	if metrics.GPUs[0].MemoryUsedMB != 17706 {
		t.Errorf("GPU0 mem used: expected 17706, got %d", metrics.GPUs[0].MemoryUsedMB)
	}
	if metrics.GPUs[0].MemoryTotalMB != 20480 {
		t.Errorf("GPU0 mem total: expected 20480, got %d", metrics.GPUs[0].MemoryTotalMB)
	}
	if len(metrics.TopUsers) != 1 {
		t.Fatalf("expected 1 top user, got %d", len(metrics.TopUsers))
	}
	if metrics.TopUsers[0].User != "charlie" {
		t.Errorf("expected user charlie, got %s", metrics.TopUsers[0].User)
	}
}

func TestParseMetrics_InsufficientSections(t *testing.T) {
	output := "12.5" + delimiter + "45.3"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for insufficient sections")
	}
}

func TestParseMetrics_BadCPU(t *testing.T) {
	output := "" + delimiter + "not_a_number" + delimiter + "45.3" + delimiter + "0.52, 0.48, 0.41"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for bad CPU value")
	}
}

func TestParseMetrics_BadMemory(t *testing.T) {
	output := "" + delimiter + "12.5" + delimiter + "bad" + delimiter + "0.52, 0.48, 0.41"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for bad memory value")
	}
}

func TestParseMetrics_BadLoadAvg(t *testing.T) {
	output := "" + delimiter + "12.5" + delimiter + "45.3" + delimiter + "0.52, bad, 0.41"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for bad load avg value")
	}
}

func TestParseMetrics_MissingGPUGraceful(t *testing.T) {
	// GPU server where nvidia-smi output is empty/missing
	output := "" + delimiter + // empty user section
		"9.8" + delimiter +
		"26.1" + delimiter +
		" 5.41, 5.43, 5.20" + delimiter +
		"" + delimiter + // empty GPU util
		"" // empty GPU mem

	metrics, err := ParseMetrics(output, true)
	if err != nil {
		t.Fatal(err)
	}
	// Should still return metrics, just without GPU data
	if metrics.CPUPercent != 9.8 {
		t.Errorf("CPU: expected 9.8, got %f", metrics.CPUPercent)
	}
	if metrics.GPUs != nil {
		t.Error("expected nil GPUs when nvidia-smi output is empty")
	}
}

func TestParseGPUUtilization(t *testing.T) {
	vals, err := parseGPUUtilization("75\n100\n0\n50")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 4 {
		t.Fatalf("expected 4 values, got %d", len(vals))
	}
	expected := []int{75, 100, 0, 50}
	for i, v := range vals {
		if v != expected[i] {
			t.Errorf("GPU %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestParseGPUMemory(t *testing.T) {
	vals, err := parseGPUMemory("17706, 20480\n8192, 16384")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(vals))
	}
	if vals[0][0] != 17706 || vals[0][1] != 20480 {
		t.Errorf("GPU0 mem: expected [17706, 20480], got %v", vals[0])
	}
	if vals[1][0] != 8192 || vals[1][1] != 16384 {
		t.Errorf("GPU1 mem: expected [8192, 16384], got %v", vals[1])
	}
}

func TestParseTopUsers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []UserCPU
		wantErr  bool
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:  "single user",
			input: "alice 42.1",
			expected: []UserCPU{
				{User: "alice", CPUPercent: 42.1},
			},
		},
		{
			name:  "multiple users",
			input: "alice 42.1\nbob 10.3\ncharlie 5.0",
			expected: []UserCPU{
				{User: "alice", CPUPercent: 42.1},
				{User: "bob", CPUPercent: 10.3},
				{User: "charlie", CPUPercent: 5.0},
			},
		},
		{
			name:    "bad percent",
			input:   "alice notanumber",
			wantErr: true,
		},
		{
			name:    "too few fields",
			input:   "alice",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTopUsers(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d users, got %d", len(tt.expected), len(got))
			}
			for i, u := range got {
				if u.User != tt.expected[i].User || u.CPUPercent != tt.expected[i].CPUPercent {
					t.Errorf("user %d: expected %v, got %v", i, tt.expected[i], u)
				}
			}
		})
	}
}

// mockSSHClient is a test double for SSHClient.
type mockSSHClient struct {
	mu      sync.Mutex
	outputs map[string]string
	errors  map[string]error
	callLog []string
}

func newMockSSHClient() *mockSSHClient {
	return &mockSSHClient{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (m *mockSSHClient) RunCommand(server Server, command string) (string, error) {
	m.mu.Lock()
	m.callLog = append(m.callLog, server.Name)
	m.mu.Unlock()

	if err, ok := m.errors[server.Name]; ok {
		return "", err
	}
	return m.outputs[server.Name], nil
}

func (m *mockSSHClient) Close() error { return nil }

func TestCheckServer_Online(t *testing.T) {
	client := newMockSSHClient()
	client.outputs["test"] = "" + delimiter + "12.5" + delimiter + "45.3" + delimiter + "0.52, 0.48, 0.41"

	status := CheckServer(client, Server{Name: "test"})
	if status.Status != "online" {
		t.Errorf("expected online, got %s", status.Status)
	}
	if status.Metrics == nil {
		t.Fatal("expected metrics")
	}
	if status.Metrics.CPUPercent != 12.5 {
		t.Errorf("CPU: expected 12.5, got %f", status.Metrics.CPUPercent)
	}
}

func TestCheckServer_Offline(t *testing.T) {
	client := newMockSSHClient()
	client.errors["test"] = fmt.Errorf("connection timed out")

	status := CheckServer(client, Server{Name: "test"})
	if status.Status != "offline" {
		t.Errorf("expected offline, got %s", status.Status)
	}
	if status.Error == "" {
		t.Error("expected error message")
	}
	if status.Metrics != nil {
		t.Error("expected nil metrics for offline server")
	}
}

func TestCheckAllServers_Parallel(t *testing.T) {
	client := newMockSSHClient()
	servers := make([]Server, 10)
	for i := range servers {
		name := fmt.Sprintf("server%02d", i)
		servers[i] = Server{Name: name}
		client.outputs[name] = "" + delimiter + "1.0" + delimiter + "2.0" + delimiter + "0.1, 0.2, 0.3"
	}

	result := CheckAllServers(client, servers)
	if len(result.Servers) != 10 {
		t.Fatalf("expected 10 results, got %d", len(result.Servers))
	}

	// Verify all servers were checked
	for _, s := range result.Servers {
		if s.Status != "online" {
			t.Errorf("server %s: expected online, got %s", s.Name, s.Status)
		}
	}

	// Verify order is preserved (index-based assignment)
	for i, s := range result.Servers {
		expected := fmt.Sprintf("server%02d", i)
		if s.Name != expected {
			t.Errorf("result %d: expected %s, got %s", i, expected, s.Name)
		}
	}
}

func TestCheckAllServers_BoundedConcurrency(t *testing.T) {
	// Verify that the semaphore bounds concurrency to maxConcurrent.
	// We can't easily test the exact concurrency level, but we can verify
	// that all servers are checked even with many servers.
	client := newMockSSHClient()
	servers := make([]Server, 20)
	for i := range servers {
		name := fmt.Sprintf("server%02d", i)
		servers[i] = Server{Name: name}
		client.outputs[name] = "" + delimiter + "1.0" + delimiter + "2.0" + delimiter + "0.1, 0.2, 0.3"
	}

	result := CheckAllServers(client, servers)
	if len(result.Servers) != 20 {
		t.Fatalf("expected 20 results, got %d", len(result.Servers))
	}

	client.mu.Lock()
	if len(client.callLog) != 20 {
		t.Errorf("expected 20 calls, got %d", len(client.callLog))
	}
	client.mu.Unlock()
}
