// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package expressionstcl

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeBase64JSON(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		target   interface{}
		expected interface{}
		wantErr  bool
		errMsg   string
	}{
		{
			name: "decode simple struct",
			encoded: func() string {
				data, _ := json.Marshal(map[string]string{"key": "value"})
				return base64.StdEncoding.EncodeToString(data)
			}(),
			target: &map[string]string{},
			expected: &map[string]string{
				"key": "value",
			},
		},
		{
			name: "decode service map with expressions",
			encoded: func() string {
				services := map[string]json.RawMessage{
					"db": json.RawMessage(`{"description":"Test {{ matrix.browser }}"}`),
				}
				data, _ := json.Marshal(services)
				return base64.StdEncoding.EncodeToString(data)
			}(),
			target: &map[string]json.RawMessage{},
			expected: &map[string]json.RawMessage{
				"db": json.RawMessage(`{"description":"Test {{ matrix.browser }}"}`),
			},
		},
		{
			name: "decode execute data",
			encoded: func() string {
				type ExecuteData struct {
					Tests       []json.RawMessage `json:"tests,omitempty"`
					Workflows   []json.RawMessage `json:"workflows,omitempty"`
					Async       bool              `json:"async,omitempty"`
					Parallelism int               `json:"parallelism,omitempty"`
				}
				data := ExecuteData{
					Tests:       []json.RawMessage{json.RawMessage(`{"name":"test1"}`)},
					Async:       true,
					Parallelism: 5,
				}
				jsonData, _ := json.Marshal(data)
				return base64.StdEncoding.EncodeToString(jsonData)
			}(),
			target: &struct {
				Tests       []json.RawMessage `json:"tests,omitempty"`
				Workflows   []json.RawMessage `json:"workflows,omitempty"`
				Async       bool              `json:"async,omitempty"`
				Parallelism int               `json:"parallelism,omitempty"`
			}{},
			expected: &struct {
				Tests       []json.RawMessage `json:"tests,omitempty"`
				Workflows   []json.RawMessage `json:"workflows,omitempty"`
				Async       bool              `json:"async,omitempty"`
				Parallelism int               `json:"parallelism,omitempty"`
			}{
				Tests:       []json.RawMessage{json.RawMessage(`{"name":"test1"}`)},
				Async:       true,
				Parallelism: 5,
			},
		},
		{
			name:    "invalid base64",
			encoded: "not-valid-base64!@#",
			target:  &map[string]string{},
			wantErr: true,
			errMsg:  "decoding base64",
		},
		{
			name: "invalid JSON after decode",
			encoded: func() string {
				return base64.StdEncoding.EncodeToString([]byte("not json"))
			}(),
			target:  &map[string]string{},
			wantErr: true,
			errMsg:  "parsing JSON",
		},
		{
			name:    "empty string",
			encoded: "",
			target:  &map[string]string{},
			wantErr: true,
			errMsg:  "parsing JSON", // empty string decodes to empty byte array, which fails JSON parsing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := DecodeBase64JSON(tt.encoded, tt.target)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, tt.target)
		})
	}
}

func TestEncodeBase64JSON(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "encode simple map",
			data: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "encode struct with expressions",
			data: struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}{
				Name:        "test-{{ index }}",
				Description: "Instance {{ index + 1 }} of {{ count }}",
			},
		},
		{
			name: "encode nil",
			data: nil,
		},
		{
			name: "encode empty slice",
			data: []string{},
		},
		{
			name: "encode complex nested structure",
			data: map[string]interface{}{
				"services": map[string]interface{}{
					"db": map[string]interface{}{
						"matrix": map[string]interface{}{
							"version": []string{"13", "14", "15"},
						},
					},
				},
			},
		},
		{
			name: "encode services map for processor",
			data: map[string]json.RawMessage{
				"db":    json.RawMessage(`{"description":"Database {{ matrix.version }}","matrix":{"version":["13","14","15"]}}`),
				"cache": json.RawMessage(`{"description":"Redis cache"}`),
			},
		},
		{
			name: "encode execute data structure",
			data: struct {
				Tests       []json.RawMessage `json:"tests,omitempty"`
				Workflows   []json.RawMessage `json:"workflows,omitempty"`
				Async       bool              `json:"async,omitempty"`
				Parallelism int               `json:"parallelism,omitempty"`
			}{
				Tests:       []json.RawMessage{json.RawMessage(`{"name":"test-{{ index }}","description":"Test {{ index + 1 }} of {{ count }}"}`)},
				Async:       true,
				Parallelism: 5,
			},
		},
		{
			name: "encode parallel spec with matrix",
			data: map[string]interface{}{
				"matrix": map[string]interface{}{
					"browser": []string{"chrome", "firefox"},
					"os":      []string{"linux", "windows"},
				},
				"description": "Test {{ matrix.browser }} on {{ matrix.os }}",
				"count":       4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			encoded, err := EncodeBase64JSON(tt.data)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, encoded)

			// Verify round-trip works
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			require.NoError(t, err)

			var result interface{}
			err = json.Unmarshal(decoded, &result)
			require.NoError(t, err)
		})
	}
}

func TestBase64RoundTrip(t *testing.T) {
	// Test that encode/decode round trip preserves data
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "services map",
			data: map[string]json.RawMessage{
				"db":    json.RawMessage(`{"description":"Database {{ matrix.version }}"}`),
				"cache": json.RawMessage(`{"description":"Redis cache"}`),
			},
		},
		{
			name: "parallel spec",
			data: map[string]interface{}{
				"matrix": map[string]interface{}{
					"os":      []string{"linux", "windows"},
					"browser": []string{"chrome", "firefox"},
				},
				"description": "Test on {{ matrix.os }} with {{ matrix.browser }}",
				"count":       4,
			},
		},
		{
			name: "execute data",
			data: map[string]interface{}{
				"tests": []map[string]interface{}{
					{"name": "test-{{ index }}", "description": "Test {{ index + 1 }} of {{ count }}"},
				},
				"async":       true,
				"parallelism": 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := EncodeBase64JSON(tt.data)
			require.NoError(t, err)

			// Decode into same type
			decoded := reflect.New(reflect.TypeOf(tt.data)).Interface()
			err = DecodeBase64JSON(encoded, decoded)
			require.NoError(t, err)

			// Compare - need to dereference the pointer
			decodedValue := reflect.ValueOf(decoded).Elem().Interface()

			// For maps with interface{} values, we need special comparison
			// because JSON unmarshaling converts []string to []interface{}
			expectedJSON, _ := json.Marshal(tt.data)
			actualJSON, _ := json.Marshal(decodedValue)
			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestEncodeDecodeCompatibility(t *testing.T) {
	// Test that our encode function produces output compatible with toolkit decode
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "processor encodes services for toolkit",
			data: map[string]json.RawMessage{
				"db": json.RawMessage(`{
					"description": "PostgreSQL {{ matrix.version }}",
					"matrix": {"version": ["13", "14", "15"]},
					"image": "postgres:{{ matrix.version }}-alpine"
				}`),
			},
		},
		{
			name: "processor encodes execute data for toolkit",
			data: map[string]interface{}{
				"tests": []map[string]interface{}{
					{
						"name":        "test-{{ index }}",
						"description": "Instance {{ index + 1 }} of {{ count }}",
					},
				},
				"async":       false,
				"parallelism": 10,
			},
		},
		{
			name: "processor encodes parallel spec for toolkit",
			data: map[string]interface{}{
				"matrix": map[string]interface{}{
					"browser": []string{"chrome", "firefox", "safari"},
					"os":      []string{"ubuntu", "macos"},
				},
				"shards": map[string]interface{}{
					"count": 5,
				},
				"description": "Browser {{ matrix.browser }} on {{ matrix.os }} - shard {{ shard.index }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode as processor would
			encoded, err := EncodeBase64JSON(tt.data)
			require.NoError(t, err)

			// Decode as toolkit would
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			require.NoError(t, err)

			var result interface{}
			err = json.Unmarshal(decoded, &result)
			require.NoError(t, err)

			// Verify data integrity
			expectedJSON, _ := json.Marshal(tt.data)
			actualJSON, _ := json.Marshal(result)
			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}
