package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

type APIRequest struct {
	Method      string            // "GET", "POST", etc.
	Path        string            // "/executions/{executionId}/logs"
	PathParams  map[string]string // {executionId: "123"}
	QueryParams map[string]string // Optional query parameters
	Body        any               // Optional request body
	Headers     map[string]string // Additional headers beyond auth
}

type APIClient struct {
	config *MCPServerConfig
	client *http.Client
}

func NewAPIClient(cfg *MCPServerConfig, client *http.Client) *APIClient {
	return &APIClient{
		config: cfg,
		client: client,
	}
}

func (c *APIClient) buildURL(path string, pathParams map[string]string) string {
	// Replace path parameters in the path
	finalPath := path
	for key, value := range pathParams {
		placeholder := "{" + key + "}"
		finalPath = strings.ReplaceAll(finalPath, placeholder, value)
	}

	// Build the full URL with org/env context
	return fmt.Sprintf("%s/organizations/%s/environments/%s%s",
		c.config.ControlPlaneUrl, c.config.OrgId, c.config.EnvId, finalPath)
}

func (c *APIClient) makeRequest(ctx context.Context, apiReq APIRequest) (string, error) {
	// Build the URL
	fullURL := c.buildURL(apiReq.Path, apiReq.PathParams)

	// Add query parameters if present
	if len(apiReq.QueryParams) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %w", err)
		}

		q := u.Query()
		for key, value := range apiReq.QueryParams {
			if value != "" { // Skip empty values
				q.Add(key, value)
			}
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	// Prepare request body
	var bodyReader *bytes.Reader
	if apiReq.Body != nil {
		var bodyBytes []byte
		var err error

		// Handle string bodies (like YAML) without JSON marshalling
		if str, ok := apiReq.Body.(string); ok {
			bodyBytes = []byte(str)
		} else {
			// JSON marshal for non-string bodies
			bodyBytes, err = json.Marshal(apiReq.Body)
			if err != nil {
				return "", fmt.Errorf("failed to marshal request body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create the HTTP request
	var req *http.Request
	var err error
	if bodyReader != nil {
		req, err = http.NewRequestWithContext(ctx, apiReq.Method, fullURL, bodyReader)
	} else {
		req, err = http.NewRequestWithContext(ctx, apiReq.Method, fullURL, nil)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set standard headers
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)

	// Set default Content-Type only if not already specified in custom headers
	if _, hasContentType := apiReq.Headers["Content-Type"]; !hasContentType && apiReq.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers (these can override defaults)
	for key, value := range apiReq.Headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response body as string
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}

func (c *APIClient) ListArtifacts(ctx context.Context, executionID string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflow-executions/{executionId}/artifacts",
		PathParams: map[string]string{
			"executionId": executionID,
		},
	})
}

func (c *APIClient) ReadArtifact(ctx context.Context, executionID, filename string) (string, error) {
	// First, get the artifact (could be direct content or a URL)
	// URL encode the filename to handle special characters like forward slashes
	encodedFilename := url.QueryEscape(filename)

	response, err := c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflow-executions/{executionId}/artifacts/{filename}",
		PathParams: map[string]string{
			"executionId": executionID,
			"filename":    encodedFilename,
		},
	})
	if err != nil {
		return "", err
	}

	// Try to parse as JSON to check if it's a URL response
	var urlResponse struct {
		URL  string `json:"url"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(response), &urlResponse); err == nil {
		// Check if we have a URL to download from
		artifactURL := urlResponse.URL
		if artifactURL == "" && urlResponse.Data.URL != "" {
			artifactURL = urlResponse.Data.URL
		}

		// Only treat as URL response if we actually have a URL that looks like a URL
		if artifactURL != "" && (strings.HasPrefix(artifactURL, "http://") || strings.HasPrefix(artifactURL, "https://")) {
			// This is a URL response, we must download the content
			req, err := http.NewRequestWithContext(ctx, "GET", artifactURL, nil)
			if err != nil {
				return "", fmt.Errorf("failed to create download request: %w", err)
			}

			resp, err := c.client.Do(req)
			if err != nil {
				return "", fmt.Errorf("failed to download artifact: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return "", fmt.Errorf("failed to download artifact: %d %s", resp.StatusCode, resp.Status)
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read artifact content: %w", err)
			}

			return string(bodyBytes), nil
		}
	}

	// If not a URL response, return the content as-is (could be JSON artifact content)
	return response, nil
}

func (c *APIClient) GetExecutionLogs(ctx context.Context, executionID string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflow-executions/{executionId}/logs",
		PathParams: map[string]string{
			"executionId": executionID,
		},
	})
}

func (c *APIClient) GetExecutionInfo(ctx context.Context, workflowName, executionID string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflows/{workflowName}/executions/{executionId}",
		PathParams: map[string]string{
			"workflowName": workflowName,
			"executionId":  executionID,
		},
	})
}

func (c *APIClient) ListExecutions(ctx context.Context, params tools.ListExecutionsParams) (string, error) {
	queryParams := make(map[string]string)

	if params.Selector != "" {
		queryParams["selector"] = params.Selector
	}
	if params.TextSearch != "" {
		queryParams["textSearch"] = params.TextSearch
	}
	if params.PageSize > 0 {
		queryParams["pageSize"] = strconv.Itoa(params.PageSize)
	} else {
		queryParams["pageSize"] = "10"
	}
	if params.Page > 0 {
		queryParams["page"] = strconv.Itoa(params.Page)
	} else {
		queryParams["page"] = "1"
	}
	if params.Status != "" {
		queryParams["status"] = params.Status
	}
	if params.Since != "" {
		queryParams["since"] = params.Since
	}

	path := "/agent/test-workflow-executions"
	if params.WorkflowName != "" {
		path = "/agent/test-workflows/{workflowName}/executions"
		return c.makeRequest(ctx, APIRequest{
			Method: "GET",
			Path:   path,
			PathParams: map[string]string{
				"workflowName": params.WorkflowName,
			},
			QueryParams: queryParams,
		})
	}

	return c.makeRequest(ctx, APIRequest{
		Method:      "GET",
		Path:        path,
		QueryParams: queryParams,
	})
}

func (c *APIClient) LookupExecutionID(ctx context.Context, executionName string) (string, error) {
	workflowName, _, err := extractWorkflowNameFromExecutionName(executionName)
	if err != nil {
		return "", fmt.Errorf("invalid execution name format: %w", err)
	}

	// Use text search to find the execution
	queryParams := map[string]string{
		"textSearch": executionName,
	}

	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflows/{workflowName}/executions",
		PathParams: map[string]string{
			"workflowName": workflowName,
		},
		QueryParams: queryParams,
	})
}

// ListWorkflows retrieves workflows with optional filtering
func (c *APIClient) ListWorkflows(ctx context.Context, params tools.ListWorkflowsParams) (string, error) {
	queryParams := make(map[string]string)

	if params.ResourceGroup != "" {
		queryParams["resourceGroup"] = params.ResourceGroup
	}
	if params.Selector != "" {
		queryParams["selector"] = params.Selector
	}
	if params.TextSearch != "" {
		queryParams["textSearch"] = params.TextSearch
	}
	if params.PageSize > 0 {
		queryParams["pageSize"] = strconv.Itoa(params.PageSize)
	} else {
		queryParams["pageSize"] = "10"
	}
	if params.Page > 0 {
		queryParams["page"] = strconv.Itoa(params.Page)
	} else {
		queryParams["page"] = "1"
	}
	if params.Status != "" {
		queryParams["status"] = params.Status
	}
	if params.GroupID != "" {
		queryParams["groupId"] = params.GroupID
	}

	return c.makeRequest(ctx, APIRequest{
		Method:      "GET",
		Path:        "/agent/test-workflow-with-executions",
		QueryParams: queryParams,
	})
}

func (c *APIClient) GetWorkflow(ctx context.Context, workflowName string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflow-with-executions/{workflowName}",
		PathParams: map[string]string{
			"workflowName": workflowName,
		},
	})
}

func (c *APIClient) GetWorkflowDefinition(ctx context.Context, workflowName string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflows/{workflowName}",
		PathParams: map[string]string{
			"workflowName": workflowName,
		},
		Headers: map[string]string{
			"Accept": "text/yaml",
		},
	})
}

func (c *APIClient) CreateWorkflow(ctx context.Context, workflowDefinition string) (string, error) {

	return c.makeRequest(ctx, APIRequest{
		Method: "POST",
		Path:   "/agent/test-workflows",
		Body:   workflowDefinition,
		Headers: map[string]string{
			"Content-Type": "text/yaml",
		},
	})
}

func (c *APIClient) RunWorkflow(ctx context.Context, params tools.RunWorkflowParams) (string, error) {
	body := map[string]any{
		"config": params.Config,
		"runningContext": map[string]any{
			"actor": map[string]any{
				"type": "user",
			},
			"interface": map[string]any{
				"type": "api",
			},
		},
	}

	if params.Target != nil {
		body["target"] = params.Target
	}

	return c.makeRequest(ctx, APIRequest{
		Method: "POST",
		Path:   "/agent/test-workflows/{workflowName}/executions",
		PathParams: map[string]string{
			"workflowName": params.WorkflowName,
		},
		Body: body,
	})
}

// extracts workflow name from execution name e.g., "my-workflow-123" -> ("my-workflow", 123)
func extractWorkflowNameFromExecutionName(executionName string) (string, int, error) {
	lastDashIndex := strings.LastIndex(executionName, "-")
	if lastDashIndex == -1 {
		return "", 0, fmt.Errorf("execution name must contain a dash followed by a number")
	}

	workflowName := executionName[:lastDashIndex]
	executionNumberStr := executionName[lastDashIndex+1:]

	executionNumber, err := strconv.Atoi(executionNumberStr)
	if err != nil {
		return "", 0, fmt.Errorf("execution name must end with a number, got: %s", executionNumberStr)
	}

	return workflowName, executionNumber, nil
}

// DebugAPIClient wraps an APIClient and captures debug information from the last request
type DebugAPIClient struct {
	*APIClient
	lastDebugInfo tools.DebugInfo
	transport     *DebugRoundTripper
}

// DebugRoundTripper is an HTTP RoundTripper that captures debug information
type DebugRoundTripper struct {
	base          http.RoundTripper
	lastDebugInfo *tools.DebugInfo // Pointer so we can update the parent's debug info
}

// NewDebugRoundTripper creates a new debug-enabled HTTP RoundTripper
func NewDebugRoundTripper(base http.RoundTripper, debugInfoPtr *tools.DebugInfo) *DebugRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &DebugRoundTripper{
		base:          base,
		lastDebugInfo: debugInfoPtr,
	}
}

// RoundTrip implements the http.RoundTripper interface and captures debug info
func (d *DebugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Capture request info using flexible DebugInfo map
	debugInfo := tools.DebugInfo{
		"type":             "http",
		"request_url":      req.URL.String(),
		"request_method":   req.Method,
		"request_headers":  make(map[string]string),
		"response_headers": make(map[string]string),
	}

	// Capture request headers (excluding sensitive ones)
	requestHeaders := make(map[string]string)
	for name, values := range req.Header {
		if len(values) > 0 {
			if name == "Authorization" {
				requestHeaders[name] = "[REDACTED]"
			} else {
				requestHeaders[name] = values[0]
			}
		}
	}
	debugInfo["request_headers"] = requestHeaders

	// Capture request body if present
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			// Store body info (limit size for debug output)
			if len(bodyBytes) > 0 {
				if len(bodyBytes) <= 1024 {
					debugInfo["request_body"] = string(bodyBytes)
				} else {
					debugInfo["request_body_size"] = len(bodyBytes)
				}
			}
			// Restore the request body for the actual request
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	// Make the actual HTTP request
	start := time.Now()
	resp, err := d.base.RoundTrip(req)
	duration := time.Since(start)

	debugInfo["duration_ms"] = duration.Milliseconds()

	// Capture response info
	if err != nil {
		debugInfo["error"] = err.Error()
	} else {
		debugInfo["response_status"] = resp.StatusCode
		debugInfo["response_status_text"] = resp.Status

		// Capture response headers
		responseHeaders := make(map[string]string)
		for name, values := range resp.Header {
			if len(values) > 0 {
				responseHeaders[name] = values[0]
			}
		}
		debugInfo["response_headers"] = responseHeaders

		// Capture response body (with size limit for debug)
		if resp.Body != nil {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				debugInfo["response_body_size"] = len(bodyBytes)

				// Store a preview of the response body (limited size)
				if len(bodyBytes) <= 512 {
					debugInfo["response_body_preview"] = string(bodyBytes)
				} else {
					debugInfo["response_body_preview"] = string(bodyBytes[:512]) + "... [truncated]"
				}

				// Restore the response body for the caller
				resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
	}

	// Store debug info in the parent DebugAPIClient
	if d.lastDebugInfo != nil {
		*d.lastDebugInfo = debugInfo
	}

	return resp, err
}

// NewDebugAPIClient creates a new APIClient with debug-enabled HTTP transport
func NewDebugAPIClient(cfg *MCPServerConfig, baseClient *http.Client) *DebugAPIClient {
	if baseClient == nil {
		baseClient = &http.Client{}
	}

	debugAPIClient := &DebugAPIClient{
		APIClient: &APIClient{config: cfg},
	}

	// Create debug transport that can update our debug info
	debugTransport := NewDebugRoundTripper(baseClient.Transport, &debugAPIClient.lastDebugInfo)

	// Create HTTP client with debug transport
	debugAPIClient.APIClient.client = &http.Client{
		Transport: debugTransport,
		Timeout:   baseClient.Timeout,
	}
	debugAPIClient.transport = debugTransport

	return debugAPIClient
}

// GetLastDebugInfo returns the debug information from the last HTTP request
func (d *DebugAPIClient) GetLastDebugInfo() tools.DebugInfo {
	return d.lastDebugInfo
}
