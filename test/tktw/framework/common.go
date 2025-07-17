package framework

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// MetricsCapture captures metrics from test processes
type MetricsCapture struct {
	metricsDir string
	files      map[string][]byte
}

func NewMetricsCapture(tempDir string) *MetricsCapture {
	metricsDir := filepath.Join(tempDir, "metrics")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		// Log error but continue - metrics capture is not critical
		fmt.Printf("failed to create metrics directory: %v\n", err)
	}
	return &MetricsCapture{
		metricsDir: metricsDir,
		files:      make(map[string][]byte),
	}
}

func (m *MetricsCapture) GetMetricsDir() string {
	return m.metricsDir
}

// ReadMetricsFile reads a specific metrics file
func (m *MetricsCapture) ReadMetricsFile(filename string) ([]byte, error) {
	if data, exists := m.files[filename]; exists {
		return data, nil
	}

	filePath := filepath.Join(m.metricsDir, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	m.files[filename] = data
	return data, nil
}

// ListMetricsFiles returns all available metrics files
func (m *MetricsCapture) ListMetricsFiles() ([]string, error) {
	files, err := os.ReadDir(m.metricsDir)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, file := range files {
		if !file.IsDir() {
			filenames = append(filenames, file.Name())
		}
	}

	return filenames, nil
}

type TempDir struct {
	path string
}

func NewTempDir(prefix string) (*TempDir, error) {
	path, err := os.MkdirTemp("", prefix)
	if err != nil {
		return nil, err
	}

	return &TempDir{path: path}, nil
}

func (td *TempDir) Path() string {
	return td.path
}

func (td *TempDir) Cleanup() error {
	return os.RemoveAll(td.path)
}

// ProcessCapture captures process information
type ProcessCapture struct {
	ExitCode  int
	Duration  time.Duration
	Timeout   bool
	Error     error
	startTime time.Time
}

func NewProcessCapture() *ProcessCapture {
	return &ProcessCapture{
		ExitCode: -1,
		Duration: 0,
		Timeout:  false,
		Error:    nil,
	}
}

// Reset resets the capture state
func (p *ProcessCapture) Reset() {
	p.ExitCode = -1
	p.Duration = 0
	p.Timeout = false
	p.Error = nil
	p.startTime = time.Time{}
}

// StartCapture marks the start of capture
func (p *ProcessCapture) StartCapture() {
	p.startTime = time.Now()
}

// StopCapture marks the end of capture
func (p *ProcessCapture) StopCapture() {
	if !p.startTime.IsZero() {
		p.Duration = time.Since(p.startTime)
	}
}

func (p *ProcessCapture) SetResult(exitCode int, duration time.Duration, timeout bool, err error) {
	p.ExitCode = exitCode
	p.Duration = duration
	p.Timeout = timeout
	p.Error = err
}

// AssertSuccess checks if the process completed successfully
func (p *ProcessCapture) AssertSuccess(t *testing.T) {
	require.NoError(t, p.Error, "process error")
	require.Equal(t, 0, p.ExitCode, "process exit code != 0")
	require.False(t, p.Timeout, "process timed out")
}

// AssertFailure checks if the process failed with expected exit code
func (p *ProcessCapture) AssertFailure(t *testing.T, expectedExitCode int) {
	require.Equal(t, expectedExitCode, p.ExitCode, "process exit code != %d", expectedExitCode)
	require.False(t, p.Timeout, "process timed out")
}

// LogWriter wraps io.Writer to capture log output
type LogWriter struct {
	writer io.Writer
	lines  []string
}

func NewLogWriter(writer io.Writer) *LogWriter {
	return &LogWriter{
		writer: writer,
		lines:  make([]string, 0),
	}
}

// Write implements io.Writer
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	line := string(p)
	lw.lines = append(lw.lines, line)

	if lw.writer != nil {
		return lw.writer.Write(p)
	}

	return len(p), nil
}

func (lw *LogWriter) GetLines() []string {
	return lw.lines
}

type TestContext struct {
	TempDir        *TempDir
	MetricsCapture *MetricsCapture
	ProcessCapture *ProcessCapture
	Logger         *LogWriter
}

func NewTestContext(prefix string) (*TestContext, error) {
	tempDir, err := NewTempDir(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return &TestContext{
		TempDir:        tempDir,
		MetricsCapture: NewMetricsCapture(tempDir.Path()),
		ProcessCapture: NewProcessCapture(),
		Logger:         NewLogWriter(os.Stdout),
	}, nil
}

func (tc *TestContext) Cleanup() error {
	return tc.TempDir.Cleanup()
}

// ProcessInfo represents captured process information
type ProcessInfo struct {
	ExitCode int
	Duration time.Duration
	Timeout  bool
	Error    error
}

// GetProcessInfo returns the captured process information
func (p *ProcessCapture) GetProcessInfo() ProcessInfo {
	return ProcessInfo{
		ExitCode: p.ExitCode,
		Duration: p.Duration,
		Timeout:  p.Timeout,
		Error:    p.Error,
	}
}
