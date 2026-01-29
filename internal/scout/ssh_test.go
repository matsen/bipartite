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

// containsStr is a test helper for string containment checks.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
