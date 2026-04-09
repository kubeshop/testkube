package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	maxBundleSize      = 10 * 1024 * 1024
	shareUploadTimeout = 60 * time.Second
)

// shareHTTPClient is used for unauthenticated uploads to the public shares endpoint.
var shareHTTPClient = &http.Client{
	Timeout: shareUploadTimeout,
}

func shareAndOpenExecution(cmd *cobra.Command, executionID, sharesAPIURL, viewerBaseURL string, skipArtifacts bool) {
	uris := common.NewMasterUris("", "", "", "", "", "", "", false)
	if sharesAPIURL == "" {
		sharesAPIURL = uris.Api
	}
	if viewerBaseURL == "" {
		viewerBaseURL = uris.Ui
	}

	client, _, err := common.GetClient(cmd)
	ui.ExitOnError("getting client", err)

	installationID := telemetry.GetMachineID()

	ui.Info("Fetching execution", executionID)
	execution, err := client.GetTestWorkflowExecution(executionID)
	ui.ExitOnError("getting execution", err)

	fullJSON, err := json.Marshal(execution)
	ui.ExitOnError("marshaling execution", err)

	var payload map[string]json.RawMessage
	err = json.Unmarshal(fullJSON, &payload)
	ui.ExitOnError("preparing execution payload", err)

	mustMarshal := func(v interface{}) json.RawMessage {
		b, err := json.Marshal(v)
		if err != nil {
			ui.ExitOnError("marshaling metadata field", err)
		}
		return b
	}

	payload["executionId"] = mustMarshal(execution.Id)
	if execution.Workflow != nil {
		payload["workflowName"] = mustMarshal(execution.Workflow.Name)
	}
	if execution.Result != nil && execution.Result.Status != nil {
		payload["status"] = mustMarshal(string(*execution.Result.Status))
	}
	payload["installationId"] = mustMarshal(installationID)

	executionJSON, err := json.Marshal(payload)
	ui.ExitOnError("marshaling execution metadata", err)

	var totalSize int64
	totalSize += int64(len(executionJSON))

	ui.Info("Fetching execution logs...")
	logs, err := client.GetTestWorkflowExecutionLogs(executionID)
	ui.ExitOnError("getting execution logs", err)
	totalSize += int64(len(logs))

	var artifactPaths []string
	var tmpDir string
	if !skipArtifacts {
		artifacts, err := client.GetTestWorkflowExecutionArtifacts(executionID)
		ui.ExitOnError("getting artifacts list", err)

		if len(artifacts) > 0 {
			tmpDir, err = os.MkdirTemp("", "testkube-share-*")
			ui.ExitOnError("creating temp directory", err)
			defer os.RemoveAll(tmpDir)

			ui.Info("Downloading artifacts", fmt.Sprintf("count = %d", len(artifacts)), "\n")
			for _, artifact := range artifacts {
				f, err := client.DownloadTestWorkflowArtifact(executionID, artifact.Name, tmpDir)
				ui.ExitOnError("downloading artifact "+artifact.Name, err)
				ui.Warn(" - downloading artifact ", artifact.Name)

				fi, err := os.Stat(f)
				ui.ExitOnError("reading artifact size", err)
				totalSize += fi.Size()

				if totalSize > maxBundleSize {
					ui.Failf("Total upload size (%d bytes) exceeds the %d byte limit. Use --skip-artifacts to share without artifacts.", totalSize, maxBundleSize)
				}

				artifactPaths = append(artifactPaths, f)
			}
			ui.NL()
		}
	}

	ui.Info("Uploading to share viewer...")
	token, err := uploadShare(sharesAPIURL, executionJSON, logs, artifactPaths)
	ui.ExitOnError("uploading share", err)

	viewerURL := fmt.Sprintf("%s/view/%s", viewerBaseURL, token)
	ui.NL()
	ui.Success("Share created successfully:", viewerURL)
	err = open.Run(viewerURL)
	ui.PrintOnError("opening browser", err)
}

type shareResponse struct {
	Token        string `json:"token"`
	ExecutionID  string `json:"executionId"`
	WorkflowName string `json:"workflowName"`
	Status       string `json:"status"`
}

func uploadShare(sharesAPIURL string, executionJSON, logs []byte, artifactPaths []string) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("execution", string(executionJSON)); err != nil {
		return "", fmt.Errorf("writing execution field: %w", err)
	}

	if len(logs) > 0 {
		part, err := writer.CreateFormFile("logs", "logs.txt")
		if err != nil {
			return "", fmt.Errorf("creating logs part: %w", err)
		}
		if _, err := part.Write(logs); err != nil {
			return "", fmt.Errorf("writing logs: %w", err)
		}
	}

	for _, artifactPath := range artifactPaths {
		filename := filepath.Base(artifactPath)
		part, err := writer.CreateFormFile("artifacts", filename)
		if err != nil {
			return "", fmt.Errorf("creating artifact part for %s: %w", filename, err)
		}
		f, err := os.Open(artifactPath)
		if err != nil {
			return "", fmt.Errorf("opening artifact %s: %w", filename, err)
		}
		if _, err := io.Copy(part, f); err != nil {
			_ = f.Close()
			return "", fmt.Errorf("writing artifact %s: %w", filename, err)
		}
		_ = f.Close()
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, sharesAPIURL+"/shares", &buf)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := shareHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading share: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusRequestEntityTooLarge {
			return "", fmt.Errorf("upload too large — use --skip-artifacts or reduce artifact sizes")
		}

		var prob struct {
			Detail string `json:"detail"`
		}
		if err := json.Unmarshal(body, &prob); err == nil && prob.Detail != "" {
			return "", fmt.Errorf("share upload failed (HTTP %d): %s", resp.StatusCode, prob.Detail)
		}

		return "", fmt.Errorf("share upload failed (HTTP %d)", resp.StatusCode)
	}

	var result shareResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("missing token in response")
	}

	return result.Token, nil
}
