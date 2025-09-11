package triggers

import (
	"os"
	"testing"
)

func TestLoadDataFromFile(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		filePath     string
		expected     map[string]string
		shouldCreate bool
	}{
		{
			name:         "empty file path",
			filePath:     "",
			expected:     map[string]string{},
			shouldCreate: false,
		},
		{
			name:         "non-existent file",
			filePath:     "/tmp/non-existent-file.conf",
			expected:     map[string]string{},
			shouldCreate: false,
		},
		{
			name:        "basic key-value pairs with equals",
			filePath:    "/tmp/test-agent-data-equals.conf",
			fileContent: "key1=value1\nkey2=value2\nkey3=value3",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			shouldCreate: true,
		},
		{
			name:        "basic key-value pairs with colons",
			filePath:    "/tmp/test-agent-data-colons.conf",
			fileContent: "key1: value1\nkey2: value2\nkey3: value3",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			shouldCreate: true,
		},
		{
			name:        "mixed format with comments and empty lines",
			filePath:    "/tmp/test-agent-data-mixed.conf",
			fileContent: "# This is a comment\nkey1=value1\n\n# Another comment\nkey2: value2\n\nkey3=value3\n",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			shouldCreate: true,
		},
		{
			name:        "values with spaces and special characters",
			filePath:    "/tmp/test-agent-data-spaces.conf",
			fileContent: "description=This is a test environment\nemail: test@example.com\npath=/var/lib/testkube",
			expected: map[string]string{
				"description": "This is a test environment",
				"email":       "test@example.com",
				"path":        "/var/lib/testkube",
			},
			shouldCreate: true,
		},
		{
			name:        "keys with whitespace around separators",
			filePath:    "/tmp/test-agent-data-whitespace.conf",
			fileContent: "key1 = value1\n key2 : value2 \n  key3=value3  ",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			shouldCreate: true,
		},
		{
			name:        "empty values",
			filePath:    "/tmp/test-agent-data-empty.conf",
			fileContent: "key1=\nkey2:\nkey3=value3",
			expected: map[string]string{
				"key1": "",
				"key2": "",
				"key3": "value3",
			},
			shouldCreate: true,
		},
		{
			name:        "invalid lines ignored",
			filePath:    "/tmp/test-agent-data-invalid.conf",
			fileContent: "key1=value1\ninvalid line without separator\nkey2=value2\n=invalid key\n:invalid key2\nkey3=value3",
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			shouldCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if needed
			if tt.shouldCreate {
				err := os.WriteFile(tt.filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(tt.filePath) // Clean up
			}

			// Test the function
			result := loadDataFromFile(tt.filePath)

			// Verify the result
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key '%s', expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}

			// Check for unexpected keys
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("Unexpected key '%s' found with value '%s'", key, result[key])
				}
			}
		})
	}
}

func TestWithAgentDataFilePath(t *testing.T) {
	// Create a temporary test file
	testFile := "/tmp/test-agent-data-option.conf"
	testContent := "environment=test\nregion=local\nteam=developers"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Create a service and apply the option
	service := &Service{
		Agent: watcherAgent{},
	}

	// Apply the option
	option := WithAgentDataFilePath(testFile)
	option(service)

	// Verify the data was loaded
	expected := map[string]string{
		"environment": "test",
		"region":      "local",
		"team":        "developers",
	}

	if len(service.Agent.Data) != len(expected) {
		t.Errorf("Expected %d items in agent data, got %d", len(expected), len(service.Agent.Data))
	}

	for key, expectedValue := range expected {
		if actualValue, exists := service.Agent.Data[key]; !exists {
			t.Errorf("Expected key '%s' not found in agent data", key)
		} else if actualValue != expectedValue {
			t.Errorf("For key '%s', expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

func TestWithAgentDataFilePathEmptyPath(t *testing.T) {
	// Create a service and apply the option with empty path
	service := &Service{
		Agent: watcherAgent{},
	}

	// Apply the option with empty path
	option := WithAgentDataFilePath("")
	option(service)

	// Verify no data was loaded (should be nil or empty)
	if service.Agent.Data != nil && len(service.Agent.Data) > 0 {
		t.Errorf("Expected no agent data for empty path, got %d items", len(service.Agent.Data))
	}
}
