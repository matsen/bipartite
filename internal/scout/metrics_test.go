package scout

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// procStatLine builds a minimal /proc/<pid>/stat line: the given session and
// utime jiffies (stime 0), with enough trailing fields for parseProcSnapshot.
func procStatLine(pid int, comm, session string, jiffies int) string {
	// After "(comm)": state ppid pgrp session tty tpgid flags minflt cminflt
	// majflt cmajflt utime stime — session at index 3, utime at index 11.
	return fmt.Sprintf("%d (%s) S 1 0 %s 0 -1 0 0 0 0 0 %d 0", pid, comm, session, jiffies)
}

// cpuStatLine builds a /proc/stat aggregate "cpu" line (user nice system idle
// iowait, then zeros for irq/softirq/steal).
func cpuStatLine(user, nice, system, idle, iowait int) string {
	return fmt.Sprintf("cpu %d %d %d %d %d 0 0 0", user, nice, system, idle, iowait)
}

// procSectionCPU assembles procSampleCmd's output, prepending the given /proc/stat
// "cpu" lines to each snapshot block.
func procSectionCPU(clk, sid, sshd, cpuBefore, cpuAfter string, nameLines, snap1, snap2 []string) string {
	header := append([]string{"CLK " + clk + " SID " + sid + " SSHD " + sshd}, nameLines...)
	return strings.Join(header, "\n") + "\n" + procSnapMarker + "\n" +
		strings.Join(append([]string{cpuBefore}, snap1...), "\n") + "\n" + procSnapMarker + "\n" +
		strings.Join(append([]string{cpuAfter}, snap2...), "\n")
}

// procSection assembles procSampleCmd's output with default /proc/stat lines that
// yield 0% overall CPU (idle-only delta), for tests that only care about per-user.
func procSection(clk, sid, sshd string, nameLines, snap1, snap2 []string) string {
	return procSectionCPU(clk, sid, sshd,
		cpuStatLine(0, 0, 0, 100, 0), cpuStatLine(0, 0, 0, 200, 0),
		nameLines, snap1, snap2)
}

func TestBuildCommand_NoGPU(t *testing.T) {
	cmd := BuildCommand(Server{Name: "test", HasGPU: false})
	if containsStr(cmd, "nvidia-smi") {
		t.Error("command for non-GPU server should not contain nvidia-smi")
	}
	if containsStr(cmd, "top ") {
		t.Error("command should no longer shell out to top (overall CPU comes from /proc/stat)")
	}
	if !containsStr(cmd, "free -m") {
		t.Error("command should contain free")
	}
	if !containsStr(cmd, "uptime") {
		t.Error("command should contain uptime")
	}
	if !containsStr(cmd, "cat /proc/stat /proc/[0-9]*/stat") {
		t.Error("command should sample /proc/stat and every /proc/<pid>/stat")
	}
	if !containsStr(cmd, "stat -c") {
		t.Error("sampler should resolve names via stat (full, LDAP-aware)")
	}
	if !containsStr(cmd, procSnapMarker) {
		t.Error("sampler should delimit its snapshots with procSnapMarker")
	}
}

func TestBuildCommand_WithGPU(t *testing.T) {
	cmd := BuildCommand(Server{Name: "test", HasGPU: true})
	if !containsStr(cmd, "nvidia-smi") {
		t.Error("command for GPU server should contain nvidia-smi")
	}
}

func TestParseMetrics_NoGPU(t *testing.T) {
	// alice: delta 21 jiffies / (100*0.5) * 100 = 42.0%; bob: delta 5 = 10.0%.
	// Overall: Δtotal=100, Δidle=50 → 50% busy.
	sample := procSectionCPU("100", "999", "0",
		cpuStatLine(0, 0, 0, 100, 0), cpuStatLine(30, 0, 20, 150, 0),
		[]string{"alice /proc/10", "bob /proc/20"},
		[]string{procStatLine(10, "bash", "10", 0), procStatLine(20, "python", "20", 0)},
		[]string{procStatLine(10, "bash", "10", 21), procStatLine(20, "python", "20", 5)},
	)
	output := sample + delimiter +
		"45.3" + delimiter +
		" 0.52, 0.48, 0.41"

	metrics, err := ParseMetrics(output, false)
	if err != nil {
		t.Fatal(err)
	}

	if metrics.CPUPercent != 50.0 {
		t.Errorf("CPU: expected 50.0, got %f", metrics.CPUPercent)
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
	if metrics.TopUsers[0].User != "alice" || metrics.TopUsers[0].CPUPercent != 42.0 {
		t.Errorf("expected alice 42.0, got %s %f", metrics.TopUsers[0].User, metrics.TopUsers[0].CPUPercent)
	}
}

func TestParseMetrics_WithGPU(t *testing.T) {
	// Overall: Δtotal=100, Δidle=10 → 90% busy.
	sample := procSectionCPU("100", "999", "0",
		cpuStatLine(0, 0, 0, 100, 0), cpuStatLine(80, 0, 10, 110, 0),
		[]string{"charlie /proc/30"},
		[]string{procStatLine(30, "matlab", "30", 0)},
		[]string{procStatLine(30, "matlab", "30", 44)},
	)
	output := sample + delimiter +
		"26.1" + delimiter +
		" 5.41, 5.43, 5.20" + delimiter +
		"100\n100" + delimiter +
		"17706, 20480\n17706, 20480"

	metrics, err := ParseMetrics(output, true)
	if err != nil {
		t.Fatal(err)
	}

	if metrics.CPUPercent != 90.0 {
		t.Errorf("CPU: expected 90.0, got %f", metrics.CPUPercent)
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

// validSample is a well-formed sample section yielding 0 users and 0% overall CPU.
func validSample() string {
	return procSection("100", "999", "0", nil, nil, nil)
}

func TestParseMetrics_BadCPU(t *testing.T) {
	// /proc/stat cpu line has non-numeric fields.
	sample := procSectionCPU("100", "999", "0",
		"cpu x y z 100 0", "cpu x y z 200 0", nil, nil, nil)
	output := sample + delimiter + "45.3" + delimiter + "0.52, 0.48, 0.41"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for bad CPU value")
	}
}

func TestParseMetrics_BadMemory(t *testing.T) {
	output := validSample() + delimiter + "bad" + delimiter + "0.52, 0.48, 0.41"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for bad memory value")
	}
}

func TestParseMetrics_BadLoadAvg(t *testing.T) {
	output := validSample() + delimiter + "45.3" + delimiter + "0.52, bad, 0.41"
	_, err := ParseMetrics(output, false)
	if err == nil {
		t.Fatal("expected error for bad load avg value")
	}
}

func TestParseMetrics_MissingGPUGraceful(t *testing.T) {
	// GPU server where nvidia-smi output is empty/missing
	output := validSample() + delimiter +
		"26.1" + delimiter +
		" 5.41, 5.43, 5.20" + delimiter +
		"" + delimiter + // empty GPU util
		"" // empty GPU mem

	metrics, err := ParseMetrics(output, true)
	if err != nil {
		t.Fatal(err)
	}
	// Should still return metrics, just without GPU data
	if metrics.CPUPercent != 0.0 {
		t.Errorf("CPU: expected 0.0, got %f", metrics.CPUPercent)
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

func TestParseProcUserCPU_Deltas(t *testing.T) {
	// denom = 100 * 0.5 = 50, so pct = delta_jiffies * 2.
	// alice: 30 jiffies -> 60.0%; bob: 5 -> 10.0%. Sorted descending.
	section := procSection("100", "999", "0",
		[]string{"alice /proc/10", "bob /proc/20"},
		[]string{procStatLine(10, "bash", "10", 100), procStatLine(20, "py", "20", 0)},
		[]string{procStatLine(10, "bash", "10", 130), procStatLine(20, "py", "20", 5)},
	)
	got, _, err := parseProcSample(section)
	if err != nil {
		t.Fatal(err)
	}
	want := []UserCPU{{User: "alice", CPUPercent: 60.0}, {User: "bob", CPUPercent: 10.0}}
	if len(got) != len(want) {
		t.Fatalf("expected %d users, got %d (%v)", len(want), len(got), got)
	}
	for i, u := range got {
		if u != want[i] {
			t.Errorf("user %d: expected %v, got %v", i, want[i], u)
		}
	}
}

func TestParseProcUserCPU_CommWithSpacesAndParens(t *testing.T) {
	// utime is split from the text after the *last* ')', so a comm containing
	// spaces and ')' must not corrupt the field offsets.
	section := procSection("100", "999", "0",
		[]string{"carol /proc/40"},
		[]string{procStatLine(40, "weird ) name", "40", 0)},
		[]string{procStatLine(40, "weird ) name", "40", 7)},
	)
	got, _, err := parseProcSample(section)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].User != "carol" || got[0].CPUPercent != 14.0 {
		t.Fatalf("expected carol 14.0, got %v", got)
	}
}

func TestParseProcUserCPU_ExcludesOwnSession(t *testing.T) {
	// pid 50 is in scout's own login session (SID 999) and must be dropped despite
	// a large delta — that's the cat/stat pipeline doing the sampling.
	section := procSection("100", "999", "0",
		[]string{"alice /proc/10", "scout /proc/50"},
		[]string{procStatLine(10, "bash", "10", 0), procStatLine(50, "cat", "999", 0)},
		[]string{procStatLine(10, "bash", "10", 5), procStatLine(50, "cat", "999", 9999)},
	)
	got, _, err := parseProcSample(section)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].User != "alice" {
		t.Fatalf("expected only alice, got %v", got)
	}
}

func TestParseProcUserCPU_ExcludesServingSshd(t *testing.T) {
	// pid 50 is the sshd serving the connection (SSHD 50). It lives in a different
	// session (777, not our SID 999), so only the explicit SSHD-pid exclusion
	// drops it — otherwise the CPU it spends streaming output back is mis-charged.
	section := procSection("100", "999", "50",
		[]string{"alice /proc/10", "matsen /proc/50"},
		[]string{procStatLine(10, "bash", "10", 0), procStatLine(50, "sshd", "777", 0)},
		[]string{procStatLine(10, "bash", "10", 5), procStatLine(50, "sshd", "777", 9999)},
	)
	got, _, err := parseProcSample(section)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].User != "alice" {
		t.Fatalf("expected only alice (sshd excluded), got %v", got)
	}
}

func TestParseProcUserCPU_ThresholdAndUnknownUser(t *testing.T) {
	// With CLK 1000, denom = 500. dave: 3 jiffies -> 0.6% (dropped, <= 1.0).
	// frank: 10 -> 2.0% (kept). pid 70 has no name entry -> labeled "?".
	section := procSection("1000", "999", "0",
		[]string{"dave /proc/60", "frank /proc/80"},
		[]string{procStatLine(60, "a", "60", 0), procStatLine(80, "b", "80", 0), procStatLine(70, "c", "70", 0)},
		[]string{procStatLine(60, "a", "60", 3), procStatLine(80, "b", "80", 10), procStatLine(70, "c", "70", 25)},
	)
	got, _, err := parseProcSample(section)
	if err != nil {
		t.Fatal(err)
	}
	// "?" (pid 70): 25 jiffies -> 5.0%; frank 2.0%; dave dropped.
	want := []UserCPU{{User: "?", CPUPercent: 5.0}, {User: "frank", CPUPercent: 2.0}}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i, u := range got {
		if u != want[i] {
			t.Errorf("user %d: expected %v, got %v", i, want[i], u)
		}
	}
}

func TestParseProcUserCPU_Malformed(t *testing.T) {
	for _, tc := range []struct {
		name, input string
	}{
		{"empty", ""},
		{"wrong block count", "CLK 100 SID 1 SSHD 1\n" + procSnapMarker + "\nsnap1"},
		{"bad header", "garbage\n" + procSnapMarker + "\n" + procSnapMarker},
		{"bad clk", "CLK x SID 1 SSHD 1\n" + procSnapMarker + "\n" + procSnapMarker},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := parseProcSample(tc.input); err == nil {
				t.Errorf("expected error for %q", tc.input)
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
	// Δtotal=200, Δidle=175 → 12.5% busy.
	sample := procSectionCPU("100", "999", "0",
		"cpu 0 0 0 100 0", "cpu 25 0 0 275 0", nil, nil, nil)
	client.outputs["test"] = sample + delimiter + "45.3" + delimiter + "0.52, 0.48, 0.41"

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
		client.outputs[name] = validSample() + delimiter + "2.0" + delimiter + "0.1, 0.2, 0.3"
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
		client.outputs[name] = validSample() + delimiter + "2.0" + delimiter + "0.1, 0.2, 0.3"
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
