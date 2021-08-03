package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/problem"
)

const (
	DefaultURI    = "http://localhost:8080"
	WatchInterval = time.Second
)

func NewScriptsAPI(URI string) ScriptsAPI {
	return ScriptsAPI{
		URI: URI,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

type ScriptsAPI struct {
	URI    string
	client HTTPClient
}

func (c ScriptsAPI) Get(id string) (script kubetest.Script, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/scripts/%s", id)
	resp, err := c.client.Get(uri)
	if err != nil {
		return script, err
	}

	if c.isErrorResponse(resp) {
		pr, err := c.getProblemFromResponse(resp)
		if err != nil {
			return script, fmt.Errorf("can't get problem from api response: %w", err)
		}
		return script, fmt.Errorf("api returned error: %s", pr.Detail)
	}

	return c.getScriptFromResponse(resp)
}

func (c ScriptsAPI) GetExecution(scriptID, executionID string) (execution kubetest.ScriptExecution, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/scripts/%s/executions/%s", scriptID, executionID)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}

	if c.isErrorResponse(resp) {
		pr, err := c.getProblemFromResponse(resp)
		if err != nil {
			return execution, fmt.Errorf("can't get problem from api response: %w", err)
		}
		return execution, fmt.Errorf("api returned error: %s", pr.Detail)
	}

	return c.getExecutionFromResponse(resp)
}

// GetExecutions list all executions for given script name
func (c ScriptsAPI) GetExecutions(scriptID string) (execution kubetest.ScriptExecutions, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/scripts/%s/executions", scriptID)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}

	if c.isErrorResponse(resp) {
		pr, err := c.getProblemFromResponse(resp)
		if err != nil {
			return execution, fmt.Errorf("can't get problem from api response: %w", err)
		}
		return execution, fmt.Errorf("api returned error: %s", pr.Detail)
	}

	return c.getExecutionsFromResponse(resp)
}

// Create creates new Script Custom Resource
func (c ScriptsAPI) Create(scriptName, scriptType, scriptContent, namespace string) (script kubetest.Script, err error) {
	uri := fmt.Sprintf(c.URI + "/v1/scripts")

	request := kubetest.ScriptCreateRequest{
		Name:      scriptName,
		Content:   scriptContent,
		Type_:     scriptType,
		Namespace: namespace,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return script, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return script, err
	}

	if c.isErrorResponse(resp) {
		pr, err := c.getProblemFromResponse(resp)
		if err != nil {
			return script, fmt.Errorf("can't get problem from api response: %w", err)
		}
		return script, fmt.Errorf("api returned error: %s", pr.Detail)
	}

	return c.getScriptFromResponse(resp)
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c ScriptsAPI) Execute(scriptID, executionName string, executionParams kubetest.ExecutionParams) (execution kubetest.ScriptExecution, err error) {
	// TODO call executor API - need to get parameters (what executor?) taken from CRD?
	uri := fmt.Sprintf(c.URI+"/v1/scripts/%s/executions", scriptID)

	// TODO migrate to OpenAPI ScriptExecutionRequest
	request := ExecuteRequest{
		Name:   executionName,
		Params: executionParams,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return execution, err
	}

	if c.isErrorResponse(resp) {
		pr, err := c.getProblemFromResponse(resp)
		if err != nil {
			return execution, fmt.Errorf("can't get problem from api response: %w", err)
		}
		return execution, fmt.Errorf("api returned error: %s", pr.Detail)
	}

	return c.getExecutionFromResponse(resp)
}

// GetExecutions list all executions in given script
func (c ScriptsAPI) ListScripts(namespace string) (scripts kubetest.Scripts, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/scripts?namespace=%s", namespace)
	resp, err := c.client.Get(uri)
	if err != nil {
		return scripts, fmt.Errorf("GET client error: %w", err)
	}
	defer resp.Body.Close()

	if c.isErrorResponse(resp) {
		pr, err := c.getProblemFromResponse(resp)
		if err != nil {
			return scripts, fmt.Errorf("can't get problem from api response: %w", err)
		}
		return scripts, fmt.Errorf("api returned error: %s", pr.Detail)
	}

	err = json.NewDecoder(resp.Body).Decode(&scripts)
	return
}

func (c ScriptsAPI) getExecutionFromResponse(resp *http.Response) (execution kubetest.ScriptExecution, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}

func (c ScriptsAPI) getExecutionsFromResponse(resp *http.Response) (executions kubetest.ScriptExecutions, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&executions)
	return
}

func (c ScriptsAPI) getScriptFromResponse(resp *http.Response) (script kubetest.Script, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&script)
	return
}
func (c ScriptsAPI) getProblemFromResponse(resp *http.Response) (p problem.Problem, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&p)
	return
}

func (c ScriptsAPI) isErrorResponse(resp *http.Response) bool {
	return resp.StatusCode >= 400
}
