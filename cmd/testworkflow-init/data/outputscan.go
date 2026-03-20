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
func ScanStepOutputs(stepId string) error {
	return scanStepOutputsFrom(outputsDir, stepId)
}

func scanStepOutputsFrom(dir, stepId string) error {
	if stepId == "" {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read outputs directory: %w", err)
	}

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

		state.SetStepOutput(stepId, name, strings.TrimSpace(string(content)))
	}
	return nil
}

func PrepareOutputsDir() error {
	if err := os.RemoveAll(outputsDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear outputs directory: %w", err)
	}
	if err := os.MkdirAll(outputsDir, 0777); err != nil {
		return fmt.Errorf("failed to create outputs directory: %w", err)
	}
	return nil
}
