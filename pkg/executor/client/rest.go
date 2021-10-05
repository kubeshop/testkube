package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/kubeshop/kubtest/pkg/api/v1/kubtest"
	"github.com/kubeshop/kubtest/pkg/problem"
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

// Watch will get valid execution result after async Execute, execution will be returned when success or error occurs
// Worker should set valid state for success or error after script completion
// TODO add timeout
func (c RestExecutorClient) Watch(id string) (events chan ResultEvent) {
	events = make(chan ResultEvent)

	go func() {
		ticker := time.NewTicker(WatchInterval)
		for range ticker.C {
			result, err := c.Get(id)

			events <- ResultEvent{
				Result: result,
				Error:  err,
			}

			if err != nil || result.IsCompleted() {
				close(events)
				return
			}
		}

	}()

	return events
}

func (c RestExecutorClient) Get(id string) (execution kubtest.ExecutionResult, err error) {

	uri := fmt.Sprintf(c.URI+"/v1/executions/%s", id)
	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("rest-executor/get-execution returned error: %w", err)
	}

	result, err := c.getResultFromResponse(resp)
	fmt.Printf("executor-rest-client result: %+v\n", result)

	return result, err
}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results with Get
func (c RestExecutorClient) Execute(options ExecuteOptions) (execution kubtest.ExecutionResult, err error) {
	request := MapExecutionOptionsToStartRequest(options)
	body, err := json.Marshal(kubtest.ExecutorStartRequest(request))
	if err != nil {
		return execution, err
	}

	resp, err := c.client.Post(c.URI+"/v1/executions/", "application/json", bytes.NewReader(body))
	if err != nil {
		return execution, err
	}

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("rest-executor/execute returned error: %w", err)
	}

	return c.getResultFromResponse(resp)
}

func (c RestExecutorClient) Abort(id string) error {
	return nil
}

func (c RestExecutorClient) getResultFromResponse(resp *http.Response) (result kubtest.ExecutionResult, err error) {
	defer resp.Body.Close()

	var execution kubtest.Execution

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("can't read response body: %w", err)
	}

	if err = json.Unmarshal(bytes, &execution); err != nil {
		// if there is strange result try to decode to interface and attach to error
		var out interface{}
		if jerr := json.Unmarshal(bytes, &out); jerr != nil {
			return result, fmt.Errorf("JSON decode error: %w", fmt.Errorf("%w", jerr))
		}
		return result, fmt.Errorf("JSON decode error: %w, trying to decode response: %+v", err, out)
	}

	if execution.ExecutionResult == nil {
		var out interface{}
		if jerr := json.Unmarshal(bytes, &out); jerr != nil {
			return result, fmt.Errorf("JSON decode error: %w", fmt.Errorf("%w", jerr))
		}
		return result, fmt.Errorf("no result in execution: %w, trying to decode response: %+v", err, out)
	}

	return *execution.ExecutionResult, nil
}

func (c RestExecutorClient) responseError(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		var pr problem.Problem

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("can't get problem from api response: can't read response body %w", err)
		}
		defer resp.Body.Close()

		err = json.Unmarshal(bytes, &pr)
		if err != nil {
			return fmt.Errorf("can't get problem from api response: %w, output: %s", err, string(bytes))
		}

		return fmt.Errorf("problem: %+v", pr.Detail)
	}

	return nil
}
