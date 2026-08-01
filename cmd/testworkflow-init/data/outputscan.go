package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	outputsDir = "/testkube/outputs"
)

const (
	MaxOutputSize = 4096
)

func GetOutputsDir() string {
	return outputsDir
}

func SetOutputsDir(dir string) {
	outputsDir = dir
}

// ScanStepOutputs reads files from OutputsDir and stores their contents
// as per-step outputs. Files exceeding MaxOutputSize are skipped.
// It returns the scanned values, so the caller may publish them further
// (e.g. as workflow-level outputs readable by a parent workflow).
func ScanStepOutputs(stepId string) (map[string]string, error) {
	return scanStepOutputsFrom(outputsDir, stepId)
}

func scanStepOutputsFrom(dir, stepId string) (map[string]string, error) {
	if stepId == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read outputs directory: %w", err)
	}

	values := make(map[string]string)
	state := GetState()
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !entry.Type().IsRegular() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.Size() > MaxOutputSize {
			fmt.Fprintf(os.Stderr, "warn: step output %q exceeds %d byte limit, skipping (use step.results for large files)\n", name, MaxOutputSize)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: failed to read step output %q: %s\n", name, err.Error())
			continue
		}

		value := strings.TrimSpace(string(content))
		state.SetStepOutput(stepId, name, value)
		values[name] = value
	}
	return values, nil
}

func PrepareOutputsDir() error {
	if err := os.RemoveAll(outputsDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear outputs directory: %w", err)
	}
	if err := os.MkdirAll(outputsDir, 0777); err != nil {
		return fmt.Errorf("failed to create outputs directory: %w", err)
	}
	EnsureGroupWritable(outputsDir)
	return nil
}

// EnsureGroupWritable adds the group write bit to a directory while preserving
// all existing permission bits (including setgid from FSGroup).
// This is needed because MkdirAll respects umask (typically 0022), creating
// directories as 0755. In multi-container pods sharing an FSGroup, other
// containers with different UIDs need the group write bit to create entries.
func EnsureGroupWritable(path string) {
	if info, err := os.Stat(path); err == nil {
		_ = os.Chmod(path, info.Mode()|0020)
	}
}
