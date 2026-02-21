package scout

import (
	"fmt"
	"strings"
	"testing"
)

func TestWrapSSHError_AuthFailure(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{ProxyJump: "jump.example.com"},
	}

	err := client.wrapSSHError(
		fmt.Errorf("ssh: handshake failed: ssh: no supported methods remain"),
		"server01",
	)
	expected := "SSH authentication failed for server01"
	if !containsStr(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

func TestWrapSSHError_Timeout(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{},
	}

	err := client.wrapSSHError(
		fmt.Errorf("dial tcp: i/o timeout"),
		"server01",
	)
	expected := "connection to server01 timed out"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestWrapSSHError_ProxyTimeout(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{ProxyJump: "jump.example.com"},
	}

	err := client.wrapSSHError(
		fmt.Errorf("dial tcp jump.example.com:22: i/o timeout"),
		"server01",
	)
	expected := "cannot reach proxy jump.example.com: connection timed out"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestWrapSSHError_ConnectionRefused(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{},
	}

	err := client.wrapSSHError(
		fmt.Errorf("dial tcp: connection refused"),
		"server01",
	)
	expected := "connection refused by server01"
	if !containsStr(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

func TestWrapSSHError_GenericError(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{},
	}

	err := client.wrapSSHError(
		fmt.Errorf("something unexpected happened"),
		"server01",
	)
	expected := "SSH error connecting to server01"
	if !containsStr(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

func TestParseSSHConfigUser(t *testing.T) {
	config := `
Host jump.example.com
    User jdoe
    IdentityFile ~/.ssh/id_rsa

Host server01
    User admin
    ProxyJump jump.example.com

Host *.cluster.example.com
    User labuser

# Comment line
Host multi-pattern alt-name
    User multiuser
`

	tests := []struct {
		hostname string
		want     string
	}{
		{"jump.example.com", "jdoe"},
		{"server01", "admin"},
		{"gpu01.cluster.example.com", "labuser"},
		{"multi-pattern", "multiuser"},
		{"alt-name", "multiuser"},
		{"unknown-host", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			got := parseSSHConfigUser(config, tt.hostname)
			if got != tt.want {
				t.Errorf("parseSSHConfigUser(%q) = %q, want %q", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestParseSSHConfigUser_FirstMatchWins(t *testing.T) {
	config := `
Host server01
    User specific-user

Host *
    User default-user
`
	// Specific match should win over wildcard
	if got := parseSSHConfigUser(config, "server01"); got != "specific-user" {
		t.Errorf("expected specific-user, got %q", got)
	}
	// Wildcard should match unmatched hosts
	if got := parseSSHConfigUser(config, "other-server"); got != "default-user" {
		t.Errorf("expected default-user, got %q", got)
	}
}

func TestParseSSHConfigUser_EmptyConfig(t *testing.T) {
	if got := parseSSHConfigUser("", "server01"); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// containsStr is a test helper for string containment checks.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
