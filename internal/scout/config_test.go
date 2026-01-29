package scout

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "servers.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_Valid(t *testing.T) {
	path := writeTestConfig(t, `
servers:
  - name: mantis
    has_gpu: true
  - name: cricket
  - pattern: "beetle{01..05}"
    has_gpu: true

ssh:
  proxy_jump: jumphost.example.org
  connect_timeout: 15
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Servers) != 3 {
		t.Fatalf("expected 3 server entries, got %d", len(cfg.Servers))
	}
	if cfg.SSH.ProxyJump != "jumphost.example.org" {
		t.Errorf("expected proxy_jump jumphost.example.org, got %s", cfg.SSH.ProxyJump)
	}
	if cfg.SSH.ConnectTimeout != 15 {
		t.Errorf("expected connect_timeout 15, got %d", cfg.SSH.ConnectTimeout)
	}
	if !cfg.Servers[0].HasGPU {
		t.Error("expected mantis to have GPU")
	}
	if cfg.Servers[1].HasGPU {
		t.Error("expected cricket to not have GPU")
	}
}

func TestLoadConfig_DefaultTimeout(t *testing.T) {
	path := writeTestConfig(t, `
servers:
  - name: foo
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SSH.ConnectTimeout != 10 {
		t.Errorf("expected default timeout 10, got %d", cfg.SSH.ConnectTimeout)
	}
}

func TestLoadConfig_EmptyServers(t *testing.T) {
	path := writeTestConfig(t, `
servers: []
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for empty servers")
	}
}

func TestLoadConfig_MissingNameAndPattern(t *testing.T) {
	path := writeTestConfig(t, `
servers:
  - has_gpu: true
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for entry without name or pattern")
	}
}

func TestLoadConfig_BothNameAndPattern(t *testing.T) {
	path := writeTestConfig(t, `
servers:
  - name: foo
    pattern: "bar{01..03}"
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for entry with both name and pattern")
	}
}

func TestLoadConfig_InvalidPattern(t *testing.T) {
	path := writeTestConfig(t, `
servers:
  - pattern: "badpattern"
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}
}

func TestLoadConfig_MalformedYAML(t *testing.T) {
	path := writeTestConfig(t, `
servers:
  - name: [invalid yaml structure
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/servers.yml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExpandServers_NameOnly(t *testing.T) {
	cfg := &ScoutConfig{
		Servers: []ServerEntry{
			{Name: "mantis", HasGPU: true},
			{Name: "cricket"},
		},
	}
	servers, err := ExpandServers(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if servers[0].Name != "mantis" || !servers[0].HasGPU {
		t.Errorf("unexpected first server: %+v", servers[0])
	}
	if servers[1].Name != "cricket" || servers[1].HasGPU {
		t.Errorf("unexpected second server: %+v", servers[1])
	}
}

func TestExpandServers_Pattern(t *testing.T) {
	cfg := &ScoutConfig{
		Servers: []ServerEntry{
			{Pattern: "beetle{01..05}", HasGPU: true},
		},
	}
	servers, err := ExpandServers(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 5 {
		t.Fatalf("expected 5 servers, got %d", len(servers))
	}

	expected := []string{"beetle01", "beetle02", "beetle03", "beetle04", "beetle05"}
	for i, srv := range servers {
		if srv.Name != expected[i] {
			t.Errorf("server %d: expected %s, got %s", i, expected[i], srv.Name)
		}
		if !srv.HasGPU {
			t.Errorf("server %d: expected HasGPU=true", i)
		}
	}
}

func TestExpandServers_MixedEntries(t *testing.T) {
	cfg := &ScoutConfig{
		Servers: []ServerEntry{
			{Name: "mantis", HasGPU: true},
			{Pattern: "beetle{01..03}", HasGPU: true},
			{Name: "cricket"},
		},
	}
	servers, err := ExpandServers(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 5 {
		t.Fatalf("expected 5 servers, got %d", len(servers))
	}

	names := make([]string, len(servers))
	for i, s := range servers {
		names[i] = s.Name
	}
	expectedNames := []string{"mantis", "beetle01", "beetle02", "beetle03", "cricket"}
	for i, name := range expectedNames {
		if names[i] != name {
			t.Errorf("server %d: expected %s, got %s", i, name, names[i])
		}
	}
}

func TestExpandPattern_ZeroPadding(t *testing.T) {
	tests := []struct {
		pattern  string
		expected []string
	}{
		{"node{1..3}", []string{"node1", "node2", "node3"}},
		{"node{01..03}", []string{"node01", "node02", "node03"}},
		{"node{001..003}", []string{"node001", "node002", "node003"}},
		{"gpu{08..10}", []string{"gpu08", "gpu09", "gpu10"}},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result, err := expandPattern(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d names, got %d", len(tt.expected), len(result))
			}
			for i, name := range result {
				if name != tt.expected[i] {
					t.Errorf("name %d: expected %s, got %s", i, tt.expected[i], name)
				}
			}
		})
	}
}

func TestExpandPattern_StartGtEnd(t *testing.T) {
	_, err := expandPattern("node{05..01}")
	if err == nil {
		t.Fatal("expected error when start > end")
	}
}

func TestExpandPattern_SingleValue(t *testing.T) {
	result, err := expandPattern("node{03..03}")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 name, got %d", len(result))
	}
	if result[0] != "node03" {
		t.Errorf("expected node03, got %s", result[0])
	}
}
