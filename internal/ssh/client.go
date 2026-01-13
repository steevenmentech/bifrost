package ssh

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// CLient represents an SSH client
type Client struct {
	config *ssh.ClientConfig
	client *ssh.Client
	host   string
	port   int
}

// NewClient() creates a new SSH client
func NewClient(host string, port int, username, password string) (*Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return &Client{
		config: config,
		host:   host,
		port:   port,
	}, nil
}

// Connect establishes the SSH connection
func (c *Client) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	client, err := ssh.Dial("tcp", addr, c.config)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	c.client = client
	return nil
}

// StartInteractiveSession starts an interactive shell session
func (c *Client) StartInteractiveSession() error {
	if c.client == nil {
		return fmt.Errorf("not connected")
	}

	// Create a session
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Get terminal size
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = 80
		height = 24
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	// Set up terminal for raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Connect input/output
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Handle window resize
	go c.handleResize(session)

	// Start remote shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait for session to finish
	if err := session.Wait(); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			// Session exited with non-zero status
			return fmt.Errorf("remote command exited with status %d", exitErr.ExitStatus())
		}
		return fmt.Errorf("session wait failed: %w", err)
	}

	return nil
}

// handleResize handles terminal window resize events
func (c *Client) handleResize(session *ssh.Session) {
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	for range sigwinch {
		width, height, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			continue
		}

		// Send window change request
		if err := session.WindowChange(height, width); err != nil {
			// Ignore error, window change is best-effort
			continue
		}
	}
}

// RunCommand runs a single command and returns the output
func (c *Client) RunCommand(command string) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("not connected")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %w", err)
	}

	return string(output), nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ConnectFromConfig creates and connects an SSH client using connection details
func ConnectFromConfig(host string, port int, username, password string) (*Client, error) {
	client, err := NewClient(host, port, username, password)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}
