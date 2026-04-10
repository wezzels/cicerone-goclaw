// Package vm provides VM management for workspace deployment.
package vm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/ssh"
)

// KeyManager handles SSH key generation and deployment for VMs.
type KeyManager struct {
	keysDir string
}

// NewKeyManager creates a new key manager.
func NewKeyManager() (*KeyManager, error) {
	keysDir, err := GetVMKeysDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys directory: %w", err)
	}

	if err := EnsureKeyDir(); err != nil {
		return nil, fmt.Errorf("failed to create keys directory: %w", err)
	}

	return &KeyManager{keysDir: keysDir}, nil
}

// KeyInfo contains information about an SSH key pair.
type KeyInfo struct {
	PrivateKeyPath string
	PublicKeyPath  string
	PublicKey      string
	Comment        string
}

// GenerateKey generates a new Ed25519 SSH key pair for a VM.
func (km *KeyManager) GenerateKey(vmName string, comment string) (*KeyInfo, error) {
	keyPath, err := GetVMKeyPath(vmName)
	if err != nil {
		return nil, err
	}

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		// Key exists, return existing key info
		return km.GetKeyInfo(vmName)
	}

	// Ensure keys directory exists
	if err := EnsureKeyDir(); err != nil {
		return nil, fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Use ssh-keygen to generate proper Ed25519 key
	if comment == "" {
		comment = fmt.Sprintf("cicerone-%s", vmName)
	}

	cmd := exec.Command("ssh-keygen",
		"-t", "ed25519",
		"-f", keyPath,
		"-C", comment,
		"-N", "", // No passphrase
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w\n%s", err, output)
	}

	return km.GetKeyInfo(vmName)
}

// GetKeyInfo returns information about an existing key.
func (km *KeyManager) GetKeyInfo(vmName string) (*KeyInfo, error) {
	keyPath, err := GetVMKeyPath(vmName)
	if err != nil {
		return nil, err
	}

	pubKeyPath := keyPath + ".pub"

	// Check if private key exists
	if _, err := os.Stat(keyPath); err != nil {
		return nil, fmt.Errorf("private key not found: %w", err)
	}

	// Read public key
	pubKeyData, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, fmt.Errorf("public key not found: %w", err)
	}

	// Parse public key to extract comment
	pubKeyStr := strings.TrimSpace(string(pubKeyData))
	parts := strings.SplitN(pubKeyStr, " ", 3)
	comment := ""
	if len(parts) >= 3 {
		comment = parts[2]
	}

	return &KeyInfo{
		PrivateKeyPath: keyPath,
		PublicKeyPath:  pubKeyPath,
		PublicKey:      pubKeyStr,
		Comment:        comment,
	}, nil
}

// KeyExists checks if a key already exists for the VM.
func (km *KeyManager) KeyExists(vmName string) bool {
	keyPath, err := GetVMKeyPath(vmName)
	if err != nil {
		return false
	}

	_, err = os.Stat(keyPath)
	return err == nil
}

// RemoveKey removes the key pair for a VM.
func (km *KeyManager) RemoveKey(vmName string) error {
	keyPath, err := GetVMKeyPath(vmName)
	if err != nil {
		return err
	}

	// Remove private key
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove private key: %w", err)
	}

	// Remove public key
	pubKeyPath := keyPath + ".pub"
	if err := os.Remove(pubKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove public key: %w", err)
	}

	return nil
}

// DeployKey deploys an SSH public key to a VM.
func (km *KeyManager) DeployKey(ctx context.Context, vmName, host string, port int, user string, publicKey []byte) error {
	// Password required for initial key deployment
	// This will be handled by the CLI which prompts for password
	return fmt.Errorf("password required for initial key deployment - use DeployKeyWithPassword")
}

// DeployKeyWithPassword deploys SSH public key using password authentication.
func (km *KeyManager) DeployKeyWithPassword(ctx context.Context, vmName, host string, port int, user, password string, publicKey []byte) error {
	sshCfg := &ssh.Config{
		Host:    host,
		Port:    port,
		User:    user,
		Timeout: 30,
	}

	client, err := ssh.NewClientWithPassword(sshCfg, password)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Ensure .ssh directory exists
	_, _, err = client.Exec(ctx, "mkdir -p ~/.ssh && chmod 700 ~/.ssh")
	if err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Add public key to authorized_keys
	cmd := fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", strings.TrimSpace(string(publicKey)))
	_, _, err = client.Exec(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to add key to authorized_keys: %w", err)
	}

	return nil
}

// DeployKeyWithKey deploys a new key using an existing key for authentication.
func (km *KeyManager) DeployKeyWithKey(ctx context.Context, vmName, host string, port int, user, existingKeyPath string, publicKey []byte) error {
	sshCfg := &ssh.Config{
		Host:    host,
		Port:    port,
		User:    user,
		KeyPath: existingKeyPath,
		Timeout: 30,
	}

	client, err := ssh.NewClient(sshCfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Add public key to authorized_keys
	cmd := fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys", strings.TrimSpace(string(publicKey)))
	_, _, err = client.Exec(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to add key to authorized_keys: %w", err)
	}

	return nil
}

// TestKey tests if an SSH key can authenticate to a VM.
func (km *KeyManager) TestKey(ctx context.Context, host string, port int, user, keyPath string) error {
	sshCfg := &ssh.Config{
		Host:    host,
		Port:    port,
		User:    user,
		KeyPath: keyPath,
		Timeout: 10,
	}

	client, err := ssh.NewClient(sshCfg)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Run simple test command
	_, _, err = client.Exec(ctx, "echo 'key test successful'")
	if err != nil {
		return fmt.Errorf("key authentication failed: %w", err)
	}

	return nil
}

// SetupKeyForVM sets up SSH key for a VM, generating if needed.
// Returns the key info and whether key was newly generated.
func (km *KeyManager) SetupKeyForVM(ctx context.Context, vmName string, generateIfNeeded bool) (*KeyInfo, bool, error) {
	// Check if key already exists
	if km.KeyExists(vmName) {
		info, err := km.GetKeyInfo(vmName)
		if err != nil {
			return nil, false, err
		}
		return info, false, nil
	}

	if !generateIfNeeded {
		return nil, false, fmt.Errorf("key not found for VM %s", vmName)
	}

	// Generate new key
	comment := fmt.Sprintf("cicerone-%s", vmName)
	info, err := km.GenerateKey(vmName, comment)
	if err != nil {
		return nil, false, err
	}

	return info, true, nil
}

// AddToKnownHosts adds a VM's host key to known_hosts.
func (km *KeyManager) AddToKnownHosts(host string, port int) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	// Ensure .ssh directory exists
	if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
		return err
	}

	// Use ssh-keyscan to get host key
	addr := fmt.Sprintf("%s:%d", host, port)
	if port == 22 {
		addr = host
	}

	cmd := exec.Command("ssh-keyscan", "-t", "ed25519", addr)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to scan host keys: %w", err)
	}

	// Check if already in known_hosts
	existing, err := os.ReadFile(knownHostsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Append if not already present
	hostKeyLine := strings.TrimSpace(string(output))
	if !bytes.Contains(existing, []byte(hostKeyLine)) {
		f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.WriteString(hostKeyLine + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// RemoveFromKnownHosts removes a VM's host key from known_hosts.
func (km *KeyManager) RemoveFromKnownHosts(host string, port int) error {
	addr := host
	if port != 22 {
		addr = fmt.Sprintf("[%s]:%d", host, port)
	}

	cmd := exec.Command("ssh-keygen", "-R", addr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove host key: %w", err)
	}

	return nil
}

// AddToSSHConfig adds a host entry to ~/.ssh/config.
func (km *KeyManager) AddToSSHConfig(vmName, host string, port int, keyPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".ssh", "config")

	// Ensure .ssh directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return err
	}

	// Read existing config
	existing, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if entry already exists
	if strings.Contains(string(existing), "Host "+vmName) {
		return nil // Already configured
	}

	// Create new entry
	expandedKeyPath := keyPath
	if strings.HasPrefix(keyPath, home) {
		expandedKeyPath = "~" + strings.TrimPrefix(keyPath, home)
	}

	entry := fmt.Sprintf(`
Host %s
    HostName %s
    Port %d
    User root
    IdentityFile %s
    StrictHostKeyChecking accept-new
`, vmName, host, port, expandedKeyPath)

	// Append entry
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return err
	}

	return nil
}

// RemoveFromSSHConfig removes a host entry from ~/.ssh/config.
func (km *KeyManager) RemoveFromSSHConfig(vmName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".ssh", "config")

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Remove host block
	lines := strings.Split(string(content), "\n")
	var newLines []string
	skip := false

	for _, line := range lines {
		if strings.HasPrefix(line, "Host "+vmName) {
			skip = true
			continue
		}
		if skip && strings.HasPrefix(line, "Host ") && !strings.HasPrefix(line, "Host "+vmName) {
			skip = false
		}
		if !skip {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// ReadPassword reads password from terminal (for interactive use).
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Newline after password
	return string(password), err
}

// Private helper functions