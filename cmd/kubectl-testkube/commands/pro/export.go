package pro

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	exportPageSize = 100
)

// SequenceEntry holds the current sequence number for a workflow.
type SequenceEntry struct {
	WorkflowName string `json:"workflowName"`
	Number       int32  `json:"number"`
}

// ExportData exports all execution metadata and logs from the running standalone
// agent into a .tar.gz archive. It must be called while DB and MinIO are still up.
// Returns the path to the created archive.
func ExportData(apiClient client.Client, outputDir string) (string, error) {
	// Create a temp directory for staging files
	stagingDir, err := os.MkdirTemp("", "testkube-export-*")
	if err != nil {
		return "", fmt.Errorf("creating staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	execDir := filepath.Join(stagingDir, "executions")
	logsDir := filepath.Join(stagingDir, "logs")
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return "", fmt.Errorf("creating executions directory: %w", err)
	}
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return "", fmt.Errorf("creating logs directory: %w", err)
	}

	// Track max sequence number per workflow
	sequences := map[string]int32{}

	// Paginate through all executions
	page := 0
	totalExported := 0
	for {
		opts := client.FilterTestWorkflowExecutionOptions{Page: page}
		result, err := apiClient.ListTestWorkflowExecutions("", exportPageSize, opts)
		if err != nil {
			return "", fmt.Errorf("listing executions (page %d): %w", page, err)
		}

		if len(result.Results) == 0 {
			break
		}

		for _, summary := range result.Results {
			// Get full execution details
			execution, err := apiClient.GetTestWorkflowExecution(summary.Id)
			if err != nil {
				ui.Warn(fmt.Sprintf("Warning: failed to get execution %s: %s (skipping)", summary.Id, err))
				continue
			}

			// Write execution metadata as JSON
			if err := writeExecutionJSON(execDir, execution); err != nil {
				ui.Warn(fmt.Sprintf("Warning: failed to write execution %s metadata: %s (skipping)", execution.Id, err))
				continue
			}

			// Fetch and write logs
			if err := writeExecutionLogs(apiClient, logsDir, execution.Id); err != nil {
				ui.Warn(fmt.Sprintf("Warning: failed to write logs for execution %s: %s (skipping)", execution.Id, err))
			}

			// Track max sequence per workflow
			if execution.Workflow != nil {
				wfName := execution.Workflow.Name
				if execution.Number > sequences[wfName] {
					sequences[wfName] = execution.Number
				}
			}

			totalExported++
		}

		// If we got fewer results than the page size, we've reached the end
		if len(result.Results) < exportPageSize {
			break
		}

		page++
	}

	// Write sequences.json
	if err := writeSequences(stagingDir, sequences); err != nil {
		return "", fmt.Errorf("writing sequences: %w", err)
	}

	// Package into tar.gz
	archivePath, err := createArchive(outputDir, stagingDir)
	if err != nil {
		return "", fmt.Errorf("creating archive: %w", err)
	}

	ui.Success(fmt.Sprintf("Exported %d executions to %s", totalExported, archivePath))
	return archivePath, nil
}

// writeExecutionJSON writes a single execution's metadata to executions/<id>.json.
func writeExecutionJSON(execDir string, execution testkube.TestWorkflowExecution) error {
	data, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling execution: %w", err)
	}

	filePath := filepath.Join(execDir, execution.Id+".json")
	return os.WriteFile(filePath, data, 0o644)
}

// writeExecutionLogs fetches logs for an execution and writes them to logs/<id>.log.
func writeExecutionLogs(apiClient client.Client, logsDir string, executionID string) error {
	logs, err := apiClient.GetTestWorkflowExecutionLogs(executionID)
	if err != nil {
		return fmt.Errorf("fetching logs: %w", err)
	}

	if len(logs) == 0 {
		return nil
	}

	filePath := filepath.Join(logsDir, executionID+".log")
	return os.WriteFile(filePath, logs, 0o644)
}

// writeSequences writes the workflow sequence numbers to sequences.json.
func writeSequences(stagingDir string, sequences map[string]int32) error {
	entries := make([]SequenceEntry, 0, len(sequences))
	for name, number := range sequences {
		entries = append(entries, SequenceEntry{
			WorkflowName: name,
			Number:       number,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sequences: %w", err)
	}

	return os.WriteFile(filepath.Join(stagingDir, "sequences.json"), data, 0o644)
}

// createArchive packages the staging directory into a .tar.gz file.
func createArchive(outputDir, stagingDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating output directory: %w", err)
	}

	archiveName := fmt.Sprintf("testkube-export-%s.tar.gz", time.Now().UTC().Format("20060102-150405"))
	archivePath := filepath.Join(outputDir, archiveName)

	outFile, err := os.Create(archivePath)
	if err != nil {
		return "", fmt.Errorf("creating archive file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	tarWriter := tar.NewWriter(gzWriter)

	walkErr := filepath.Walk(stagingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get path relative to staging directory
		relPath, err := filepath.Rel(stagingDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating tar header: %w", err)
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}

		_, copyErr := io.Copy(tarWriter, f)
		f.Close()
		return copyErr
	})

	// Close writers explicitly in order to ensure proper flushing
	if closeErr := tarWriter.Close(); closeErr != nil && walkErr == nil {
		walkErr = fmt.Errorf("closing tar writer: %w", closeErr)
	}
	if closeErr := gzWriter.Close(); closeErr != nil && walkErr == nil {
		walkErr = fmt.Errorf("closing gzip writer: %w", closeErr)
	}

	if walkErr != nil {
		return "", walkErr
	}

	return archivePath, nil
}
