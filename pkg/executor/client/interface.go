package client

import (
	"context"
	"io"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/options"
	"github.com/kubeshop/testkube/pkg/executor/output"
)

const (
	// GitUsernameSecretName is git username secret name
	GitUsernameSecretName = "git-username"
	// GitUsernameEnvVarName is git username environment var name
	GitUsernameEnvVarName = "RUNNER_GITUSERNAME"
	// GitTokenSecretName is git token secret name
	GitTokenSecretName = "git-token"
	// GitTokenEnvVarName is git token environment var name
	GitTokenEnvVarName = "RUNNER_GITTOKEN"
	// SecretTest is a test secret
	SecretTest = "secrets"
	// SecretSource is a source secret
	SecretSource = "source-secrets"
)

// TODO check if we can remove this - unused in testkube
// ResultEvent event passed when watching execution changes
type ResultEvent struct {
	Result testkube.ExecutionResult
	Error  error
}

// Executor abstraction to implement new executors
//
//go:generate mockgen -destination=./mock_executor.go -package=client "github.com/kubeshop/testkube/pkg/executor/client" Executor
type Executor interface {
	// Execute starts new external test execution, reads data and returns ID
	// execution is started asynchronously client can check later for results
	Execute(ctx context.Context, execution *testkube.Execution, options options.ExecuteOptions) (result *testkube.ExecutionResult, err error)

	// Abort aborts pending execution, do nothing when there is no pending execution
	Abort(ctx context.Context, execution *testkube.Execution) (result *testkube.ExecutionResult, err error)

	Logs(ctx context.Context, id string) (logs chan output.Output, err error)
}

// TODO check if we can remove this - unused in testkube
// HTTPClient interface for getting REST based requests
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}
