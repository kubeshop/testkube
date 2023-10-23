package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewExecutorClient creates new Executor client
func NewExecutorClient(executorTransport Transport[testkube.ExecutorDetails]) ExecutorClient {
	return ExecutorClient{
		executorTransport: executorTransport,
	}
}

// ExecutorClient is a client for executors
type ExecutorClient struct {
	executorTransport Transport[testkube.ExecutorDetails]
}

// GetExecutor gets executor by name
func (c ExecutorClient) GetExecutor(name string) (executor testkube.ExecutorDetails, err error) {
	uri := c.executorTransport.GetURI("/executors/%s", name)
	return c.executorTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListExecutors list all executors
func (c ExecutorClient) ListExecutors(selector string) (executors testkube.ExecutorsDetails, err error) {
	uri := c.executorTransport.GetURI("/executors")
	params := map[string]string{
		"selector": selector,
	}

	return c.executorTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateExecutor creates new Executor Custom Resource
func (c ExecutorClient) CreateExecutor(options UpsertExecutorOptions) (executor testkube.ExecutorDetails, err error) {
	uri := c.executorTransport.GetURI("/executors")
	request := testkube.ExecutorUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return executor, err
	}

	return c.executorTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateExecutor updates Executor Custom Resource
func (c ExecutorClient) UpdateExecutor(options UpdateExecutorOptions) (executor testkube.ExecutorDetails, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.executorTransport.GetURI("/executors/%s", name)
	request := testkube.ExecutorUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return executor, err
	}

	return c.executorTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteExecutors deletes all executors
func (c ExecutorClient) DeleteExecutors(selector string) (err error) {
	uri := c.executorTransport.GetURI("/executors")
	return c.executorTransport.Delete(uri, selector, true)
}

// DeleteExecutor deletes single executor by name
func (c ExecutorClient) DeleteExecutor(name string) (err error) {
	uri := c.executorTransport.GetURI("/executors/%s", name)
	return c.executorTransport.Delete(uri, "", true)
}
