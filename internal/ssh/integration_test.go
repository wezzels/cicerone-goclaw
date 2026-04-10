package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests require a real SSH server
// Run with: go test -run Integration ./internal/ssh/...
//
// Set environment variables for credentials:
//   SSH_TEST_HOST     - SSH server hostname (default: 10.0.0.117)
//   SSH_TEST_USER     - SSH username (default: wez)
//   SSH_TEST_PASSWORD - SSH password (optional, for password auth)
//   SSH_TEST_KEY      - Path to SSH private key (optional, for key auth)

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func TestIntegration_ClientWithKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	host := getEnvOrDefault("SSH_TEST_HOST", "10.0.0.117")
	user := getEnvOrDefault("SSH_TEST_USER", "wez")
	keyPath := getEnvOrDefault("SSH_TEST_KEY", os.Getenv("HOME")+"/.cicerone/keys/id_ed25519_darth")

	cfg := &Config{
		Host:    host,
		Port:    22,
		User:    user,
		KeyPath: keyPath,
		Timeout: 30 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient with key failed: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	// Test exec
	ctx := context.Background()
	stdout, _, err := client.Exec(ctx, "echo key-auth-success")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if string(stdout) != "key-auth-success\n" {
		t.Errorf("Exec stdout = %q, want 'key-auth-success'", string(stdout))
	}
}

func TestIntegration_ClientWithPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	host := getEnvOrDefault("SSH_TEST_HOST", "10.0.0.117")
	user := getEnvOrDefault("SSH_TEST_USER", "wez")
	password := os.Getenv("SSH_TEST_PASSWORD")

	if password == "" {
		t.Skip("SSH_TEST_PASSWORD not set, skipping password auth test")
	}

	cfg := &Config{
		Host:    host,
		Port:    22,
		User:    user,
		Timeout: 30 * time.Second,
	}

	client, err := NewClientWithPassword(cfg, password)
	if err != nil {
		t.Fatalf("NewClientWithPassword failed: %v", err)
	}
	defer client.Close()

	// Test connection
	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	// Test exec
	ctx := context.Background()
	stdout, stderr, err := client.Exec(ctx, "echo hello")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if string(stdout) != "hello\n" {
		t.Errorf("Exec stdout = %q, want 'hello'", string(stdout))
	}
	if len(stderr) > 0 {
		t.Errorf("Exec stderr = %s", stderr)
	}
}

func TestIntegration_ExecCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	host := getEnvOrDefault("SSH_TEST_HOST", "10.0.0.117")
	user := getEnvOrDefault("SSH_TEST_USER", "wez")

	// Prefer key auth, fall back to password
	keyPath := getEnvOrDefault("SSH_TEST_KEY", os.Getenv("HOME")+"/.cicerone/keys/id_ed25519_darth")
	password := os.Getenv("SSH_TEST_PASSWORD")

	var client *Client
	var err error

	cfg := &Config{
		Host:    host,
		Port:    22,
		User:    user,
		Timeout: 30 * time.Second,
	}

	// Try key auth first
	if _, keyErr := os.Stat(keyPath); keyErr == nil {
		cfg.KeyPath = keyPath
		client, err = NewClient(cfg)
	} else if password != "" {
		// Fall back to password
		client, err = NewClientWithPassword(cfg, password)
	} else {
		t.Skip("No SSH_TEST_PASSWORD or valid SSH_TEST_KEY, skipping")
	}

	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test multiple commands
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"pwd", "pwd", "/home/wez"},
		{"whoami", "whoami", "wez"},
		{"echo", "echo test", "test"},
		{"env", "echo $HOME", "/home/wez"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, _, err := client.Exec(ctx, tt.cmd)
			if err != nil {
				t.Errorf("Exec(%s) failed: %v", tt.cmd, err)
				return
			}
			// Trim newline for comparison
			got := string(stdout)
			if len(got) > 0 && got[len(got)-1] == '\n' {
				got = got[:len(got)-1]
			}
			if got != tt.want {
				t.Errorf("Exec(%s) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestIntegration_FileTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	host := getEnvOrDefault("SSH_TEST_HOST", "10.0.0.117")
	user := getEnvOrDefault("SSH_TEST_USER", "wez")
	keyPath := getEnvOrDefault("SSH_TEST_KEY", os.Getenv("HOME")+"/.cicerone/keys/id_ed25519_darth")
	password := os.Getenv("SSH_TEST_PASSWORD")

	cfg := &Config{
		Host:    host,
		Port:    22,
		User:    user,
		Timeout: 30 * time.Second,
	}

	var client *Client
	var err error

	// Try key auth first
	if _, keyErr := os.Stat(keyPath); keyErr == nil {
		cfg.KeyPath = keyPath
		client, err = NewClient(cfg)
	} else if password != "" {
		client, err = NewClientWithPassword(cfg, password)
	} else {
		t.Skip("No SSH_TEST_PASSWORD or valid SSH_TEST_KEY, skipping")
	}

	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create test content
	testContent := "test file content for cicerone ssh tests"
	localFile := filepath.Join(t.TempDir(), "test_upload.txt")

	// Write local test file
	if err := os.WriteFile(localFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Test Push (upload)
	remotePath := "/tmp/cicerone_test_upload.txt"
	transfer, err := NewTransfer(client)
	if err != nil {
		t.Fatalf("NewTransfer failed: %v", err)
	}
	defer transfer.Close()

	err = transfer.Push(localFile, remotePath)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify file was uploaded
	stdout, _, err := client.Exec(ctx, "cat "+remotePath)
	if err != nil {
		t.Fatalf("Exec cat failed: %v", err)
	}
	if string(stdout) != testContent {
		t.Errorf("Uploaded content = %q, want %q", string(stdout), testContent)
	}

	// Test Pull (download)
	localDownload := filepath.Join(t.TempDir(), "test_download.txt")
	err = transfer.Pull(remotePath, localDownload)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	// Verify downloaded content
	downloaded, err := os.ReadFile(localDownload)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(downloaded) != testContent {
		t.Errorf("Downloaded content = %q, want %q", string(downloaded), testContent)
	}

	// Cleanup remote file
	_, _, _ = client.Exec(ctx, "rm "+remotePath)
}

func TestIntegration_Shell(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	host := getEnvOrDefault("SSH_TEST_HOST", "10.0.0.117")
	user := getEnvOrDefault("SSH_TEST_USER", "wez")
	keyPath := getEnvOrDefault("SSH_TEST_KEY", os.Getenv("HOME")+"/.cicerone/keys/id_ed25519_darth")
	password := os.Getenv("SSH_TEST_PASSWORD")

	cfg := &Config{
		Host:    host,
		Port:    22,
		User:    user,
		Timeout: 30 * time.Second,
	}

	var client *Client
	var err error

	// Try key auth first
	if _, keyErr := os.Stat(keyPath); keyErr == nil {
		cfg.KeyPath = keyPath
		client, err = NewClient(cfg)
	} else if password != "" {
		client, err = NewClientWithPassword(cfg, password)
	} else {
		t.Skip("No SSH_TEST_PASSWORD or valid SSH_TEST_KEY, skipping")
	}

	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test shell session creation (doesn't run interactively in tests)
	// Just verify it can be created
	session, err := client.RawClient().NewSession()
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	session.Close()
}

func TestIntegration_ExecWithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	host := getEnvOrDefault("SSH_TEST_HOST", "10.0.0.117")
	user := getEnvOrDefault("SSH_TEST_USER", "wez")
	keyPath := getEnvOrDefault("SSH_TEST_KEY", os.Getenv("HOME")+"/.cicerone/keys/id_ed25519_darth")
	password := os.Getenv("SSH_TEST_PASSWORD")

	cfg := &Config{
		Host:    host,
		Port:    22,
		User:    user,
		Timeout: 30 * time.Second,
	}

	var client *Client
	var err error

	// Try key auth first
	if _, keyErr := os.Stat(keyPath); keyErr == nil {
		cfg.KeyPath = keyPath
		client, err = NewClient(cfg)
	} else if password != "" {
		client, err = NewClientWithPassword(cfg, password)
	} else {
		t.Skip("No SSH_TEST_PASSWORD or valid SSH_TEST_KEY, skipping")
	}

	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Test timeout
	result, err := client.ExecWithTimeout("echo test", 5*time.Second)
	if err != nil {
		t.Errorf("ExecWithTimeout failed: %v", err)
	}
	if string(result) != "test\n" {
		t.Errorf("ExecWithTimeout result = %q, want 'test'", string(result))
	}

	// Test command that should timeout
	_, err = client.ExecWithTimeout("sleep 10", 100*time.Millisecond)
	if err == nil {
		t.Error("ExecWithTimeout should timeout for slow command")
	}
}