// Package jsonpath provides a JSONPath query interface for extracting data from JSON/YAML structures.
// It wraps the underlying JSONPath implementation to provide consistent error handling,
// null-safe results, and safety limits (timeouts, output size).
package jsonpath

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	// DefaultTimeout is the default query timeout.
	DefaultTimeout = 10 * time.Second

	// DefaultMaxOutputSize is the default maximum output size in bytes.
	DefaultMaxOutputSize = 100 * 1024 // 100KB

	// DefaultMaxInputSize is the default maximum input size in bytes.
	DefaultMaxInputSize = 10 * 1024 * 1024 // 10MB
)

// Engine defines the interface for JSONPath query engines.
// This abstraction allows swapping the underlying implementation.
type Engine interface {
	// Query executes a JSONPath expression against the provided data.
	// Returns the matching values as a slice.
	// Missing paths return an empty slice, not an error.
	// Invalid path syntax returns an error.
	Query(path string, data any) ([]any, error)
}

// Options configures the query behavior.
type Options struct {
	// Timeout is the maximum time allowed for query execution.
	Timeout time.Duration

	// MaxOutputSize is the maximum allowed output size in bytes.
	MaxOutputSize int

	// MaxInputSize is the maximum allowed input size in bytes.
	MaxInputSize int
}

// DefaultOptions returns the default query options.
func DefaultOptions() Options {
	return Options{
		Timeout:       DefaultTimeout,
		MaxOutputSize: DefaultMaxOutputSize,
		MaxInputSize:  DefaultMaxInputSize,
	}
}

// QueryError wraps errors with additional context.
type QueryError struct {
	Path    string
	Message string
	Err     error
}

func (e *QueryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("jsonpath query %q: %s: %v", e.Path, e.Message, e.Err)
	}
	return fmt.Sprintf("jsonpath query %q: %s", e.Path, e.Message)
}

func (e *QueryError) Unwrap() error {
	return e.Err
}

// Query executes a JSONPath expression against the provided data using default options.
// This is a convenience function that uses the default ojg engine.
//
// The path should be a valid JSONPath expression:
//   - $              - Root element
//   - $.property     - Child property
//   - $['property']  - Bracket notation
//   - $[n]           - Array index
//   - $[*]           - All array elements
//   - $..property    - Recursive descent (find all matching keys)
//   - $[?(@.key == 'val')] - Filter by equality
//   - $[?(@.key != 'val')] - Filter by inequality
//
// Returns:
//   - []any: Matching values (empty slice if path doesn't match anything)
//   - error: Only on invalid path syntax or timeouts
func Query(path string, data any) ([]any, error) {
	return QueryWithContext(context.Background(), path, data, DefaultOptions())
}

// QueryWithContext executes a JSONPath expression with context and custom options.
func QueryWithContext(ctx context.Context, path string, data any, opts Options) ([]any, error) {
	// Apply timeout from options if context doesn't have a deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Check input size limit to prevent memory issues with very large data
	if opts.MaxInputSize > 0 {
		inputSize := estimateSize(data)
		if inputSize > opts.MaxInputSize {
			return nil, &QueryError{
				Path:    path,
				Message: fmt.Sprintf("input data exceeds maximum size (%d > %d bytes)", inputSize, opts.MaxInputSize),
			}
		}
	}

	// Use the default engine
	engine := defaultEngine()

	// Execute query with panic recovery
	resultCh := make(chan queryResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- queryResult{
					err: &QueryError{
						Path:    path,
						Message: "query panicked",
						Err:     fmt.Errorf("%v", r),
					},
				}
			}
		}()

		result, err := engine.Query(path, data)
		resultCh <- queryResult{result: result, err: err}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		return nil, &QueryError{
			Path:    path,
			Message: "query timed out",
			Err:     ctx.Err(),
		}
	case res := <-resultCh:
		return res.result, res.err
	}
}

type queryResult struct {
	result []any
	err    error
}

// estimateSize returns an approximate size of the data in bytes.
// Uses JSON marshaling as a reasonable estimate for memory usage.
func estimateSize(data any) int {
	if data == nil {
		return 0
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		// If marshaling fails, return 0 to skip size check
		// (the query will likely fail later with a more specific error)
		return 0
	}
	return len(bytes)
}
