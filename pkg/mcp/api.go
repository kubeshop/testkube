package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/kubeshop/testkube/pkg/mcp/tools"
)

type ApiScope int

const (
	ApiScopeNone ApiScope = iota
	ApiScopeOrg
	ApiScopeOrgEnv
)

type APIRequest struct {
	Method      string            // "GET", "POST", etc.
	Path        string            // "/executions/{executionId}/logs"
	Scope       ApiScope          // API scope level (default: ApiScopeNone)
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

func (c *APIClient) buildApiUrl(path string, pathParams map[string]string, scope ApiScope) string {
	// Replace path parameters in the path
	finalPath := path
	for key, value := range pathParams {
		placeholder := "{" + key + "}"
		finalPath = strings.ReplaceAll(finalPath, placeholder, value)
	}

	// Build the full URL based on scope
	switch scope {
	case ApiScopeNone:
		return c.config.ControlPlaneUrl + finalPath
	case ApiScopeOrg:
		return fmt.Sprintf("%s/organizations/%s%s",
			c.config.ControlPlaneUrl, c.config.OrgId, finalPath)
	case ApiScopeOrgEnv:
		return fmt.Sprintf("%s/organizations/%s/environments/%s%s",
			c.config.ControlPlaneUrl, c.config.OrgId, c.config.EnvId, finalPath)
	default:
		return c.config.ControlPlaneUrl + finalPath
	}
}

func (c *APIClient) makeRequest(ctx context.Context, apiReq APIRequest) (string, error) {
	// Check if debug collection is active
	debugInfo := GetDebugInfo(ctx)
	if debugInfo != nil {
		debugInfo.Source = "http"
		debugInfo.Data["method"] = apiReq.Method
	}

	// Build the URL
	fullURL := c.buildApiUrl(apiReq.Path, apiReq.PathParams, apiReq.Scope)
	if debugInfo != nil {
		debugInfo.Data["url"] = fullURL
	}

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
		if debugInfo != nil {
			debugInfo.Data["status"] = resp.StatusCode
			debugInfo.Data["requestHeaders"] = redactSensitiveHeaders(req.Header)
			debugInfo.Data["responseHeaders"] = redactSensitiveHeaders(resp.Header)
		}
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response body as string
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Populate debug info if active
	if debugInfo != nil {
		debugInfo.Data["status"] = resp.StatusCode
		debugInfo.Data["requestHeaders"] = redactSensitiveHeaders(req.Header)
		debugInfo.Data["responseHeaders"] = redactSensitiveHeaders(resp.Header)
	}

	return string(bodyBytes), nil
}

var sensitiveHeaders = []string{
	"authorization",
	"cookie",
	"x-api-key",
	"x-auth-token",
}

// redactSensitiveHeaders removes sensitive information from headers
func redactSensitiveHeaders(headers http.Header) map[string][]string {
	if headers == nil {
		return nil
	}
	redacted := make(map[string][]string)
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if slices.Contains(sensitiveHeaders, lowerKey) {
			redacted[key] = []string{"[REDACTED]"}
		} else {
			redacted[key] = values
		}
	}
	return redacted
}

func (c *APIClient) ListArtifacts(ctx context.Context, executionID string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflow-executions/{executionId}/artifacts",
		Scope:  ApiScopeOrgEnv,
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
		Scope:  ApiScopeOrgEnv,
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
		Scope:  ApiScopeOrgEnv,
		PathParams: map[string]string{
			"executionId": executionID,
		},
	})
}

func (c *APIClient) GetExecutionInfo(ctx context.Context, workflowName, executionID string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflows/{workflowName}/executions/{executionId}",
		Scope:  ApiScopeOrgEnv,
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
	if params.Page >= 0 {
		queryParams["page"] = strconv.Itoa(params.Page)
	} else {
		queryParams["page"] = "0"
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
			Scope:  ApiScopeOrgEnv,
			PathParams: map[string]string{
				"workflowName": params.WorkflowName,
			},
			QueryParams: queryParams,
		})
	}

	return c.makeRequest(ctx, APIRequest{
		Method:      "GET",
		Path:        path,
		Scope:       ApiScopeOrgEnv,
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
		Scope:  ApiScopeOrgEnv,
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
	if params.Page >= 0 {
		queryParams["page"] = strconv.Itoa(params.Page)
	} else {
		queryParams["page"] = "0"
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
		Scope:       ApiScopeOrgEnv,
		QueryParams: queryParams,
	})
}

func (c *APIClient) GetWorkflow(ctx context.Context, workflowName string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflow-with-executions/{workflowName}",
		Scope:  ApiScopeOrgEnv,
		PathParams: map[string]string{
			"workflowName": workflowName,
		},
	})
}

func (c *APIClient) GetWorkflowDefinition(ctx context.Context, workflowName string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflows/{workflowName}",
		Scope:  ApiScopeOrgEnv,
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
		Scope:  ApiScopeOrgEnv,
		Body:   workflowDefinition,
		Headers: map[string]string{
			"Content-Type": "text/yaml",
		},
	})
}

func (c *APIClient) UpdateWorkflow(ctx context.Context, workflowName, workflowDefinition string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "PUT",
		Path:   "/agent/test-workflows/{workflowName}",
		Scope:  ApiScopeOrgEnv,
		PathParams: map[string]string{
			"workflowName": workflowName,
		},
		Body: workflowDefinition,
		Headers: map[string]string{
			"Content-Type": "text/yaml",
		},
	})
}

// convertTargetToExecutionTarget converts the MCP target parameter to the proper ExecutionTarget format
// Supports multiple input formats:
// 1. {"name": "agent-name"} -> {"match": {"name": ["agent-name"]}}
// 2. {"labels": {"env": "prod", "type": "runner"}} -> {"match": {"env": ["prod"], "type": ["runner"]}}
// 3. {"match": {"env": ["prod"]}} -> passed through as-is
// 4. {"not": {"env": ["dev"]}} -> passed through as-is
// 5. {"replicate": ["env"]} -> passed through as-is
func convertTargetToExecutionTarget(target map[string]any) map[string]any {
	result := make(map[string]any)

	// Handle name-based targeting (convert to match format)
	if name, exists := target["name"]; exists {
		if nameStr, ok := name.(string); ok {
			result["match"] = map[string]any{
				"name": []string{nameStr},
			}
		}
		return result
	}

	// Handle labels-based targeting (convert to match format)
	if labels, exists := target["labels"]; exists {
		if labelsMap, ok := labels.(map[string]any); ok {
			match := make(map[string]any)
			for key, value := range labelsMap {
				switch v := value.(type) {
				case string:
					match[key] = []string{v}
				case []string:
					match[key] = v
				case []any:
					// Convert []any to []string
					strSlice := make([]string, len(v))
					for i, item := range v {
						if str, ok := item.(string); ok {
							strSlice[i] = str
						}
					}
					match[key] = strSlice
				}
			}
			if len(match) > 0 {
				result["match"] = match
			}
		}
		return result
	}

	// Handle standard ExecutionTarget fields (pass through)
	if match, exists := target["match"]; exists {
		result["match"] = match
	}
	if not, exists := target["not"]; exists {
		result["not"] = not
	}
	if replicate, exists := target["replicate"]; exists {
		result["replicate"] = replicate
	}

	return result
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
		body["target"] = convertTargetToExecutionTarget(params.Target)
	}

	return c.makeRequest(ctx, APIRequest{
		Method: "POST",
		Path:   "/agent/test-workflows/{workflowName}/executions",
		Scope:  ApiScopeOrgEnv,
		PathParams: map[string]string{
			"workflowName": params.WorkflowName,
		},
		Body: body,
	})
}

func (c *APIClient) ListLabels(ctx context.Context) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/labels",
		Scope:  ApiScopeOrgEnv,
	})
}

func (c *APIClient) ListResourceGroups(ctx context.Context) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/groups",
		Scope:  ApiScopeOrg,
	})
}

func (c *APIClient) ListAgents(ctx context.Context, params tools.ListAgentsParams) (string, error) {
	queryParams := make(map[string]string)

	if params.Type != "" {
		queryParams["type"] = params.Type
	}
	if params.Capability != "" {
		queryParams["capability"] = params.Capability
	}
	if params.PageSize > 0 {
		queryParams["pageSize"] = strconv.Itoa(params.PageSize)
	} else {
		queryParams["pageSize"] = "20"
	}
	if params.Page >= 0 {
		queryParams["page"] = strconv.Itoa(params.Page)
	} else {
		queryParams["page"] = "0"
	}

	queryParams["includeDeleted"] = "false"

	// Add environment ID as a filter
	queryParams["environmentId"] = c.config.EnvId

	return c.makeRequest(ctx, APIRequest{
		Method:      "GET",
		Path:        "/agents",
		Scope:       ApiScopeOrg,
		QueryParams: queryParams,
	})
}

func (c *APIClient) AbortWorkflowExecution(ctx context.Context, workflowName, executionId string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "POST",
		Path:   "/agent/test-workflows/{workflowName}/executions/{executionId}/abort",
		Scope:  ApiScopeOrgEnv,
		PathParams: map[string]string{
			"workflowName": workflowName,
			"executionId":  executionId,
		},
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

func (c *APIClient) GetWorkflowMetrics(ctx context.Context, workflowName string) (string, error) {
	return c.makeRequest(ctx, APIRequest{
		Method: "GET",
		Path:   "/agent/test-workflows/{workflowName}/metrics",
		Scope:  ApiScopeOrgEnv,
		PathParams: map[string]string{
			"workflowName": workflowName,
		},
	})
}
