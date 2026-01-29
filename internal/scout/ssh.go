package scout

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHClient abstracts SSH operations for testing.
type SSHClient interface {
	// RunCommand executes a command on the given server and returns combined output.
	RunCommand(server Server, command string) (string, error)
	// Close releases any resources held by the client.
	Close() error
}

// RealSSHClient implements SSHClient using actual SSH connections.
type RealSSHClient struct {
	sshConfig   SSHConfig
	agentConn   net.Conn // connection to SSH agent, closed in Close()
	agentClient agent.ExtendedAgent
	signers     []ssh.Signer
	username    string
}

// NewSSHClient creates a new SSH client that connects via the SSH agent.
func NewSSHClient(cfg SSHConfig) (*RealSSHClient, error) {
	// Check for SSH agent
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return nil, fmt.Errorf("SSH agent not running. Start with `eval $(ssh-agent)` and add keys with `ssh-add`")
	}

	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to SSH agent at %s: %w", authSock, err)
	}

	agentClient := agent.NewClient(conn)

	// Verify agent has keys
	keys, err := agentClient.List()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("listing SSH agent keys: %w", err)
	}
	if len(keys) == 0 {
		conn.Close()
		return nil, fmt.Errorf("SSH agent has no keys. Add keys with `ssh-add`")
	}

	signers, err := agentClient.Signers()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("getting SSH agent signers: %w", err)
	}

	// Determine SSH username from current OS user
	username := ""
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	return &RealSSHClient{
		sshConfig:   cfg,
		agentConn:   conn,
		agentClient: agentClient,
		signers:     signers,
		username:    username,
	}, nil
}

// RunCommand connects to a server (optionally via ProxyJump) and runs a command.
func (c *RealSSHClient) RunCommand(server Server, command string) (string, error) {
	timeout := time.Duration(c.sshConfig.ConnectTimeout) * time.Second

	// InsecureIgnoreHostKey disables host key verification. This is acceptable
	// for an internal tool on a trusted network where servers are managed
	// infrastructure. For untrusted networks, use a known_hosts file instead.
	clientConfig := &ssh.ClientConfig{
		User:            c.username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(c.signers...)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	var client *ssh.Client
	var jumpClient *ssh.Client
	var err error

	if c.sshConfig.ProxyJump != "" {
		client, jumpClient, err = c.dialViaProxy(server.Name, clientConfig, timeout)
		if jumpClient != nil {
			defer jumpClient.Close()
		}
	} else {
		client, err = ssh.Dial("tcp", server.Name+":22", clientConfig)
	}
	if err != nil {
		return "", c.wrapSSHError(err, server.Name)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("creating SSH session on %s: %w", server.Name, err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		// Command execution errors are non-fatal — we still have partial output
		return string(output), nil
	}
	return string(output), nil
}

// Close releases SSH client resources including the agent connection.
func (c *RealSSHClient) Close() error {
	if c.agentConn != nil {
		return c.agentConn.Close()
	}
	return nil
}

// dialViaProxy connects to the target server through a ProxyJump host.
// Returns both the target client and the jump client; caller must close both.
func (c *RealSSHClient) dialViaProxy(target string, config *ssh.ClientConfig, timeout time.Duration) (client *ssh.Client, jumpClient *ssh.Client, err error) {
	// See comment in RunCommand about InsecureIgnoreHostKey.
	proxyConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            config.Auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	jumpClient, err = ssh.Dial("tcp", c.sshConfig.ProxyJump+":22", proxyConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot reach proxy %s: %w", c.sshConfig.ProxyJump, err)
	}

	// Dial target through the proxy
	targetConn, err := jumpClient.Dial("tcp", target+":22")
	if err != nil {
		jumpClient.Close()
		return nil, nil, fmt.Errorf("cannot reach %s through proxy %s: %w", target, c.sshConfig.ProxyJump, err)
	}

	ncc, chans, reqs, err := ssh.NewClientConn(targetConn, target+":22", config)
	if err != nil {
		targetConn.Close()
		jumpClient.Close()
		return nil, nil, fmt.Errorf("SSH handshake with %s failed: %w", target, err)
	}

	return ssh.NewClient(ncc, chans, reqs), jumpClient, nil
}

// wrapSSHError produces actionable error messages based on SSH error types.
func (c *RealSSHClient) wrapSSHError(err error, server string) error {
	errStr := err.Error()

	// Check for common SSH error patterns
	switch {
	case strings.Contains(errStr, "no supported methods remain"):
		return fmt.Errorf("SSH authentication failed for %s. Check ~/.ssh/config and ensure your key is authorized", server)
	case strings.Contains(errStr, "i/o timeout") || strings.Contains(errStr, "connection timed out"):
		if c.sshConfig.ProxyJump != "" && strings.Contains(errStr, c.sshConfig.ProxyJump) {
			return fmt.Errorf("cannot reach proxy %s: connection timed out", c.sshConfig.ProxyJump)
		}
		return fmt.Errorf("connection to %s timed out", server)
	case strings.Contains(errStr, "connection refused"):
		return fmt.Errorf("connection refused by %s — is SSH running on the server?", server)
	default:
		return fmt.Errorf("SSH error connecting to %s: %w", server, err)
	}
}
