package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

const (
	DefaultURI    = "http://localhost:8080"
	WatchInterval = time.Second
)

func NewRESTClient(URI string) ScriptsAPI {
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
	return c.getScriptFromResponse(resp)
}

func (c ScriptsAPI) GetExecution(scriptID, executionID string) (execution kubetest.ScriptExecution, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/scripts/%s/executions/%s", scriptID, executionID)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c ScriptsAPI) Execute(scriptID string) (execution kubetest.ScriptExecution, err error) {
	// TODO call executor API - need to have parameters (what executor?) taken from CRD?
	uri := fmt.Sprintf(c.URI+"/v1/scripts/%s/executions", scriptID)
	resp, err := c.client.Post(uri, "application/json", nil)
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

func (c ScriptsAPI) getExecutionFromResponse(resp *http.Response) (execution kubetest.ScriptExecution, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}

func (c ScriptsAPI) getScriptFromResponse(resp *http.Response) (script kubetest.Script, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&script)
	return
}
