package scout

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrapSSHError_AuthFailure(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{ProxyJump: "jump.example.com"},
	}

	err := client.wrapSSHError(
		fmt.Errorf("ssh: handshake failed: ssh: no supported methods remain"),
		"server01", "localuser",
	)
	errMsg := err.Error()
	if !containsStr(errMsg, "SSH authentication failed for server01") {
		t.Errorf("expected auth failure mention, got %q", errMsg)
	}
	if !containsStr(errMsg, "localuser") {
		t.Errorf("expected username in error, got %q", errMsg)
	}
}

func TestWrapSSHError_Timeout(t *testing.T) {
	client := &RealSSHClient{
		sshConfig: SSHConfig{},
	}

	err := client.wrapSSHError(
		fmt.Errorf("dial tcp: i/o timeout"),
		"server01", "testuser",
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
		"server01", "testuser",
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
		"server01", "testuser",
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
		"server01", "testuser",
	)
	expected := "SSH error connecting to server01"
	if !containsStr(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

// containsStr is a test helper for string containment checks.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

// writeSSHConfig writes content to a temp file and returns its path.
func writeSSHConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSSHConfigUser_BasicMatch(t *testing.T) {
	path := writeSSHConfig(t, `
Host jump.cluster.edu
    User remoteuser
    HostName 10.0.0.1
`)
	got := sshConfigUserFromFile(path, "jump.cluster.edu")
	if got != "remoteuser" {
		t.Errorf("expected %q, got %q", "remoteuser", got)
	}
}

func TestSSHConfigUser_WildcardStar(t *testing.T) {
	path := writeSSHConfig(t, `
Host *
    User defaultuser
`)
	got := sshConfigUserFromFile(path, "anything.example.com")
	if got != "defaultuser" {
		t.Errorf("expected %q, got %q", "defaultuser", got)
	}
}

func TestSSHConfigUser_GlobPattern(t *testing.T) {
	path := writeSSHConfig(t, `
Host server*
    User clusteruser
`)
	got := sshConfigUserFromFile(path, "server01")
	if got != "clusteruser" {
		t.Errorf("expected %q, got %q", "clusteruser", got)
	}
}

func TestSSHConfigUser_NoMatch(t *testing.T) {
	path := writeSSHConfig(t, `
Host jump.cluster.edu
    User remoteuser
`)
	got := sshConfigUserFromFile(path, "other.host.com")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestSSHConfigUser_FirstMatchWins(t *testing.T) {
	path := writeSSHConfig(t, `
Host jump.cluster.edu
    User firstuser

Host jump.cluster.edu
    User seconduser
`)
	got := sshConfigUserFromFile(path, "jump.cluster.edu")
	if got != "firstuser" {
		t.Errorf("expected %q, got %q", "firstuser", got)
	}
}

func TestSSHConfigUser_CommentsAndBlanks(t *testing.T) {
	path := writeSSHConfig(t, `
# This is a comment
Host jump.cluster.edu

    # Indented comment
    User commentuser
`)
	got := sshConfigUserFromFile(path, "jump.cluster.edu")
	if got != "commentuser" {
		t.Errorf("expected %q, got %q", "commentuser", got)
	}
}

func TestSSHConfigUser_MissingFile(t *testing.T) {
	got := sshConfigUserFromFile("/nonexistent/path/config", "anything")
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}

func TestSSHConfigUser_MultiplePatterns(t *testing.T) {
	path := writeSSHConfig(t, `
Host jump1 jump2 jump3
    User multiuser
`)
	for _, host := range []string{"jump1", "jump2", "jump3"} {
		got := sshConfigUserFromFile(path, host)
		if got != "multiuser" {
			t.Errorf("host %s: expected %q, got %q", host, "multiuser", got)
		}
	}
}

func TestSSHConfigUser_SpecificBeforeWildcard(t *testing.T) {
	path := writeSSHConfig(t, `
Host jump.cluster.edu
    User specificuser

Host *
    User fallbackuser
`)
	got := sshConfigUserFromFile(path, "jump.cluster.edu")
	if got != "specificuser" {
		t.Errorf("expected %q, got %q", "specificuser", got)
	}

	// Non-matching host should fall through to wildcard
	got = sshConfigUserFromFile(path, "other.host.com")
	if got != "fallbackuser" {
		t.Errorf("expected %q, got %q", "fallbackuser", got)
	}
}
