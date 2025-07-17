package framework

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ToolkitFramework struct {
	ctx     *TestContext
	args    []string
	timeout time.Duration
	mu      sync.RWMutex
}

func NewToolkitFramework() *ToolkitFramework {
	ctx, err := NewTestContext("testkube-toolkit-")
	if err != nil {
		panic(err)
	}

	return &ToolkitFramework{
		ctx:     ctx,
		args:    []string{"--help"},
		timeout: 30 * time.Second,
	}
}

// WithCommand sets the command to run
func (h *ToolkitFramework) WithCommand(command string) *ToolkitFramework {
	h.args = []string{command}
	return h
}

// WithArgs sets the command arguments
func (h *ToolkitFramework) WithArgs(args ...string) *ToolkitFramework {
	h.args = append(h.args, args...)
	return h
}

// WithTimeout sets the execution timeout
func (h *ToolkitFramework) WithTimeout(timeout time.Duration) *ToolkitFramework {
	h.timeout = timeout
	return h
}

// Setup prepares the harness for execution
func (h *ToolkitFramework) Setup(t *testing.T) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Create necessary directories
	dirs := []string{
		"tmp",
		"data",
		"artifacts",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(h.ctx.TempDir.Path(), dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return err
		}
	}

	return nil
}

// Run executes the toolkit process
func (h *ToolkitFramework) Run(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find the toolkit binary
	binaryPath := h.findToolkitBinary()
	if binaryPath == "" {
		return fmt.Errorf("testworkflow-toolkit binary not found")
	}

	// Create the command
	cmd := exec.CommandContext(ctx, binaryPath, h.args...)
	cmd.Dir = h.ctx.TempDir.Path()

	// Connect output
	cmd.Stdout = h.ctx.Logger
	cmd.Stderr = h.ctx.Logger

	// Set environment variables - inherit from current process and add our custom path overrides
	cmd.Env = os.Environ()

	// Override the hardcoded paths to use our temp directory (same as init harness)
	tktwPath := filepath.Join(h.ctx.TempDir.Path(), ".tktw")
	termLogPath := filepath.Join(h.ctx.TempDir.Path(), "termination.log")

	// Set environment variables to override the default paths
	cmd.Env = append(cmd.Env,
		fmt.Sprintf("TESTKUBE_TW_INTERNAL_PATH=%s", tktwPath),
		fmt.Sprintf("TESTKUBE_TW_TERMINATION_LOG_PATH=%s", termLogPath),
	)

	// Create the .tktw directory if it doesn't exist
	if err := os.MkdirAll(tktwPath, 0755); err != nil {
		return fmt.Errorf("failed to create .tktw directory: %v", err)
	}

	// Execute the command
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	h.ctx.ProcessCapture.SetResult(exitCode, duration, false, err)

	if err != nil {
		return fmt.Errorf("toolkit process failed with exit code %d: %v", exitCode, err)
	}

	return nil
}

// findToolkitBinary locates the testworkflow-toolkit binary
func (h *ToolkitFramework) findToolkitBinary() string {
	// Look for the binary in common locations
	candidates := []string{
		"../../../build/testworkflow-toolkit/testworkflow-toolkit",
		"../../build/testworkflow-toolkit/testworkflow-toolkit",
		"build/testworkflow-toolkit/testworkflow-toolkit",
		"./testworkflow-toolkit",
	}

	for _, candidate := range candidates {
		if path, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Try to find it in PATH
	if path, err := exec.LookPath("testworkflow-toolkit"); err == nil {
		return path
	}

	return ""
}

// Cleanup cleans up the harness
func (h *ToolkitFramework) Cleanup(t *testing.T) error {
	return h.ctx.Cleanup()
}

func (h *ToolkitFramework) GetMetrics() *MetricsCapture {
	return h.ctx.MetricsCapture
}

func (h *ToolkitFramework) GetProcess() *ProcessCapture {
	return h.ctx.ProcessCapture
}

func (h *ToolkitFramework) GetTempDir() *TempDir {
	return h.ctx.TempDir
}

// AssertSuccess asserts that the toolkit process completed successfully
func (h *ToolkitFramework) AssertSuccess(t *testing.T) {
	h.ctx.ProcessCapture.AssertSuccess(t)
}

// AssertFailure asserts that the toolkit process failed with expected exit code
func (h *ToolkitFramework) AssertFailure(t *testing.T, expectedExitCode int) {
	h.ctx.ProcessCapture.AssertFailure(t, expectedExitCode)
}

// AssertOutputContains asserts that output contains expected text
func (h *ToolkitFramework) AssertOutputContains(t *testing.T, expected string) {
	lines := h.ctx.Logger.GetLines()
	combined := ""
	for _, line := range lines {
		combined += line
	}
	require.Contains(t, combined, expected, "Output should contain: %s", expected)
}

// CreateFile creates a file in the temp directory
func (h *ToolkitFramework) CreateFile(t *testing.T, filename string, content []byte) {
	filePath := filepath.Join(h.ctx.TempDir.Path(), filename)
	require.NoError(t, os.WriteFile(filePath, content, 0644), "Should be able to create file %s", filename)
}
