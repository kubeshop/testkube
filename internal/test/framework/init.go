package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runner"
)

// testExecutionMutex ensures proper test isolation by preventing concurrent test execution
var testExecutionMutex sync.Mutex

type InitTestFramework struct {
	ctx         *TestContext
	timeout     time.Duration
	envSnapshot map[string]string
	isSetup     bool
	mu          sync.Mutex
}

func NewInitTestFramework() *InitTestFramework {
	ctx, err := NewTestContext("testkube-init-")
	if err != nil {
		panic(fmt.Errorf("failed to create test context: %v", err))
	}

	return &InitTestFramework{
		ctx:         ctx,
		timeout:     30 * time.Second,
		envSnapshot: make(map[string]string),
	}
}

// WithTimeout sets the execution timeout
func (f *InitTestFramework) WithTimeout(timeout time.Duration) *InitTestFramework {
	f.timeout = timeout
	return f
}

// Setup initializes the test environment with proper isolation
func (f *InitTestFramework) Setup(t *testing.T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isSetup {
		return fmt.Errorf("framework already set up")
	}

	// Acquire global test lock to prevent concurrent test execution
	testExecutionMutex.Lock()

	// Create test directory structure
	if err := f.createTestEnvironment(); err != nil {
		testExecutionMutex.Unlock()
		return fmt.Errorf("failed to create test environment: %v", err)
	}

	// Configure environment variables with proper isolation
	if err := f.configureEnvironment(); err != nil {
		testExecutionMutex.Unlock()
		return fmt.Errorf("failed to configure environment: %v", err)
	}

	// Clear any existing singleton state
	f.clearSingletonState()

	f.isSetup = true
	return nil
}

// Run executes the init process
func (f *InitTestFramework) Run(ctx context.Context) error {
	return f.runWithGroup(ctx, 0)
}

// RunGroup executes a specific action group
func (f *InitTestFramework) RunGroup(ctx context.Context, groupIndex int) error {
	return f.runWithGroup(ctx, groupIndex)
}

// Cleanup restores the environment and releases resources
func (f *InitTestFramework) Cleanup(t *testing.T) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.isSetup {
		return nil
	}

	// Restore original environment
	f.restoreEnvironment()

	// Clean up test resources
	var cleanupErr error
	if f.ctx != nil && f.ctx.TempDir != nil {
		cleanupErr = f.ctx.TempDir.Cleanup()
	}

	// Always release the global lock
	testExecutionMutex.Unlock()

	f.isSetup = false
	return cleanupErr
}

// GetTempDir returns the temporary directory
func (f *InitTestFramework) GetTempDir() *TempDir {
	return f.ctx.TempDir
}

// GetProcess returns the captured process information
func (f *InitTestFramework) GetProcess() ProcessInfo {
	return f.ctx.ProcessCapture.GetProcessInfo()
}

// SetTempDir allows using an existing temporary directory
func (f *InitTestFramework) SetTempDir(tempDir *TempDir) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.ctx.TempDir = tempDir
	f.configureEnvironment()

	// When using an existing temp dir, we consider the framework as set up
	// but we don't acquire the global lock since the parent test already has it
	f.isSetup = true
}

// Private methods

func (f *InitTestFramework) createTestEnvironment() error {
	tempPath := f.ctx.TempDir.Path()

	// Create directory structure expected by init process
	dirs := []string{
		"tmp",
		".tktw",
		".tktw/transfer",
		".tktw/bin",
		"data",
		"data/.testkube",
		"data/.testkube/internal",
		"data/.testkube/internal/metrics",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create termination log file
	termLogPath := filepath.Join(tempPath, "termination.log")
	if err := os.WriteFile(termLogPath, []byte{}, 0666); err != nil {
		return fmt.Errorf("failed to create termination log: %v", err)
	}

	// Create metrics directory for step
	stepRef := os.Getenv("TK_REF")
	if stepRef == "" {
		stepRef = "r6lxv49"
	}
	stepMetricsDir := filepath.Join(tempPath, "data", ".testkube", "internal", "metrics", stepRef)
	if err := os.MkdirAll(stepMetricsDir, 0755); err != nil {
		return fmt.Errorf("failed to create step metrics directory: %v", err)
	}

	// Create mock binaries
	if err := f.createMockBinaries(); err != nil {
		return fmt.Errorf("failed to create mock binaries: %v", err)
	}

	return nil
}

func (f *InitTestFramework) createMockBinaries() error {
	tempPath := f.ctx.TempDir.Path()

	// Mock shell script content
	mockShellScript := `#!/bin/sh
echo "$@"
exit 0`

	// Create init binary
	initPath := filepath.Join(tempPath, "init")
	if err := os.WriteFile(initPath, []byte(mockShellScript), 0755); err != nil {
		return fmt.Errorf("failed to create init binary: %v", err)
	}

	// Create toolkit binary
	toolkitPath := filepath.Join(tempPath, "toolkit")
	if err := os.WriteFile(toolkitPath, []byte(mockShellScript), 0755); err != nil {
		return fmt.Errorf("failed to create toolkit binary: %v", err)
	}

	// Create busybox directory and utilities
	busyboxDir := filepath.Join(tempPath, ".tktw-bin")
	if err := os.MkdirAll(busyboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create busybox directory: %v", err)
	}

	// Create shell utilities
	for _, util := range []string{"sh", "cp", "chmod", "true"} {
		utilPath := filepath.Join(busyboxDir, util)
		if err := os.WriteFile(utilPath, []byte(mockShellScript), 0755); err != nil {
			return fmt.Errorf("failed to create %s utility: %v", util, err)
		}
	}

	// Create shell at expected location
	shellPath := filepath.Join(tempPath, ".tktw", "bin", "sh")
	if err := os.WriteFile(shellPath, []byte(mockShellScript), 0755); err != nil {
		return fmt.Errorf("failed to create shell binary: %v", err)
	}

	return nil
}

func (f *InitTestFramework) configureEnvironment() error {
	tempPath := f.ctx.TempDir.Path()

	// Define test environment variables
	testEnv := map[string]string{
		"TESTKUBE_TW_INTERNAL_PATH":        filepath.Join(tempPath, ".tktw"),
		"TESTKUBE_TW_TERMINATION_LOG_PATH": filepath.Join(tempPath, "termination.log"),
		"TESTKUBE_TW_STATE_PATH":           filepath.Join(tempPath, ".tktw", "state"),
		"TESTKUBE_TW_INIT_BINARY_PATH":     filepath.Join(tempPath, "init"),
		"TESTKUBE_TW_TOOLKIT_BINARY_PATH":  filepath.Join(tempPath, "toolkit"),
		"TESTKUBE_TW_BUSYBOX_BINARY_PATH":  filepath.Join(tempPath, ".tktw-bin"),
	}

	// Backup and set environment variables
	for key, value := range testEnv {
		if _, exists := f.envSnapshot[key]; !exists {
			f.envSnapshot[key] = os.Getenv(key)
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set %s: %v", key, err)
		}
	}

	// Update global constants to match test environment
	f.updateGlobalConstants()

	return nil
}

func (f *InitTestFramework) updateGlobalConstants() {
	constants.ControlServerPort = 0

	constants.InternalPath = os.Getenv("TESTKUBE_TW_INTERNAL_PATH")
	if constants.InternalPath == "" {
		constants.InternalPath = "/.tktw"
	}

	constants.TerminationLogPath = os.Getenv("TESTKUBE_TW_TERMINATION_LOG_PATH")
	if constants.TerminationLogPath == "" {
		constants.TerminationLogPath = "/dev/termination-log"
	}

	constants.InternalBinPath = filepath.Join(constants.InternalPath, "bin")
	constants.InitPath = filepath.Join(constants.InternalPath, "init")
	constants.ToolkitPath = filepath.Join(constants.InternalPath, "toolkit")

	constants.StatePath = os.Getenv("TESTKUBE_TW_STATE_PATH")
	if constants.StatePath == "" {
		constants.StatePath = filepath.Join(constants.InternalPath, "state")
	}
}

func (f *InitTestFramework) clearSingletonState() {
	data.ClearState()
	orchestration.Setup = nil
}

func (f *InitTestFramework) restoreEnvironment() {
	for key, value := range f.envSnapshot {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
	f.envSnapshot = make(map[string]string)
}

func (f *InitTestFramework) runWithGroup(ctx context.Context, groupIndex int) error {
	if !f.isSetup {
		return fmt.Errorf("framework not set up")
	}

	// Ensure environment is properly configured
	f.updateGlobalConstants()

	// Clear singleton state and reinitialize
	f.clearSingletonState()
	orchestration.Initialize()

	// Capture process execution
	f.ctx.ProcessCapture.Reset()
	f.ctx.ProcessCapture.StartCapture()
	defer f.ctx.ProcessCapture.StopCapture()

	// Execute with panic recovery
	var panicErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("panic: %v", r)
				f.ctx.ProcessCapture.ExitCode = 1
			}
		}()

		exitCode, err := runner.RunInitWithContext(ctx, groupIndex)
		if err != nil {
			// Store the error but continue to set the exit code
			panicErr = err
		}
		f.ctx.ProcessCapture.ExitCode = exitCode
	}()

	if panicErr != nil {
		return panicErr
	}

	if f.ctx.ProcessCapture.ExitCode != 0 {
		return fmt.Errorf("init process failed with exit code %d", f.ctx.ProcessCapture.ExitCode)
	}

	return nil
}
