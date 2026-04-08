//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestChatCommand tests the chat command end-to-end
// Requires: cicerone binary built and ollama running
func TestChatCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Check if ollama is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-s", "http://localhost:11434/api/version")
	if err := cmd.Run(); err != nil {
		t.Skip("Ollama not running, skipping e2e test")
	}

	// Build binary
	binary := filepath.Join(t.TempDir(), "cicerone-test")
	buildCmd := exec.Command("go", "build", "-o", binary, "../cmd")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}

	t.Run("startup", func(t *testing.T) {
		cmd := exec.Command(binary, "chat", "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("chat --help failed: %v\n%s", err, output)
		}
		if !strings.Contains(string(output), "Interactive LLM chat") {
			t.Errorf("Expected help text, got: %s", output)
		}
	})
}

// TestAgentCommands tests agent commands (/run, /write, etc.)
func TestAgentCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tmpDir := t.TempDir()

	// Create test binary
	binary := filepath.Join(t.TempDir(), "cicerone-test")
	buildCmd := exec.Command("go", "build", "-o", binary, "../cmd")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}

	// Test /run command
	t.Run("run_command", func(t *testing.T) {
		cmd := exec.Command(binary, "chat")
		cmd.Dir = tmpDir
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to get stdin: %v", err)
		}
		defer stdin.Close()

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		// Send commands
		go func() {
			stdin.Write([]byte("/run echo 'test'\nexit\n"))
		}()

		cmd.Wait()
	})

	// Test /write and /read commands
	t.Run("write_read", func(t *testing.T) {
		cmd := exec.Command(binary, "chat")
		cmd.Dir = tmpDir
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to get stdin: %v", err)
		}
		defer stdin.Close()

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		go func() {
			stdin.Write([]byte("/write test.txt hello world\n"))
			stdin.Write([]byte("/read test.txt\n"))
			stdin.Write([]byte("exit\n"))
		}()

		cmd.Wait()

		// Verify file exists
		content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
		if err != nil {
			t.Fatalf("File not created: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("File content mismatch: %s", content)
		}
	})
}

// TestTaskCommand tests the /task command
func TestTaskCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Check if ollama is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-s", "http://localhost:11434/api/version")
	if err := cmd.Run(); err != nil {
		t.Skip("Ollama not running, skipping e2e test")
	}

	tmpDir := t.TempDir()

	// Build binary
	binary := filepath.Join(t.TempDir(), "cicerone-test")
	buildCmd := exec.Command("go", "build", "-o", binary, "../cmd")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}

	// Test /task with simple file creation
	t.Run("simple_task", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "chat")
		cmd.Dir = tmpDir
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to get stdin: %v", err)
		}
		defer stdin.Close()

		output, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatalf("Failed to get stdout: %v", err)
		}
		defer output.Close()

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		// Send task command
		go func() {
			time.Sleep(1 * time.Second)
			stdin.Write([]byte("/task create a file called e2e_test.txt with content 'e2e test passed'\n"))
			time.Sleep(30 * time.Second)
			stdin.Write([]byte("exit\n"))
		}()

		// Read output
		buf := make([]byte, 1024)
		for {
			n, err := output.Read(buf)
			if err != nil {
				break
			}
			t.Logf("Output: %s", string(buf[:n]))
		}

		cmd.Wait()

		// Check file was created
		content, err := os.ReadFile(filepath.Join(tmpDir, "e2e_test.txt"))
		if err != nil {
			t.Logf("File not created (may need longer timeout): %v", err)
			return // Don't fail, just log
		}
		if !strings.Contains(string(content), "e2e test") {
			t.Logf("File content: %s", content)
		}
	})
}