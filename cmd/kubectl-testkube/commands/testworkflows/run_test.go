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
func TestParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "simple string value",
			input:    []string{"env=production"},
			expected: map[string]string{"env": "production"},
			wantErr:  false,
		},
		{
			name:     "simple integer value",
			input:    []string{"workers=2"},
			expected: map[string]string{"workers": "2"},
			wantErr:  false,
		},
		{
			name:     "JSON object value",
			input:    []string{`customAgency={"agency":{"url":"https://test.com"}}`},
			expected: map[string]string{"customAgency": `{"agency":{"url":"https://test.com"}}`},
			wantErr:  false,
		},
		{
			name:     "JSON object with escaped quotes",
			input:    []string{`data={\"key\":\"value\"}`},
			expected: map[string]string{"data": `{\"key\":\"value\"}`},
			wantErr:  false,
		},
		{
			name:     "JSON array value",
			input:    []string{`tags=["tag1","tag2","tag3"]`},
			expected: map[string]string{"tags": `["tag1","tag2","tag3"]`},
			wantErr:  false,
		},
		{
			name:     "mixed simple and JSON",
			input:    []string{"env=prod", `config={"key":"value"}`},
			expected: map[string]string{"env": "prod", "config": `{"key":"value"}`},
			wantErr:  false,
		},
		{
			name:     "value with colon (URL)",
			input:    []string{"url=http://example.com:8080"},
			expected: map[string]string{"url": "http://example.com:8080"},
			wantErr:  false,
		},
		{
			name:     "value with equals sign",
			input:    []string{"query=param1=value1&param2=value2"},
			expected: map[string]string{"query": "param1=value1&param2=value2"},
			wantErr:  false,
		},
		{
			name:     "complex JSON with nested objects and arrays",
			input:    []string{`settings={"server":{"host":"localhost","port":8080},"features":["auth","logging"]}`},
			expected: map[string]string{"settings": `{"server":{"host":"localhost","port":8080},"features":["auth","logging"]}`},
			wantErr:  false,
		},
		{
			name:     "empty value",
			input:    []string{"key="},
			expected: map[string]string{"key": ""},
			wantErr:  false,
		},
		{
			name:     "multiple configs",
			input:    []string{"env=dev", "region=us-west", "workers=5"},
			expected: map[string]string{"env": "dev", "region": "us-west", "workers": "5"},
			wantErr:  false,
		},
		{
			name:    "invalid format no equals",
			input:   []string{"invalid"},
			wantErr: true,
			errMsg:  "invalid config format",
		},
		{
			name:    "empty key",
			input:   []string{"=value"},
			wantErr: true,
			errMsg:  "empty config key",
		},
		{
			name:    "only equals sign",
			input:   []string{"="},
			wantErr: true,
			errMsg:  "empty config key",
		},
		{
			name:     "value that looks like invalid JSON (should still work as string)",
			input:    []string{`key={invalid`},
			expected: map[string]string{"key": "{invalid"},
			wantErr:  false,
		},
		{
			name:     "JSON with spaces",
			input:    []string{`data={"key": "value with spaces"}`},
			expected: map[string]string{"data": `{"key": "value with spaces"}`},
			wantErr:  false,
		},
		{
			name:     "backward compatibility - simple key value pairs",
			input:    []string{"version=1.0.0", "timeout=30s", "retries=3"},
			expected: map[string]string{"version": "1.0.0", "timeout": "30s", "retries": "3"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseConfig(tt.input)

			if tt.wantErr {
				assert.NotNil(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestParseConfig_Integration tests the full flow from CLI parsing to backend processing
func TestParseConfig_Integration(t *testing.T) {
	tests := []struct {
		name           string
		cliInput       []string
		expectedConfig map[string]string
		description    string
	}{
		{
			name:           "bug report scenario - JSON object with URL containing colons",
			cliInput:       []string{`customAgency={\"agency\":{\"url\":\"https://test.com\"}}`},
			expectedConfig: map[string]string{"customAgency": `{\"agency\":{\"url\":\"https://test.com\"}}`},
			description:    "Verifies the exact scenario from the bug report where JSON values with colons were being split incorrectly",
		},
		{
			name:           "mixed config types",
			cliInput:       []string{"env=production", `api={"url":"https://api.example.com:8080"}`, "workers=5"},
			expectedConfig: map[string]string{"env": "production", "api": `{"url":"https://api.example.com:8080"}`, "workers": "5"},
			description:    "Verifies that simple values and JSON values can coexist without issues",
		},
		{
			name:           "complex nested JSON with multiple colons",
			cliInput:       []string{`config={"database":{"host":"db.test.com:5432","url":"postgresql://user:pass@db.test.com:5432/dbname"},"api":{"endpoint":"https://api.test.com:8443/v1"}}`},
			expectedConfig: map[string]string{"config": `{"database":{"host":"db.test.com:5432","url":"postgresql://user:pass@db.test.com:5432/dbname"},"api":{"endpoint":"https://api.test.com:8443/v1"}}`},
			description:    "Verifies complex JSON structures with multiple nested objects and colons are preserved",
		},
		{
			name:           "JSON array with objects containing URLs",
			cliInput:       []string{`services=[{"name":"api","url":"https://api.test.com:8080"},{"name":"auth","url":"https://auth.test.com:443"}]`},
			expectedConfig: map[string]string{"services": `[{"name":"api","url":"https://api.test.com:8080"},{"name":"auth","url":"https://auth.test.com:443"}]`},
			description:    "Verifies JSON arrays containing objects with URLs are handled correctly",
		},
		{
			name:           "URL without JSON wrapper",
			cliInput:       []string{"apiUrl=https://test.com:8080/api/v1"},
			expectedConfig: map[string]string{"apiUrl": "https://test.com:8080/api/v1"},
			description:    "Verifies plain URL values with colons still work (backward compatibility)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Parse CLI input (simulates what happens when user runs the command)
			parsedConfig, err := parseConfig(tt.cliInput)
			assert.Nil(t, err, "CLI parsing should not fail")
			assert.Equal(t, tt.expectedConfig, parsedConfig, "CLI parsing should produce expected config map")

			// Step 2: Verify no colons were incorrectly split
			for key, value := range parsedConfig {
				// If the original input had JSON with colons, verify they're still there
				for _, input := range tt.cliInput {
					if len(input) > len(key) && input[:len(key)] == key && input[len(key)] == '=' {
						expectedValue := input[len(key)+1:]
						assert.Equal(t, expectedValue, value, "Value should match original input after first equals sign")

						// Specifically check that colons in URLs are not split
						if value[0] == '{' || value[0] == '[' {
							// For JSON values, ensure colons are preserved
							assert.Contains(t, value, ":", "JSON values should contain colons")
							// Should NOT be split into separate parts like "agency:url:https"
							assert.NotContains(t, value, "agency:url", "Should not have incorrectly split JSON structure")
						}
					}
				}
			}
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
			result, err := parseTargetMap(tt.targets)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTargetMapErrors(t *testing.T) {
	tests := []struct {
		name    string
		targets []string
		errMsg  string
	}{
		{
			name:    "empty key",
			targets: []string{"=value"},
			errMsg:  "empty target key",
		},
		{
			name:    "empty string",
			targets: []string{""},
			errMsg:  "empty target key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTargetMap(tt.targets)
			assert.Nil(t, result)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
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
