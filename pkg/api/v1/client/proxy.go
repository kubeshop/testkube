package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubeshop/kubtest/pkg/api/v1/kubtest"
	"github.com/kubeshop/kubtest/pkg/problem"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
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

func NewProxyScriptsAPI(client *kubernetes.Clientset, config ProxyConfig) ProxyScriptsAPI {
	return ProxyScriptsAPI{
		client: client,
		config: config,
	}
}

func NewProxyConfig(namespace string) ProxyConfig {
	return ProxyConfig{
		Namespace:   namespace,
		ServiceName: "kubtest-api-server",
		ServicePort: 8088,
	}
}

type ProxyConfig struct {
	// Namespace where kubtest is installed
	Namespace string
	// API Server service name
	ServiceName string
	// API Server service port
	ServicePort int
}

type ProxyScriptsAPI struct {
	client *kubernetes.Clientset
	config ProxyConfig
}

func (c ProxyScriptsAPI) GetScript(id string) (script kubtest.Script, err error) {
	uri := c.getURI("/scripts/%s", id)
	req := c.GetProxy("GET").Suffix(uri)

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/get-script returned error: %w", err)
	}

	return c.getScriptFromResponse(resp)
}

func (c ProxyScriptsAPI) GetExecution(scriptID, executionID string) (execution kubtest.Execution, err error) {
	uri := c.getURI("/scripts/%s/executions/%s", scriptID, executionID)
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/get-execution returned error: %w", err)
	}

	return c.getExecutionFromResponse(resp)
}

// ListExecutions list all executions for given script name
func (c ProxyScriptsAPI) ListExecutions(scriptID string) (executions kubtest.ExecutionsSummary, err error) {
	uri := c.getURI("/scripts/%s/executions", scriptID)
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executions, fmt.Errorf("api/get-executions returned error: %w", err)
	}

	return c.getExecutionsFromResponse(resp)
}

// CreateScript creates new Script Custom Resource
func (c ProxyScriptsAPI) CreateScript(options CreateScriptOptions) (script kubtest.Script, err error) {
	uri := c.getURI("/scripts")

	request := kubtest.ScriptCreateRequest(options)

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
func (c ProxyScriptsAPI) ExecuteScript(id, namespace, executionName string, executionParams map[string]string) (execution kubtest.Execution, err error) {
	// TODO call executor API - need to get parameters (what executor?) taken from CRD?
	uri := c.getURI("/scripts/%s/executions", id)

	request := kubtest.ExecutionRequest{
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
func (c ProxyScriptsAPI) ListScripts(namespace string) (scripts kubtest.Scripts, err error) {
	uri := c.getURI("/scripts")
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("namespace", namespace)

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return scripts, fmt.Errorf("api/list-scripts returned error: %w", err)
	}

	return c.getScriptsFromResponse(resp)
}

// GetExecutions list all executions in given script
func (c ProxyScriptsAPI) AbortExecution(scriptID, id string) error {
	uri := c.getURI("/scripts/%s/executions/%s/abort", scriptID, id)
	req := c.GetProxy("POST").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return err
	}

	return nil
}

func (c ProxyScriptsAPI) getExecutionFromResponse(resp rest.Result) (execution kubtest.Execution, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return execution, err
	}

	err = json.Unmarshal(bytes, &execution)

	return execution, err
}

func (c ProxyScriptsAPI) getExecutionsFromResponse(resp rest.Result) (executions kubtest.ExecutionsSummary, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return executions, err
	}

	err = json.Unmarshal(bytes, &executions)

	return executions, err
}

func (c ProxyScriptsAPI) getScriptsFromResponse(resp rest.Result) (scripts kubtest.Scripts, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return scripts, err
	}

	err = json.Unmarshal(bytes, &scripts)

	return scripts, err
}

func (c ProxyScriptsAPI) getScriptFromResponse(resp rest.Result) (script kubtest.Script, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return script, err
	}

	err = json.Unmarshal(bytes, &script)

	return script, err
}
func (c ProxyScriptsAPI) getProblemFromResponse(resp rest.Result) (problem.Problem, error) {
	bytes, respErr := resp.Raw()

	problemResponse := problem.Problem{}
	err := json.Unmarshal(bytes, &problemResponse)

	// add kubeAPI client error to details
	if respErr != nil {
		problemResponse.Detail += ";\nresp error:" + respErr.Error()
	}

	return problemResponse, err
}

// responseError tries to lookup if response is of Problem type
func (c ProxyScriptsAPI) responseError(resp rest.Result) error {
	if resp.Error() != nil {
		pr, err := c.getProblemFromResponse(resp)

		// if can't process response return content from response
		if err != nil {
			content, _ := resp.Raw()
			return fmt.Errorf("api server response: '%s'\nerror: %w", content, resp.Error())
		}

		return fmt.Errorf("api server problem: %s", pr.Detail)
	}

	return nil
}

func (c ProxyScriptsAPI) GetProxy(requestType string) *rest.Request {
	return c.client.CoreV1().RESTClient().Verb(requestType).
		Namespace(c.config.Namespace).
		Resource("services").
		SetHeader("Content-Type", "application/json").
		Name(fmt.Sprintf("%s:%d", c.config.ServiceName, c.config.ServicePort)).
		SubResource("proxy")
}

func (c ProxyScriptsAPI) getURI(pathTemplate string, params ...interface{}) string {
	path := fmt.Sprintf(pathTemplate, params...)
	return fmt.Sprintf("%s%s", Version, path)
}
