package client

import (
	"io"
	"net/http"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
)

type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
}

type Client interface {
	GetScript(id string) (script kubetest.Script, err error)
	GetExecution(scriptID, executionID string) (execution kubetest.ScriptExecution, err error)
	ListExecutions(scriptID string) (executions kubetest.ScriptExecutions, err error)
	CreateScript(name, scriptType, content, namespace string) (script kubetest.Script, err error)
	ExecuteScript(id, namespace, executionName string, executionParams map[string]string) (execution kubetest.ScriptExecution, err error)
	ListScripts(namespace string) (scripts kubetest.Scripts, err error)
}
