package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH client connection
type Client struct {
	client    *ssh.Client
	config    *Config
	connected bool
	mu        sync.Mutex
}

// NewClient creates a new SSH client
func NewClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Expand key path
	keyPath, err := ExpandPath(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand key path: %w", err)
	}

	// Read private key
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", keyPath, err)
	}

	// Parse private key
	var signer ssh.Signer
	if cfg.Password != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cfg.Password))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// SSH config
	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
		Timeout:         cfg.Timeout,
	}

	// Connect
	addr := cfg.Address()
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{
		client:    client,
		config:    cfg,
		connected: true,
	}, nil
}

// NewClientWithPassword creates an SSH client using password auth
func NewClientWithPassword(cfg *Config, password string) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         cfg.Timeout,
	}

	addr := cfg.Address()
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{
		client:    client,
		config:    cfg,
		connected: true,
	}, nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	return c.client.Close()
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Exec executes a command on the remote server
func (c *Client) Exec(ctx context.Context, command string) (stdout, stderr []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, nil, fmt.Errorf("not connected")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	// Run command in goroutine to support context cancellation
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, try to close session
		session.Signal(ssh.SIGKILL)
		return nil, nil, ctx.Err()
	case err := <-done:
		return outBuf.Bytes(), errBuf.Bytes(), err
	}
}

// ExecSimple executes a command and returns combined output
func (c *Client) ExecSimple(command string) ([]byte, error) {
	ctx := context.Background()
	stdout, stderr, err := c.Exec(ctx, command)
	if err != nil {
		return append(stdout, stderr...), err
	}
	return stdout, nil
}

// Shell starts an interactive shell
func (c *Client) Shell() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up terminal
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // Disable echoing
		ssh.TTY_OP_ISPEED: 14400, // Input speed
		ssh.TTY_OP_OSPEED: 14400, // Output speed
	}

	// Request PTY
	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm"
	}

	if err := session.RequestPty(term, 80, 40, modes); err != nil {
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	// Set IO
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait for session to end
	return session.Wait()
}

// StartTunnel creates a local port forward
func (c *Client) StartTunnel(localPort int, remoteHost string, remotePort int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Listen on local port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return fmt.Errorf("failed to listen on %d: %w", localPort, err)
	}

	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)

	// Accept connections in background
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				return
			}

			// Forward connection
			go c.forwardConnection(localConn, remoteAddr)
		}
	}()

	return nil
}

// forwardConnection forwards a connection to remote
func (c *Client) forwardConnection(localConn net.Conn, remoteAddr string) {
	defer localConn.Close()

	// Dial remote through SSH
	remoteConn, err := c.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(localConn, remoteConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(remoteConn, localConn)
	}()

	wg.Wait()
}

// CopyFile copies a file to/from remote using cat (fallback if SFTP unavailable)
func (c *Client) CopyFile(src, dst string, toRemote bool) error {
	if toRemote {
		return c.copyFileToRemote(src, dst)
	}
	return c.copyFileFromRemote(src, dst)
}

func (c *Client) copyFileToRemote(localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Write to remote using cat
	cmd := fmt.Sprintf("cat > %q", remotePath)
	_, _, err = c.Exec(context.Background(), cmd)
	if err != nil {
		return fmt.Errorf("failed to write remote file: %w", err)
	}

	// Better approach using stdin
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = bytes.NewReader(data)
	return session.Run(fmt.Sprintf("cat > %q", remotePath))
}

func (c *Client) copyFileFromRemote(remotePath, localPath string) error {
	// Read from remote using cat
	cmd := fmt.Sprintf("cat %q", remotePath)
	stdout, _, err := c.Exec(context.Background(), cmd)
	if err != nil {
		return fmt.Errorf("failed to read remote file: %w", err)
	}

	// Write to local
	return os.WriteFile(localPath, stdout, 0644)
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

// RawClient returns the underlying ssh.Client
func (c *Client) RawClient() *ssh.Client {
	return c.client
}