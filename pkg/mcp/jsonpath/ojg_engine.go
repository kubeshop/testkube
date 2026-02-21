package jsonpath

import (
	"github.com/ohler55/ojg/jp"
	"github.com/ohler55/ojg/oj"
)

// ojgEngine implements the Engine interface using ohler55/ojg.
type ojgEngine struct{}

// defaultEngine returns the default JSONPath engine (ojg).
func defaultEngine() Engine {
	return &ojgEngine{}
}

// Query executes a JSONPath expression against the provided data.
// It handles null-safe results by returning empty slices for missing paths.
func (e *ojgEngine) Query(path string, data any) ([]any, error) {
	// Parse the JSONPath expression
	expr, err := jp.ParseString(path)
	if err != nil {
		return nil, &QueryError{
			Path:    path,
			Message: "invalid JSONPath syntax",
			Err:     err,
		}
	}

	// Normalize data to ensure consistent handling
	// ojg works best with its native types, so we convert through JSON if needed
	normalized, err := normalizeData(data)
	if err != nil {
		return nil, &QueryError{
			Path:    path,
			Message: "failed to normalize input data",
			Err:     err,
		}
	}

	// Execute the query
	results := expr.Get(normalized)

	// Return empty slice for no matches (null-safe)
	if results == nil {
		return []any{}, nil
	}

	// Flatten single results into slice
	return results, nil
}

// normalizeData converts the input data to a format that ojg can query consistently.
// This handles cases where data might be a struct, map with non-string keys, etc.
func normalizeData(data any) (any, error) {
	// If data is already nil, return empty map
	if data == nil {
		return map[string]any{}, nil
	}

	// Convert through JSON to normalize the data structure
	// This ensures consistent handling of structs, typed maps, etc.
	jsonBytes, err := oj.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Parse back to generic types
	result, err := oj.Parse(jsonBytes)
	if err != nil {
		return nil, err
	}

	return result, nil
}
