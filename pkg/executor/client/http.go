package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

const (
	DefaultURI    = "http://localhost:8082"
	WatchInterval = time.Second
)

func NewHTTPExecutorClient(URI string) HTTPExecutorClient {
	return HTTPExecutorClient{
		URI: URI,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

type HTTPExecutorClient struct {
	URI    string
	client HTTPClient
}

// Watch will get valid execution after async Execute, execution will be returned when success or error occurs
// TODO add timeout later
func (c HTTPExecutorClient) Watch(id string, callback func(kubetest.Execution) error) (execution kubetest.Execution, err error) {
	ticker := time.NewTicker(WatchInterval)
	for range ticker.C {
		execution, err = c.Get(id)

		if cbErr := callback(execution); cbErr != nil {
			return execution, fmt.Errorf("watch callback error: %w", cbErr)
		}
		if err != nil || execution.IsCompleted() {
			return execution, err
		}
	}
	return
}

func (c HTTPExecutorClient) Get(id string) (execution kubetest.Execution, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/executions/%s", id)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c HTTPExecutorClient) Execute(content string) (execution kubetest.Execution, err error) {

	// create request
	request := kubetest.ExecuteRequest{
		Metadata: json.RawMessage([]byte(content)),
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	// TODO call executors kube API (not ready yet)
	// - need to have parameters (what executor?) taken from CRD?
	resp, err := c.client.Post(c.URI+"/v1/executions/", "application/json", bytes.NewReader(body))
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

func (c HTTPExecutorClient) getExecutionFromResponse(resp *http.Response) (execution kubetest.Execution, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}
