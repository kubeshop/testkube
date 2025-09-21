package testworkflows

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func TestGetIterationDelay(t *testing.T) {
	tests := []struct {
		name      string
		iteration int
		expected  time.Duration
	}{
		{
			name:      "below first threshold",
			iteration: 3,
			expected:  500 * time.Millisecond,
		},
		{
			name:      "between thresholds",
			iteration: 50,
			expected:  1 * time.Second,
		},
		{
			name:      "above second threshold",
			iteration: 200,
			expected:  5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIterationDelay(tt.iteration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTargetMap(t *testing.T) {
	tests := []struct {
		name     string
		targets  []string
		expected map[string][]string
	}{
		{
			name:     "single value",
			targets:  []string{"env=production"},
			expected: map[string][]string{"env": {"production"}},
		},
		{
			name:     "comma-separated values",
			targets:  []string{"env=dev,staging,prod"},
			expected: map[string][]string{"env": {"dev", "staging", "prod"}},
		},
		{
			name:    "multiple keys",
			targets: []string{"env=production", "region=us-west-2"},
			expected: map[string][]string{
				"env":    {"production"},
				"region": {"us-west-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTargetMap(tt.targets)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name     string
		options  []Options
		expected string
	}{
		{
			name:     "no options",
			options:  []Options{},
			expected: "",
		},
		{
			name:     "with prefix",
			options:  []Options{{Prefix: "[test] "}},
			expected: "[test] ",
		},
		{
			name:     "returns first non-empty",
			options:  []Options{{Prefix: ""}, {Prefix: "[second] "}},
			expected: "[second] ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPrefix(tt.options)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionError(t *testing.T) {
	tests := []struct {
		name           string
		err            executionError
		expectedError  string
		expectedUnwrap string
	}{
		{
			name: "with execution ID",
			err: executionError{
				Operation:   "fetch logs",
				ExecutionID: "exec-123",
				Cause:       errors.New("timeout"),
			},
			expectedError:  "fetch logs for execution exec-123: timeout",
			expectedUnwrap: "timeout",
		},
		{
			name: "without execution ID",
			err: executionError{
				Operation: "parse",
				Cause:     errors.New("invalid"),
			},
			expectedError:  "parse: invalid",
			expectedUnwrap: "invalid",
		},
		{
			name: "complex operation with ID",
			err: executionError{
				Operation:   "watch logs",
				ExecutionID: "workflow-abc-456",
				Cause:       errors.New("connection refused"),
			},
			expectedError:  "watch logs for execution workflow-abc-456: connection refused",
			expectedUnwrap: "connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedError, tt.err.Error())
			assert.Equal(t, tt.expectedUnwrap, tt.err.Unwrap().Error())
		})
	}
}

func TestTrimTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "removes RFC3339 timestamp",
			line:     "2024-01-15T10:30:45.123456789Z Starting application",
			expected: "Starting application",
		},
		{
			name:     "no timestamp to remove",
			line:     "Simple log line without timestamp",
			expected: "Simple log line without timestamp",
		},
		{
			name:     "timestamp without content after",
			line:     "2024-01-15T10:30:45.123456789Z",
			expected: "2024-01-15T10:30:45.123456789Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimTimestamp(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractNextLine(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		expectedLine string
		expectedRest []byte
	}{
		{
			name:         "single line without newline",
			input:        []byte("single line"),
			expectedLine: "single line",
			expectedRest: nil,
		},
		{
			name:         "multiple lines",
			input:        []byte("first\nsecond\nthird"),
			expectedLine: "first",
			expectedRest: []byte("second\nthird"),
		},
		{
			name:         "empty input",
			input:        []byte{},
			expectedLine: "",
			expectedRest: nil,
		},
		{
			name:         "single newline only",
			input:        []byte("\n"),
			expectedLine: "",
			expectedRest: []byte{},
		},
		{
			name:         "line ending with newline",
			input:        []byte("line\n"),
			expectedLine: "line",
			expectedRest: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, rest := extractNextLine(tt.input)
			assert.Equal(t, tt.expectedLine, line)
			if tt.expectedRest == nil {
				assert.Nil(t, rest)
			} else {
				assert.Equal(t, tt.expectedRest, rest)
			}
		})
	}
}

func TestShouldPrintOnlyError(t *testing.T) {
	tests := []struct {
		name     string
		results  map[string]testkube.TestWorkflowStepResult
		logs     []byte
		expected bool
	}{
		{
			name: "single init error with no logs should print only error",
			results: map[string]testkube.TestWorkflowStepResult{
				"": {ErrorMessage: "init failed"},
			},
			logs:     []byte{},
			expected: true,
		},
		{
			name: "single init error with logs should not print only error",
			results: map[string]testkube.TestWorkflowStepResult{
				"": {ErrorMessage: "init failed"},
			},
			logs:     []byte("some log content"),
			expected: false,
		},
		{
			name: "multiple results should not print only error",
			results: map[string]testkube.TestWorkflowStepResult{
				"":      {ErrorMessage: "init failed"},
				"step1": {},
			},
			logs:     []byte{},
			expected: false,
		},
		{
			name: "no error message should not print only error",
			results: map[string]testkube.TestWorkflowStepResult{
				"": {},
			},
			logs:     []byte{},
			expected: false,
		},
		{
			name:     "empty results should not print only error",
			results:  map[string]testkube.TestWorkflowStepResult{},
			logs:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldPrintOnlyError(tt.results, tt.logs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractResults(t *testing.T) {
	t.Run("nil result returns empty map", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{}
		results := extractResults(execution)
		assert.Equal(t, map[string]testkube.TestWorkflowStepResult{}, results)
	})

	t.Run("with initialization and steps", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Result: &testkube.TestWorkflowResult{
				Initialization: &testkube.TestWorkflowStepResult{
					ErrorMessage: "init error",
				},
				Steps: map[string]testkube.TestWorkflowStepResult{
					"step1": {ErrorMessage: "error1"},
					"step2": {ErrorMessage: "error2"},
				},
			},
		}
		results := extractResults(execution)

		assert.Equal(t, "init error", results[""].ErrorMessage)
		assert.Equal(t, "error1", results["step1"].ErrorMessage)
		assert.Equal(t, "error2", results["step2"].ErrorMessage)
		assert.Len(t, results, 3)
	})

	t.Run("with only initialization", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Result: &testkube.TestWorkflowResult{
				Initialization: &testkube.TestWorkflowStepResult{
					ErrorMessage: "init failed",
				},
			},
		}
		results := extractResults(execution)

		assert.Equal(t, "init failed", results[""].ErrorMessage)
		assert.Len(t, results, 1)
	})

	t.Run("with only steps no initialization", func(t *testing.T) {
		execution := testkube.TestWorkflowExecution{
			Result: &testkube.TestWorkflowResult{
				Steps: map[string]testkube.TestWorkflowStepResult{
					"step1": {ErrorMessage: "step error"},
				},
			},
		}
		results := extractResults(execution)

		assert.Equal(t, "step error", results["step1"].ErrorMessage)
		assert.Len(t, results, 1)
	})
}
