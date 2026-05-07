package formatters

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

// IsEmptyInput checks if the raw input is empty, whitespace-only, or "null".
func IsEmptyInput(raw string) bool {
	if raw == "" {
		return true
	}
	trimmed := strings.TrimSpace(raw)
	return trimmed == "" || trimmed == "null"
}

// ParseJSON parses raw input (JSON or YAML) into the target type.
// Returns an empty value if the input is empty or null.
func ParseJSON[T any](raw string) (T, bool, error) {
	var result T
	if IsEmptyInput(raw) {
		return result, true, nil
	}

	trimmed := strings.TrimSpace(raw)

	// Detect input format: JSON starts with [ or {
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return result, false, err
		}
	} else {
		if err := yaml.Unmarshal([]byte(raw), &result); err != nil {
			return result, false, err
		}
	}

	return result, false, nil
}

// FormatJSON marshals data to a compact JSON string.
func FormatJSON(data any) (string, error) {
	out, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
