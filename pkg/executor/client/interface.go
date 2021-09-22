package client

import (
	"io"
	"net/http"

	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

// ResultEvent event passed when watching execution changes
type ResultEvent struct {
	Result kubtest.Result
	Error  error
}

// ExecutorClient abstraction to implement new executors
type ExecutorClient interface {
	// Watch returns ExecuteEvents stream
	Watch(id string) (events chan ResultEvent)

	// Get synnchronous request to executor to get kubtestExecution
	Get(id string) (execution kubtest.Result, err error)

	// Execute starts new external script execution, reads data and returns ID
	// execution is started asynchronously client can check later for results
	Execute(options ExecuteOptions) (execution kubtest.Result, err error)

	// Abort aborts pending execution, do nothing when there is no pending execution
	Abort(id string) (err error)
}

// HTTPClient interface for getting REST based requests
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}
