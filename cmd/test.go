package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/crab-meat-repos/cicerone-goclaw/internal/ssh"
	"github.com/crab-meat-repos/cicerone-goclaw/internal/workspace"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [path]",
	Short: "Run tests in workspace",
	Long: `Run tests locally or on a remote host via SSH.

Supports Go test patterns and remote execution.

Examples:
  cicerone test                    # Test current directory
  cicerone test ./pkg/...          # Test package
  cicerone test --cover            # With coverage
  cicerone test --remote darth     # Run on remote host
  cicerone test -run TestName      # Run specific test
  cicerone test --bench .          # Run benchmarks`,
	RunE: runTest,
}

var (
	testRemote   string
	testCover    bool
	testVerbose  bool
	testRun      string
	testBench    string
	testTimeout  time.Duration
	testParallel int
)

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&testRemote, "remote", "r", "", "remote host name (from SSH config)")
	testCmd.Flags().BoolVarP(&testCover, "cover", "c", false, "enable coverage")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "verbose output")
	testCmd.Flags().StringVar(&testRun, "run", "", "run tests matching pattern")
	testCmd.Flags().StringVar(&testBench, "bench", "", "run benchmarks matching pattern")
	testCmd.Flags().DurationVar(&testTimeout, "timeout", 60*time.Second, "test timeout")
	testCmd.Flags().IntVarP(&testParallel, "parallel", "p", 0, "parallel tests (default GOMAXPROCS)")
}

func runTest(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Build test command
	testArgs := []string{"test"}
	if testVerbose {
		testArgs = append(testArgs, "-v")
	}
	if testCover {
		testArgs = append(testArgs, "-cover")
	}
	if testRun != "" {
		testArgs = append(testArgs, "-run", testRun)
	}
	if testBench != "" {
		testArgs = append(testArgs, "-bench", testBench)
	}
	if testParallel > 0 {
		testArgs = append(testArgs, "-parallel", fmt.Sprintf("%d", testParallel))
	}
	testArgs = append(testArgs, path)

	// Run locally or remotely
	if testRemote != "" {
		return runRemoteTest(testArgs)
	}

	return runLocalTest(testArgs)
}

func runLocalTest(args []string) error {
	fmt.Printf("Running: go %s\n", strings.Join(args, " "))

	// Create workspace
	w, err := workspace.New(".")
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create executor
	exec := workspace.NewExecutor(w)
	exec.SetTimeout(testTimeout)

	// Run test
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	output, err := exec.RunWithContext(ctx, "go", args...)
	if len(output) > 0 {
		fmt.Print(string(output))
	}

	if err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	return nil
}

func runRemoteTest(args []string) error {
	// Load SSH hosts
	hosts, err := loadSSHHosts()
	if err != nil {
		return fmt.Errorf("failed to load hosts: %w", err)
	}

	h, exists := hosts[testRemote]
	if !exists {
		return fmt.Errorf("host '%s' not found", testRemote)
	}

	fmt.Printf("Running tests on %s...\n", testRemote)
	fmt.Printf("Command: go %s\n", strings.Join(args, " "))

	// Create SSH client
	cfg := ssh.ConfigFromHostAlias(&h)
	cfg.Timeout = testTimeout

	client, err := ssh.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer client.Close()

	// Run test
	ctx := context.Background()
	cmd := fmt.Sprintf("go %s", strings.Join(args, " "))
	stdout, stderr, err := client.Exec(ctx, cmd)

	if len(stdout) > 0 {
		fmt.Print(string(stdout))
	}
	if len(stderr) > 0 {
		fmt.Fprint(os.Stderr, string(stderr))
	}

	if err != nil {
		return fmt.Errorf("remote test failed: %w", err)
	}

	fmt.Println("✓ Tests passed on remote")
	return nil
}

// Build and run tests with coverage report
var testCoverCmd = &cobra.Command{
	Use:   "cover",
	Short: "Generate coverage report",
	Long:  `Run tests with coverage and generate HTML report.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Run tests with coverage
		testCmd := exec.Command("go", "test", "-coverprofile=coverage.out", "./...")
		testCmd.Stdout = os.Stdout
		testCmd.Stderr = os.Stderr

		if err := testCmd.Run(); err != nil {
			return fmt.Errorf("tests failed: %w", err)
		}

		// Generate HTML
		coverCmd := exec.Command("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
		if err := coverCmd.Run(); err != nil {
			return fmt.Errorf("coverage report failed: %w", err)
		}

		fmt.Println("✓ Coverage report generated: coverage.html")
		return nil
	},
}

// Benchmark tests
var testBenchCmd = &cobra.Command{
	Use:   "bench [pattern]",
	Short: "Run benchmarks",
	Long:  `Run benchmarks matching pattern.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := "."
		if len(args) > 0 {
			pattern = args[0]
		}

		benchCmd := exec.Command("go", "test", "-bench", pattern, "-benchmem", "./...")
		benchCmd.Stdout = os.Stdout
		benchCmd.Stderr = os.Stderr

		return benchCmd.Run()
	},
}

func init() {
	testCmd.AddCommand(testCoverCmd)
	testCmd.AddCommand(testBenchCmd)
}