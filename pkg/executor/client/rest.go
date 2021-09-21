package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

type RestExecutorConfig struct {
	URI string `default:"http://localhost:8082"`
}

func NewRestExecutorClient(config RestExecutorConfig) RestExecutorClient {
	return RestExecutorClient{
		URI: config.URI,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

type RestExecutorClient struct {
	URI    string
	client HTTPClient
}

// Watch will get valid execution after async Execute, execution will be returned when success or error occurs
// Worker should set valid state for success or error after script completion
// TODO add timeout
func (c RestExecutorClient) Watch(id string) (events chan ExecuteEvent) {
	events = make(chan ExecuteEvent)

	go func() {
		ticker := time.NewTicker(WatchInterval)
		for range ticker.C {
			execution, err := c.Get(id)

			events <- ExecuteEvent{
				Execution: execution,
				Error:     err,
			}

			if err != nil || execution.IsCompleted() {
				close(events)
				return
			}
		}

	}()

	return events
}

func (c RestExecutorClient) Get(id string) (execution kubtest.Execution, err error) {
	uri := fmt.Sprintf(c.URI+"/v1/executions/%s", id)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results with Get
func (c RestExecutorClient) Execute(options ExecuteOptions) (execution kubtest.Execution, err error) {
	request := MapExecutionOptionsToExecutionRequest(options)
	body, err := json.Marshal(kubtest.ExecutionRequest(request))
	if err != nil {
		return execution, err
	}

	resp, err := c.client.Post(c.URI+"/v1/executions/", "application/json", bytes.NewReader(body))
	if err != nil {
		return execution, err
	}
	return c.getExecutionFromResponse(resp)
}

func (c RestExecutorClient) Abort(id string) error {
	return nil
}

func (c RestExecutorClient) getExecutionFromResponse(resp *http.Response) (execution kubtest.Execution, err error) {
	defer resp.Body.Close()

	// parse response
	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}
