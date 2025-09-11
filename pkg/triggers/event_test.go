package triggers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDataFromFile(t *testing.T) {
	// Test with empty file path
	t.Run("empty file path", func(t *testing.T) {
		data, err := loadDataFromFile("")
		if err != nil {
			t.Errorf("expected no error for empty path, got: %v", err)
		}
		if data != nil {
			t.Errorf("expected nil data for empty path, got: %v", data)
		}
	})

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		data, err := loadDataFromFile("/non/existent/file.txt")
		if err != nil {
			t.Errorf("expected no error for non-existent file, got: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("expected empty map for non-existent file, got: %v", data)
		}
	})

	// Test with valid file
	t.Run("valid file", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test-data.txt")

		content := `# This is a comment
environment=production
region=us-west-2

# Another comment
app_version=1.2.3
custom_key=custom_value
`
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		data, err := loadDataFromFile(tmpFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := map[string]string{
			"environment": "production",
			"region":      "us-west-2",
			"app_version": "1.2.3",
			"custom_key":  "custom_value",
		}

		if len(data) != len(expected) {
			t.Errorf("expected %d keys, got %d", len(expected), len(data))
		}

		for key, expectedValue := range expected {
			if actualValue, exists := data[key]; !exists {
				t.Errorf("expected key %s not found", key)
			} else if actualValue != expectedValue {
				t.Errorf("for key %s: expected %s, got %s", key, expectedValue, actualValue)
			}
		}
	})

	// Test with invalid format
	t.Run("invalid format", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "invalid-data.txt")

		content := `valid_key=valid_value
invalid_line_without_equals
another_key=another_value
`
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := loadDataFromFile(tmpFile)
		if err == nil {
			t.Error("expected error for invalid format, got nil")
		}
	})

	// Test with empty key
	t.Run("empty key", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "empty-key.txt")

		content := `valid_key=valid_value
=empty_key_value
`
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := loadDataFromFile(tmpFile)
		if err == nil {
			t.Error("expected error for empty key, got nil")
		}
	})

	// Test with whitespace handling
	t.Run("whitespace handling", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "whitespace-data.txt")

		content := `  key1  =  value1  
	key2	=	value2	
key3=value3
`
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		data, err := loadDataFromFile(tmpFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expected := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		for key, expectedValue := range expected {
			if actualValue, exists := data[key]; !exists {
				t.Errorf("expected key %s not found", key)
			} else if actualValue != expectedValue {
				t.Errorf("for key %s: expected %s, got %s", key, expectedValue, actualValue)
			}
		}
	})
}
