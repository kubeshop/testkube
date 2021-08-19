package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

const (
	WatchInterval = time.Second
)

type Config struct {
	URI string `default:"http://localhost:8082"`
}

func NewHTTPExecutorClient(config Config) HTTPExecutorClient {
	return HTTPExecutorClient{
		URI: config.URI,
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
func (c HTTPExecutorClient) Watch(id string, callback func(kubtest.Execution) error) (execution kubtest.Execution, err error) {
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

func (c HTTPExecutorClient) Get(id string) (execution kubtest.Execution, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/executions/%s", id)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c HTTPExecutorClient) Execute(content string, params map[string]string) (execution kubtest.Execution, err error) {

	// create request
	request := kubtest.ExecutionRequest{
		Metadata: content,
		Params:   params,
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

func (c HTTPExecutorClient) getExecutionFromResponse(resp *http.Response) (execution kubtest.Execution, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}
