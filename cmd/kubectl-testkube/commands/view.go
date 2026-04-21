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
	"strings"
	"time"

	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	"github.com/kubeshop/testkube/cmd/kubectl-testkube/config"
	apiclientv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/ui"
)

const viewUploadTimeout = 60 * time.Second

var viewHTTPClient = &http.Client{
	Timeout: viewUploadTimeout,
}

func NewViewCmd() *cobra.Command {
	var sharesAPIURL string
	var viewerBaseURL string
	var skipArtifacts bool
	var force bool
	var wait bool
	cmd := &cobra.Command{
		Use:   "view <executionId|executionName>",
		Short: "View a test workflow execution in the browser",
		Long: `Open a test workflow execution in the browser.

When connected to Testkube Cloud / Control Plane the execution is opened
directly in the dashboard — no data is uploaded.

When running standalone (kubeconfig context) the execution data is uploaded
as a public tokenized preview and the viewer URL is opened instead.

Accepts either an execution ID (UUID) or an execution name (e.g. my-workflow-12345).`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			ui.ExitOnError("loading config file", err)

			if cfg.ContextType == config.ContextTypeCloud {
				openCloudExecution(cmd, cfg, args[0])
				return
			}

			if !force {
				ui.Warn("This will make the execution data, logs, and artifacts fully public via a tokenized URL.")
				if !ui.Confirm("Do you want to continue?") {
					return
				}
			}

			uploadAndViewExecution(cmd, cfg, args[0], sharesAPIURL, viewerBaseURL, skipArtifacts, wait)
		},
	}

	cmd.Flags().StringVar(&sharesAPIURL, "shares-api-url", "", "shares API base URL (default: https://api.testkube.io)")
	cmd.Flags().StringVar(&viewerBaseURL, "viewer-url", "", "viewer base URL (default: https://view.testkube.io)")
	cmd.Flags().BoolVar(&skipArtifacts, "skip-artifacts", false, "skip uploading artifacts when sharing an execution")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "wait for the execution to finish before uploading")
	return cmd
}

func uploadAndViewExecution(cmd *cobra.Command, cfg config.Data, arg, sharesAPIURL, viewerBaseURL string, skipArtifacts, wait bool) {
	uris := common.NewMasterUris("", "", "", "", "", false)
	if sharesAPIURL == "" {
		sharesAPIURL = uris.Api
	}
	if viewerBaseURL == "" {
		viewerBaseURL = uris.View
	}

	client, _, err := common.GetClient(cmd)
	ui.ExitOnError("getting client", err)

	installationID := telemetry.GetMachineID()

	ui.Info("Fetching execution", arg)
	execution, err := client.GetTestWorkflowExecution(arg)
	ui.ExitOnError("getting execution", err)

	if execution.Result == nil || !execution.Result.IsFinished() {
		if !wait {
			ui.Failf("Execution is still running. Use --wait (-w) to wait for it to finish before uploading.")
			return
		}
		ui.Info("Waiting for execution to finish...")
		execution, err = waitForExecution(client, execution.Id)
		ui.ExitOnError("waiting for execution to finish", err)
		ui.Info("Execution finished with status", string(*execution.Result.Status))
	}

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

	ui.Info("Fetching execution logs...")
	allLogs, err := client.GetTestWorkflowExecutionLogs(execution.Id)
	ui.ExitOnError("getting execution logs", err)

	var artifactUploads []artifactUpload
	if !skipArtifacts {
		artifacts, err := client.GetTestWorkflowExecutionArtifacts(execution.Id)
		ui.ExitOnError("getting artifacts list", err)

		if len(artifacts) > 0 {
			tmpDir, err := os.MkdirTemp("", "testkube-view-*")
			ui.ExitOnError("creating temp directory", err)
			defer os.RemoveAll(tmpDir)

			ui.Info("Downloading artifacts", fmt.Sprintf("count = %d", len(artifacts)), "\n")
			for _, artifact := range artifacts {
				ui.Warn(" - downloading artifact ", artifact.Name)

				f, err := client.DownloadTestWorkflowArtifact(execution.Id, artifact.Name, tmpDir)
				ui.ExitOnError("downloading artifact "+artifact.Name, err)

				artifactUploads = append(artifactUploads, artifactUpload{
					Path: f,
					Name: artifact.Name,
				})
			}
			ui.NL()
		}
	}

	ui.Info("Uploading to viewer...")
	token, err := uploadExecution(sharesAPIURL, executionJSON, allLogs, artifactUploads)
	if err != nil {
		if cfg.TelemetryEnabled {
			out, telErr := telemetry.SendPreviewEvent(cmd, common.Version, execution.Id, int32(len(artifactUploads)), skipArtifacts, err.Error())
			if ui.Verbose && telErr != nil {
				ui.Err(telErr)
			}
			ui.Debug("telemetry view error event response", out)
		}
		ui.ExitOnError("uploading execution", err)
	}

	viewerURL := fmt.Sprintf("%s/execution-views/%s", viewerBaseURL, token)
	ui.NL()
	ui.Success("View created successfully:", viewerURL)

	if cfg.TelemetryEnabled {
		out, telErr := telemetry.SendPreviewEvent(cmd, common.Version, execution.Id, int32(len(artifactUploads)), skipArtifacts, "")
		if ui.Verbose && telErr != nil {
			ui.Err(telErr)
		}
		ui.Debug("telemetry event response", out)
	}

	err = open.Run(viewerURL)
	ui.PrintOnError("opening browser", err)
}

func waitForExecution(client apiclientv1.Client, executionID string) (testkube.TestWorkflowExecution, error) {
	var elapsed time.Duration
	for i := 0; ; i++ {
		execution, err := client.GetTestWorkflowExecution(executionID)
		if err != nil {
			return testkube.TestWorkflowExecution{}, err
		}
		if execution.Result != nil && execution.Result.IsFinished() {
			ui.Printf("\r")
			return execution, nil
		}

		delay := viewPollDelay(i)
		elapsed += delay
		ui.Printf("\r  Waiting... (elapsed %s)", elapsed.Round(time.Second))
		time.Sleep(delay)
	}
}

func viewPollDelay(iteration int) time.Duration {
	if iteration < 5 {
		return 500 * time.Millisecond
	}
	if iteration < 50 {
		return 2 * time.Second
	}
	return 5 * time.Second
}

type viewResponse struct {
	Token        string `json:"token"`
	ExecutionID  string `json:"executionId"`
	WorkflowName string `json:"workflowName"`
	Status       string `json:"status"`
}

type artifactUpload struct {
	Path string
	Name string
}

func uploadExecution(sharesAPIURL string, executionJSON, allLogs []byte, artifacts []artifactUpload) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("execution", string(executionJSON)); err != nil {
		return "", fmt.Errorf("writing execution field: %w", err)
	}

	if len(allLogs) > 0 {
		part, err := writer.CreateFormFile("logs", "logs.txt")
		if err != nil {
			return "", fmt.Errorf("creating logs part: %w", err)
		}
		if _, err := part.Write(allLogs); err != nil {
			return "", fmt.Errorf("writing logs: %w", err)
		}
	}

	for _, a := range artifacts {

		filename := filepath.ToSlash(a.Name)

		if err := writer.WriteField("artifactPaths", filename); err != nil {
			return "", fmt.Errorf("writing artifactPaths field for %s: %w", filename, err)
		}

		part, err := writer.CreateFormFile("artifacts", filename)
		if err != nil {
			return "", fmt.Errorf("creating artifact part for %s: %w", filename, err)
		}
		f, err := os.Open(a.Path)
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, sharesAPIURL+"/public-executions", &buf)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := viewHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading execution: %w", err)
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
			return "", fmt.Errorf("upload failed (HTTP %d): %s", resp.StatusCode, prob.Detail)
		}

		return "", fmt.Errorf("upload failed (HTTP %d)", resp.StatusCode)
	}

	var result viewResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("missing token in response")
	}

	return result.Token, nil
}

func openCloudExecution(cmd *cobra.Command, cfg config.Data, arg string) {
	var missing []string
	if cfg.CloudContext.UiUri == "" {
		missing = append(missing, "uiUri")
	}
	if cfg.CloudContext.OrganizationId == "" {
		missing = append(missing, "organizationId")
	}
	if cfg.CloudContext.EnvironmentId == "" {
		missing = append(missing, "environmentId")
	}
	if len(missing) > 0 {
		hint := "Run `testkube login` to authenticate with Testkube Cloud, or set it manually with `testkube set context --org-id <id> --env-id <id>`"
		if cfg.CloudContext.UiUri == "" {
			hint += " (to also set the dashboard URL, pass `--ui-uri-override <url>` to `testkube login` or `testkube set context`)"
		}
		ui.Failf("cloud context is incomplete, missing: %s. %s", strings.Join(missing, ", "), hint)
	}

	client, _, err := common.GetClient(cmd)
	ui.ExitOnError("getting client", err)

	execution, err := client.GetTestWorkflowExecution(arg)
	ui.ExitOnError("getting execution", err)

	workflowName := ""
	if execution.Workflow != nil {
		workflowName = execution.Workflow.Name
	}

	uri := fmt.Sprintf(
		"%s/organization/%s/environment/%s/dashboard/test-workflows/%s/execution/%s",
		cfg.CloudContext.UiUri,
		cfg.CloudContext.OrganizationId,
		cfg.CloudContext.EnvironmentId,
		workflowName,
		execution.Id,
	)

	ui.ExecutionLink("Execution URI:", uri)
	err = open.Run(uri)
	ui.PrintOnError("opening browser", err)
}
