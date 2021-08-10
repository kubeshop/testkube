package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubeshop/kubetest/pkg/api/kubetest"
	"github.com/kubeshop/kubetest/pkg/problem"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClientSet() (clientset *kubernetes.Clientset, err error) {
	clcfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return clientset, err
	}

	restcfg, err := clientcmd.NewNonInteractiveClientConfig(
		*clcfg, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return clientset, err
	}

	return kubernetes.NewForConfig(restcfg)
}

func NewProxyScriptsAPI(client *kubernetes.Clientset) ProxyScriptsAPI {
	return ProxyScriptsAPI{
		client: client,
	}
}

type ProxyScriptsAPI struct {
	client *kubernetes.Clientset
}

func (c ProxyScriptsAPI) GetScript(id string) (script kubetest.Script, err error) {
	uri := fmt.Sprintf("v1/scripts/%s", id)
	req := c.GetProxy("GET").Suffix(uri)

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/get-script returned error: %w", err)
	}

	return c.getScriptFromResponse(resp)
}

func (c ProxyScriptsAPI) GetExecution(scriptID, executionID string) (execution kubetest.ScriptExecution, err error) {
	uri := fmt.Sprintf("v1/scripts/%s/executions/%s", scriptID, executionID)
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/get-execution returned error: %w", err)
	}

	return c.getExecutionFromResponse(resp)
}

// ListExecutions list all executions for given script name
func (c ProxyScriptsAPI) ListExecutions(scriptID string) (executions kubetest.ScriptExecutions, err error) {
	uri := fmt.Sprintf("v1/scripts/%s/executions", scriptID)
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executions, fmt.Errorf("api/get-executions returned error: %w", err)
	}

	return c.getExecutionsFromResponse(resp)
}

// CreateScript creates new Script Custom Resource
func (c ProxyScriptsAPI) CreateScript(name, scriptType, content, namespace string) (script kubetest.Script, err error) {
	uri := fmt.Sprintf("/v1/scripts")

	request := kubetest.ScriptCreateRequest{
		Name:      name,
		Content:   content,
		Type_:     scriptType,
		Namespace: namespace,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return script, err
	}

	req := c.GetProxy("POST").Suffix(uri).Body(body)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/create-script returned error: %w", err)
	}

	return c.getScriptFromResponse(resp)
}

// ExecuteScript starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c ProxyScriptsAPI) ExecuteScript(id, namespace, executionName string, executionParams map[string]string) (execution kubetest.ScriptExecution, err error) {
	// TODO call executor API - need to get parameters (what executor?) taken from CRD?
	uri := fmt.Sprintf("v1/scripts/%s/executions", id)

	request := kubetest.ScriptExecutionRequest{
		Name:      executionName,
		Namespace: namespace,
		Params:    executionParams,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	req := c.GetProxy("POST").Suffix(uri).Body(body)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/execute-script returned error: %w", err)
	}

	return c.getExecutionFromResponse(resp)
}

// GetExecutions list all executions in given script
func (c ProxyScriptsAPI) ListScripts(namespace string) (scripts kubetest.Scripts, err error) {
	req := c.GetProxy("GET").
		Suffix("v1/scripts").
		Param("namespace", namespace)

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return scripts, fmt.Errorf("api/list-scripts returned error: %w", err)
	}

	return c.getScriptsFromResponse(resp)
}

func (c ProxyScriptsAPI) getExecutionFromResponse(resp rest.Result) (execution kubetest.ScriptExecution, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return execution, err
	}

	err = json.Unmarshal(bytes, &execution)

	return execution, err
}

func (c ProxyScriptsAPI) getExecutionsFromResponse(resp rest.Result) (executions kubetest.ScriptExecutions, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return executions, err
	}

	err = json.Unmarshal(bytes, &executions)

	return executions, err
}

func (c ProxyScriptsAPI) getScriptsFromResponse(resp rest.Result) (scripts kubetest.Scripts, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return scripts, err
	}

	err = json.Unmarshal(bytes, &scripts)

	return scripts, err
}

func (c ProxyScriptsAPI) getScriptFromResponse(resp rest.Result) (script kubetest.Script, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return script, err
	}

	err = json.Unmarshal(bytes, &script)

	return script, err
}
func (c ProxyScriptsAPI) getProblemFromResponse(resp rest.Result) (problemResponse problem.Problem, err error) {
	bytes, respErr := resp.Raw()

	err = json.Unmarshal(bytes, &problemResponse)

	// add kubeAPI client error to details
	if respErr != nil {
		problemResponse.Detail += ";\nresp error:" + respErr.Error()
	}

	return problemResponse, err
}

func (c ProxyScriptsAPI) responseError(resp rest.Result) error {
	if resp.Error() != nil {
		pr, err := c.getProblemFromResponse(resp)

		if err != nil {
			return fmt.Errorf("can't get problem from api response: %w", err)
		}

		return fmt.Errorf("problem: %+v", pr.Detail)
	}

	return nil
}

func (c ProxyScriptsAPI) GetProxy(requestType string) *rest.Request {

	return c.client.CoreV1().RESTClient().Verb(requestType).
		Namespace("default").
		Resource("services").
		Name("api-server-chart:8080").
		SubResource("proxy")
}
