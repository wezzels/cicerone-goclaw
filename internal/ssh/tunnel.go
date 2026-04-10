package ssh

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Tunnel represents an SSH tunnel
type Tunnel struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
	client     *Client
	listener   net.Listener
	active     bool
	mu         sync.Mutex
}

// TunnelManager manages multiple SSH tunnels
type TunnelManager struct {
	tunnels map[int]*Tunnel
	client  *Client
	mu      sync.Mutex
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(client *Client) *TunnelManager {
	return &TunnelManager{
		tunnels: make(map[int]*Tunnel),
		client:  client,
	}
}

// CreateTunnel creates a new tunnel
func (tm *TunnelManager) CreateTunnel(localPort int, remoteHost string, remotePort int) (*Tunnel, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tunnels[localPort]; exists {
		return nil, fmt.Errorf("tunnel already exists on port %d", localPort)
	}

	tunnel := &Tunnel{
		LocalPort:  localPort,
		RemoteHost:  remoteHost,
		RemotePort: remotePort,
		client:     tm.client,
	}

	tm.tunnels[localPort] = tunnel
	return tunnel, nil
}

// Start starts the tunnel
func (t *Tunnel) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.active {
		return fmt.Errorf("tunnel already active")
	}

	// Listen on local port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", t.LocalPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", t.LocalPort, err)
	}

	t.listener = listener
	t.active = true

	// Accept connections in background
	go t.acceptLoop()

	return nil
}

// acceptLoop accepts incoming connections
func (t *Tunnel) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			t.mu.Lock()
			t.active = false
			t.mu.Unlock()
			return
		}

		go t.handleConnection(conn)
	}
}

// handleConnection handles a single connection
func (t *Tunnel) handleConnection(localConn net.Conn) {
	defer localConn.Close()

	// Dial remote through SSH
	remoteAddr := fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort)
	remoteConn, err := t.client.RawClient().Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		copyData(localConn, remoteConn)
	}()

	go func() {
		defer wg.Done()
		copyData(remoteConn, localConn)
	}()

	wg.Wait()
}

// copyData copies data between connections
func copyData(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			_, _ = dst.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// Stop stops the tunnel
func (t *Tunnel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.active {
		return nil
	}

	t.active = false
	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}

// IsActive returns whether the tunnel is active
func (t *Tunnel) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}

// LocalAddr returns the local address
func (t *Tunnel) LocalAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", t.LocalPort)
}

// RemoteAddr returns the remote address
func (t *Tunnel) RemoteAddr() string {
	return fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort)
}

// String returns a human-readable description
func (t *Tunnel) String() string {
	return fmt.Sprintf("%s -> %s", t.LocalAddr(), t.RemoteAddr())
}

// StopAll stops all tunnels
func (tm *TunnelManager) StopAll() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var lastErr error
	for _, tunnel := range tm.tunnels {
		if err := tunnel.Stop(); err != nil {
			lastErr = err
		}
	}
	tm.tunnels = make(map[int]*Tunnel)
	return lastErr
}

// List returns all tunnels
func (tm *TunnelManager) List() []*Tunnel {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tunnels := make([]*Tunnel, 0, len(tm.tunnels))
	for _, t := range tm.tunnels {
		tunnels = append(tunnels, t)
	}
	return tunnels
}

// Get returns a tunnel by local port
func (tm *TunnelManager) Get(localPort int) (*Tunnel, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tunnel, exists := tm.tunnels[localPort]
	if !exists {
		return nil, fmt.Errorf("no tunnel on port %d", localPort)
	}
	return tunnel, nil
}

// ForwardRemoteToLocal creates a reverse tunnel (remote -> local)
func (c *Client) ForwardRemoteToLocal(remotePort int, localAddr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	// Request remote forwarding
	listener, err := c.client.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", remotePort))
	if err != nil {
		return fmt.Errorf("failed to listen on remote: %w", err)
	}

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func() {
				defer conn.Close()

				// Connect to local
				localConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					return
				}
				defer localConn.Close()

				// Bidirectional copy
				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()
					copyConn(localConn, conn)
				}()

				go func() {
					defer wg.Done()
					copyConn(conn, localConn)
				}()

				wg.Wait()
			}()
		}
	}()

	return nil
}

// copyConn copies data between connections
func copyConn(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			_, _ = dst.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// Context with timeout helper
func (c *Client) ExecWithTimeout(command string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stdout, stderr, err := c.Exec(ctx, command)
	if err != nil {
		return append(stdout, stderr...), err
	}
	return stdout, nil
}