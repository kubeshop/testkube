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

// ListExecutors list all executors

// CreateExecutor creates new Executor Custom Resource

// UpdateExecutor updates Executor Custom Resource

// DeleteExecutors deletes all executors

// DeleteExecutor deletes single executor by name
