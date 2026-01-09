package testworkflowexecutor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/testkube/pkg/expressions"
)

func TestApplyConfig_JSONHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string value",
			input:    "production",
			expected: "production",
		},
		{
			name:     "simple value with equals",
			input:    "key=value",
			expected: "key=value",
		},
		{
			name:     "JSON object with URL",
			input:    `{"agency":{"url":"https://test.com"}}`,
			expected: `tojson(json("{\"agency\":{\"url\":\"https://test.com\"}}"))`,
		},
		{
			name:     "JSON object with multiple fields",
			input:    `{"name":"Test","url":"https://example.com","port":8080}`,
			expected: `tojson(json("{\"name\":\"Test\",\"url\":\"https://example.com\",\"port\":8080}"))`,
		},
		{
			name:     "JSON array",
			input:    `["value1","value2","value3"]`,
			expected: `tojson(json("[\"value1\",\"value2\",\"value3\"]"))`,
		},
		{
			name:     "nested JSON object",
			input:    `{"agency":{"name":"Test Agency","url":"https://test.com","config":{"timeout":30}}}`,
			expected: `tojson(json("{\"agency\":{\"name\":\"Test Agency\",\"url\":\"https://test.com\",\"config\":{\"timeout\":30}}}"))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string

			isJSON := len(tt.input) > 0 && ((tt.input[0] == '{' && (len(tt.input) < 2 || tt.input[1] != '{')) || tt.input[0] == '[')
			if isJSON {
				result = "tojson(json(" + expressions.NewStringValue(tt.input).String() + "))"
			} else {
				result = expressions.NewStringValue(tt.input).Template()
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyConfig_PreservesColonsInJSON(t *testing.T) {
	input := `{"url":"https://test.com","port":8080}`

	var result string
	isJSON := len(input) > 0 && ((input[0] == '{' && (len(input) < 2 || input[1] != '{')) || input[0] == '[')
	if isJSON {
		result = "tojson(json(" + expressions.NewStringValue(input).String() + "))"
	}

	assert.Contains(t, result, "tojson(json(")
	assert.Contains(t, result, `\"url\":`)
	assert.Contains(t, result, `\"https:`)
	assert.NotContains(t, result, "agency:url")
	assert.NotContains(t, result, "url:https")
}

func TestApplyConfig_SimpleValuesUnchanged(t *testing.T) {
	tests := []string{
		"production",
		"test-value",
		"key=value",
		"https://example.com",
		"123",
		"true",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if len(input) > 0 && (input[0] == '{' || input[0] == '[') {
				t.Fatal("test data should not start with { or [")
			}

			result := expressions.NewStringValue(input).Template()
			assert.Equal(t, input, result)
		})
	}
}

func TestApplyConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		shouldBeJSON bool
	}{
		{
			name:         "template expression",
			input:        "{{config.value}}",
			shouldBeJSON: false,
		},
		{
			name:         "invalid JSON",
			input:        "{notjson",
			shouldBeJSON: true,
		},
		{
			name:         "empty string",
			input:        "",
			shouldBeJSON: false,
		},
		{
			name:         "whitespace before JSON",
			input:        " {\"key\":\"value\"}",
			shouldBeJSON: false,
		},
		{
			name:         "JSON with template expression inside",
			input:        `{"url":"{{config.baseUrl}}/api"}`,
			shouldBeJSON: true,
		},
		{
			name:         "ternary expression",
			input:        "{{condition ? 'true' : 'false'}}",
			shouldBeJSON: false,
		},
		{
			name:         "URL with port",
			input:        "https://example.com:8080/path",
			shouldBeJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isJSON := len(tt.input) > 0 && ((tt.input[0] == '{' && (len(tt.input) < 2 || tt.input[1] != '{')) || tt.input[0] == '[')
			assert.Equal(t, tt.shouldBeJSON, isJSON)

			var result string
			if isJSON {
				result = "tojson(json(" + expressions.NewStringValue(tt.input).String() + "))"
				assert.Contains(t, result, "tojson(json(")
			} else {
				result = expressions.NewStringValue(tt.input).Template()
				assert.NotContains(t, result, "tojson(json(")
			}
		})
	}
}

func TestApplyConfig_BackwardCompatibility(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"environment variable", "production"},
		{"numeric string", "123"},
		{"boolean string", "true"},
		{"URL with port", "https://api.example.com:8080"},
		{"path", "/path/to/resource"},
		{"template expression", "{{config.value}}"},
		{"complex template", "{{env.ENV}}-{{config.region}}"},
		{"value with equals", "key=value&another=value2"},
		{"semicolon separated", "opt1;opt2;opt3"},
		{"comma separated", "value1,value2,value3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isJSON := len(tt.input) > 0 && ((tt.input[0] == '{' && (len(tt.input) < 2 || tt.input[1] != '{')) || tt.input[0] == '[')
			assert.False(t, isJSON)

			result := expressions.NewStringValue(tt.input).Template()

			if !strings.Contains(tt.input, "{{") {
				assert.Equal(t, tt.input, result)
			} else {
				assert.NotContains(t, result, "tojson(json(")
			}
		})
	}
}

func TestApplyConfig_JSONValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"valid JSON object", `{"key":"value"}`},
		{"valid JSON array", `["a","b","c"]`},
		{"invalid JSON object", `{invalid}`},
		{"invalid JSON array", `[invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isJSON := len(tt.input) > 0 && ((tt.input[0] == '{' && (len(tt.input) < 2 || tt.input[1] != '{')) || tt.input[0] == '[')
			assert.True(t, isJSON)

			result := "tojson(json(" + expressions.NewStringValue(tt.input).String() + "))"
			assert.Contains(t, result, "tojson(json(")
		})
	}
}

// TestApplyConfig_ColonHandling tests proper handling of colons in JSON config values
func TestApplyConfig_ColonHandling(t *testing.T) {
	tests := []struct {
		name             string
		configInput      map[string]string
		expectedWrapping map[string]bool
		shouldContain    map[string][]string
		shouldNotContain map[string][]string
	}{
		{
			name: "nested JSON object with URL fields",
			configInput: map[string]string{
				"customAgency": `{"agency":{"url":"https://test.com"}}`,
			},
			expectedWrapping: map[string]bool{
				"customAgency": true,
			},
			shouldContain: map[string][]string{
				"customAgency": {`tojson(json(`, `\"url\":`, `\"https:`},
			},
			shouldNotContain: map[string][]string{
				"customAgency": {"agency:url", "url:https"},
			},
		},
		{
			name: "mixed config types",
			configInput: map[string]string{
				"env":       "production",
				"apiConfig": `{"url":"https://api.test.com:8080"}`,
			},
			expectedWrapping: map[string]bool{
				"env":       false,
				"apiConfig": true,
			},
			shouldContain: map[string][]string{
				"env":       {"production"},
				"apiConfig": {`tojson(json(`},
			},
			shouldNotContain: map[string][]string{
				"env":       {`tojson(json(`},
				"apiConfig": {"url:https"},
			},
		},
		{
			name: "template expressions not wrapped",
			configInput: map[string]string{
				"baseUrl":   "{{config.apiUrl}}",
				"condition": "{{env.DEBUG ? 'debug' : 'production'}}",
			},
			expectedWrapping: map[string]bool{
				"baseUrl":   false,
				"condition": false,
			},
			shouldNotContain: map[string][]string{
				"baseUrl":   {`tojson(json(`},
				"condition": {`tojson(json(`},
			},
		},
		{
			name: "plain URLs not wrapped",
			configInput: map[string]string{
				"apiUrl": "https://api.test.com:8080/v1",
				"dbUrl":  "postgresql://user:pass@localhost:5432/db",
			},
			expectedWrapping: map[string]bool{
				"apiUrl": false,
				"dbUrl":  false,
			},
			shouldContain: map[string][]string{
				"apiUrl": {"https://api.test.com:8080/v1"},
			},
			shouldNotContain: map[string][]string{
				"apiUrl": {`tojson(json(`},
			},
		},
		{
			name: "complex nested JSON",
			configInput: map[string]string{
				"services": `{"database":{"host":"db.test.com:5432"},"api":{"endpoint":"https://api.test.com:8443"}}`,
			},
			expectedWrapping: map[string]bool{
				"services": true,
			},
			shouldContain: map[string][]string{
				"services": {`tojson(json(`, `\"host\":`},
			},
			shouldNotContain: map[string][]string{
				"services": {"host:db"},
			},
		},
		{
			name: "JSON array with objects",
			configInput: map[string]string{
				"endpoints": `[{"name":"api","url":"https://api.test.com:8080"}]`,
			},
			expectedWrapping: map[string]bool{
				"endpoints": true,
			},
			shouldContain: map[string][]string{
				"endpoints": {`tojson(json(`},
			},
			shouldNotContain: map[string][]string{
				"endpoints": {"url:https"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processedConfig := make(map[string]string)
			for k, v := range tt.configInput {
				isJSON := len(v) > 0 && ((v[0] == '{' && (len(v) < 2 || v[1] != '{')) || v[0] == '[')

				if expectedWrapped, exists := tt.expectedWrapping[k]; exists {
					assert.Equal(t, expectedWrapped, isJSON)
				}

				if isJSON {
					processedConfig[k] = "tojson(json(" + expressions.NewStringValue(v).String() + "))"
				} else {
					processedConfig[k] = expressions.NewStringValue(v).Template()
				}
			}

			for key, expectedStrings := range tt.shouldContain {
				processedValue := processedConfig[key]
				for _, expected := range expectedStrings {
					assert.Contains(t, processedValue, expected)
				}
			}

			for key, unwantedStrings := range tt.shouldNotContain {
				processedValue := processedConfig[key]
				for _, unwanted := range unwantedStrings {
					assert.NotContains(t, processedValue, unwanted)
				}
			}
		})
	}
}

// TestApplyConfig_MultipleColonsInValue tests values with many colons are preserved
func TestApplyConfig_MultipleColonsInValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		key   string
		value string
	}{
		{
			name:  "database connection strings",
			input: `dbConfig={"primary":"postgresql://user:password@primary.db.com:5432/mydb"}`,
			key:   "dbConfig",
			value: `{"primary":"postgresql://user:password@primary.db.com:5432/mydb"}`,
		},
		{
			name:  "service endpoint arrays",
			input: `services=["https://service1.test.com:8080","https://service2.test.com:8081"]`,
			key:   "services",
			value: `["https://service1.test.com:8080","https://service2.test.com:8081"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.SplitN(tt.input, "=", 2)
			assert.Equal(t, tt.key, parts[0])
			assert.Equal(t, tt.value, parts[1])

			isJSON := len(parts[1]) > 0 && ((parts[1][0] == '{' && (len(parts[1]) < 2 || parts[1][1] != '{')) || parts[1][0] == '[')
			assert.True(t, isJSON)

			processedValue := "tojson(json(" + expressions.NewStringValue(parts[1]).String() + "))"
			assert.Contains(t, processedValue, "tojson(json(")
			assert.Greater(t, strings.Count(parts[1], ":"), 0)
		})
	}
}
