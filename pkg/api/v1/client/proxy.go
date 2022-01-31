package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/problem"
)

// check in compile time if interface is implemented
var _ Client = (*ProxyScriptsAPI)(nil)

func GetClientSet() (clientset kubernetes.Interface, err error) {
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

func NewProxyScriptsAPI(client kubernetes.Interface, config ProxyConfig) ProxyScriptsAPI {
	return ProxyScriptsAPI{
		client: client,
		config: config,
	}
}

func NewProxyConfig(namespace string) ProxyConfig {
	return ProxyConfig{
		Namespace:   namespace,
		ServiceName: "testkube-api-server",
		ServicePort: 8088,
	}
}

type ProxyConfig struct {
	// Namespace where testkube is installed
	Namespace string
	// API Server service name
	ServiceName string
	// API Server service port
	ServicePort int
}

type ProxyScriptsAPI struct {
	client kubernetes.Interface
	config ProxyConfig
}

// scripts and executions -----------------------------------------------------------------------------

func (c ProxyScriptsAPI) GetScript(id, namespace string) (script testkube.Script, err error) {
	uri := c.getURI("/scripts/%s", id)
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("namespace", namespace)

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/get-script returned error: %w", err)
	}

	return c.getScriptFromResponse(resp)
}

func (c ProxyScriptsAPI) GetExecution(executionID string) (execution testkube.Execution, err error) {

	uri := c.getURI("/executions/%s", executionID)

	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/get-execution returned error: %w", err)
	}

	return c.getExecutionFromResponse(resp)
}

// ListExecutions list all executions for given script name
func (c ProxyScriptsAPI) ListExecutions(scriptID string, limit int, tags []string) (executions testkube.ExecutionsResult, err error) {

	uri := c.getURI("/executions/")

	if scriptID != "" {
		uri = fmt.Sprintf("/scripts/%s/executions", scriptID)
	}

	req := c.GetProxy("GET").
		Suffix(uri).
		Param("pageSize", fmt.Sprintf("%d", limit))

	if len(tags) > 0 {
		req.Param("tags", strings.Join(tags, ","))
	}

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executions, fmt.Errorf("api/get-executions returned error: %w", err)
	}

	return c.getExecutionsFromResponse(resp)
}

func (c ProxyScriptsAPI) DeleteScripts(namespace string) error {
	uri := c.getURI("/scripts")
	return c.makeDeleteRequest(uri, namespace, true)
}

func (c ProxyScriptsAPI) DeleteScript(name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("script name '%s' is not valid", name)
	}
	uri := c.getURI("/scripts/%s", name)
	return c.makeDeleteRequest(uri, namespace, true)
}

// CreateScript creates new Script Custom Resource
func (c ProxyScriptsAPI) CreateScript(options UpsertScriptOptions) (script testkube.Script, err error) {
	uri := c.getURI("/scripts")

	request := testkube.ScriptUpsertRequest(options)

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

// UpdateScript creates new Script Custom Resource
func (c ProxyScriptsAPI) UpdateScript(options UpsertScriptOptions) (script testkube.Script, err error) {
	uri := c.getURI("/scripts/%s", options.Name)

	request := testkube.ScriptUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return script, err
	}

	req := c.GetProxy("PATCH").Suffix(uri).Body(body)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/udpate-script returned error: %w", err)
	}

	return c.getScriptFromResponse(resp)
}

// ExecuteScript starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c ProxyScriptsAPI) ExecuteScript(id, namespace, executionName string, executionParams map[string]string) (execution testkube.Execution, err error) {
	uri := c.getURI("/scripts/%s/executions", id)

	// get script to get script tags
	script, err := c.GetScript(id, namespace)
	if err != nil {
		return execution, nil
	}

	request := testkube.ExecutionRequest{
		Name:      executionName,
		Namespace: namespace,
		Params:    executionParams,
		Tags:      script.Tags,
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

func (c ProxyScriptsAPI) Logs(id string) (logs chan output.Output, err error) {
	logs = make(chan output.Output)
	uri := c.getURI("/executions/%s/logs", id)

	resp, err := c.GetProxy("GET").
		Suffix(uri).
		SetHeader("Accept", "text/event-stream").
		Stream(context.Background())

	go func() {
		defer close(logs)
		defer resp.Close()

		StreamToLogsChannel(resp, logs)
	}()

	return
}

// GetExecutions list all executions in given script
func (c ProxyScriptsAPI) ListScripts(namespace string, tags []string) (scripts testkube.Scripts, err error) {
	uri := c.getURI("/scripts")
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("namespace", namespace)

	if len(tags) > 0 {
		req.Param("tags", strings.Join(tags, ","))
	}

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return scripts, fmt.Errorf("api/list-scripts returned error: %w", err)
	}

	return c.getScriptsFromResponse(resp)
}

// GetExecutions list all executions in given script
func (c ProxyScriptsAPI) AbortExecution(scriptID, id string) error {
	uri := c.getURI("/scripts/%s/executions/%s", scriptID, id)
	return c.makeDeleteRequest(uri, "testkube", false)
}

// executor --------------------------------------------------------------------------------

func (c ProxyScriptsAPI) CreateExecutor(options CreateExecutorOptions) (executor testkube.ExecutorDetails, err error) {
	uri := c.getURI("/executors")

	request := testkube.ExecutorCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return executor, err
	}

	req := c.GetProxy("POST").Suffix(uri).Body(body)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executor, fmt.Errorf("api/create-script returned error: %w", err)
	}

	return c.getExecutorDetailsFromResponse(resp)
}

func (c ProxyScriptsAPI) GetExecutor(name string) (executor testkube.ExecutorDetails, err error) {
	uri := c.getURI("/executors/%s", name)
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executor, fmt.Errorf("api/get-executor returned error: %w", err)
	}

	return c.getExecutorDetailsFromResponse(resp)
}

func (c ProxyScriptsAPI) ListExecutors() (executors testkube.ExecutorsDetails, err error) {
	uri := c.getURI("/executors")
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("namespace", "testkube")

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executors, fmt.Errorf("api/list-executors returned error: %w", err)
	}

	return c.getExecutorsDetailsFromResponse(resp)
}

func (c ProxyScriptsAPI) DeleteExecutor(name string) (err error) {
	uri := c.getURI("/executors/%s", name)
	return c.makeDeleteRequest(uri, "testkube", false)
}

// maintenance --------------------------------------------------------------------------------

func (c ProxyScriptsAPI) GetServerInfo() (info testkube.ServerInfo, err error) {
	uri := c.getURI("/info")
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())
	if resp.Error() != nil {
		return info, resp.Error()
	}

	bytes, err := resp.Raw()
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(bytes, &info)

	return

}

func (c ProxyScriptsAPI) GetProxy(requestType string) *rest.Request {
	return c.client.CoreV1().RESTClient().Verb(requestType).
		Namespace(c.config.Namespace).
		Resource("services").
		SetHeader("Content-Type", "application/json").
		Name(fmt.Sprintf("%s:%d", c.config.ServiceName, c.config.ServicePort)).
		SubResource("proxy")
}

func (c ProxyScriptsAPI) getExecutionFromResponse(resp rest.Result) (execution testkube.Execution, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return execution, err
	}

	err = json.Unmarshal(bytes, &execution)

	return execution, err
}

func (c ProxyScriptsAPI) getExecutionsFromResponse(resp rest.Result) (executions testkube.ExecutionsResult, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return executions, err
	}

	err = json.Unmarshal(bytes, &executions)

	return executions, err
}

func (c ProxyScriptsAPI) getScriptsFromResponse(resp rest.Result) (scripts testkube.Scripts, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return scripts, err
	}

	err = json.Unmarshal(bytes, &scripts)

	return scripts, err
}

func (c ProxyScriptsAPI) getExecutorsDetailsFromResponse(resp rest.Result) (executors testkube.ExecutorsDetails, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return executors, err
	}

	err = json.Unmarshal(bytes, &executors)

	return executors, err
}

func (c ProxyScriptsAPI) getScriptFromResponse(resp rest.Result) (script testkube.Script, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return script, err
	}

	err = json.Unmarshal(bytes, &script)

	return script, err
}

func (c ProxyScriptsAPI) getExecutorDetailsFromResponse(resp rest.Result) (executor testkube.ExecutorDetails, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return executor, err
	}

	err = json.Unmarshal(bytes, &executor)

	return executor, err
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

func (c ProxyScriptsAPI) getURI(pathTemplate string, params ...interface{}) string {
	path := fmt.Sprintf(pathTemplate, params...)
	return fmt.Sprintf("%s%s", Version, path)
}

func (c ProxyScriptsAPI) makeDeleteRequest(uri string, namespace string, isContentExpected bool) error {

	req := c.GetProxy("DELETE").
		Suffix(uri).
		Param("namespace", namespace)
	resp := req.Do(context.Background())

	if resp.Error() != nil {
		return resp.Error()
	}

	if err := c.responseError(resp); err != nil {
		return err
	}

	if isContentExpected {
		var code int
		resp.StatusCode(&code)
		if code != http.StatusNoContent {
			respBody, err := resp.Raw()
			if err != nil {
				return err
			}
			return fmt.Errorf("request returned error: %s", respBody)
		}
	}

	return nil
}

func (c ProxyScriptsAPI) GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.getURI("/executions/%s/artifacts", executionID)
	req := c.GetProxy("GET").
		Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return artifacts, fmt.Errorf("api/list-artifacts returned error: %w", err)
	}

	return c.getArtifactsFromResponse(resp)

}

func (c ProxyScriptsAPI) DownloadFile(executionID, fileName, destination string) (artifact string, err error) {
	uri := c.getURI("/executions/%s/artifacts/%s", executionID, url.QueryEscape(fileName))
	req, err := c.GetProxy("GET").
		Suffix(uri).
		SetHeader("Accept", "text/event-stream").
		Stream(context.Background())
	if err != nil {
		return "", err
	}

	defer req.Close()

	f, err := os.Create(filepath.Join(destination, filepath.Base(fileName)))
	if err != nil {
		return "", err
	}

	if _, err := f.ReadFrom(req); err != nil {
		return "", err
	}

	defer f.Close()
	return f.Name(), err
}

func (c ProxyScriptsAPI) getArtifactsFromResponse(resp rest.Result) (artifacts []testkube.Artifact, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return artifacts, err
	}

	err = json.Unmarshal(bytes, &artifacts)

	return artifacts, err
}

// --------------- tests --------------------------

func (c ProxyScriptsAPI) GetTest(id, namespace string) (script testkube.Test, err error) {
	uri := c.getURI("/tests/%s", id)
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("namespace", namespace)

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/get-script returned error: %w", err)
	}

	return c.getTestFromResponse(resp)
}

func (c ProxyScriptsAPI) DeleteTest(name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("script name '%s' is not valid", name)
	}
	uri := c.getURI("/tests/%s", name)
	return c.makeDeleteRequest(uri, namespace, true)
}

func (c ProxyScriptsAPI) DeleteTests(namespace string) error {
	uri := c.getURI("/tests")
	return c.makeDeleteRequest(uri, namespace, true)
}

func (c ProxyScriptsAPI) ListTests(namespace string, tags []string) (scripts testkube.Tests, err error) {
	uri := c.getURI("/tests")
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("namespace", namespace)

	if len(tags) > 0 {
		req.Param("tags", strings.Join(tags, ","))
	}

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return scripts, fmt.Errorf("api/list-scripts returned error: %w", err)
	}

	return c.getTestsFromResponse(resp)
}

// CreateTest creates new Test Custom Resource
func (c ProxyScriptsAPI) CreateTest(options UpsertTestOptions) (script testkube.Test, err error) {
	uri := c.getURI("/tests")

	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return script, err
	}

	req := c.GetProxy("POST").Suffix(uri).Body(body)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/create-script returned error: %w", err)
	}

	return c.getTestFromResponse(resp)
}

// UpdateTest creates new Test Custom Resource
func (c ProxyScriptsAPI) UpdateTest(options UpsertTestOptions) (script testkube.Test, err error) {
	uri := c.getURI("/tests/%s", options.Name)

	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return script, err
	}

	req := c.GetProxy("PATCH").Suffix(uri).Body(body)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return script, fmt.Errorf("api/udpate-script returned error: %w", err)
	}

	return c.getTestFromResponse(resp)
}

func (c ProxyScriptsAPI) getTestFromResponse(resp rest.Result) (test testkube.Test, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return test, err
	}

	err = json.Unmarshal(bytes, &test)

	return test, err
}

// ExecuteTest starts new external test execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c ProxyScriptsAPI) ExecuteTest(id, namespace, executionName string, executionParams map[string]string) (execution testkube.TestExecution, err error) {
	uri := c.getURI("/tests/%s/executions", id)

	request := testkube.ExecutionRequest{
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
		return execution, fmt.Errorf("api/execute-test returned error: %w", err)
	}

	return c.getTestExecutionFromResponse(resp)
}

func (c ProxyScriptsAPI) GetTestExecution(executionID string) (execution testkube.TestExecution, err error) {
	uri := c.getURI("/test-executions/%s", executionID)
	req := c.GetProxy("GET").Suffix(uri)
	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/get-execution returned error: %w", err)
	}

	return c.getTestExecutionFromResponse(resp)
}

// WatchTestExecution watches for changes in test executions
func (c ProxyScriptsAPI) WatchTestExecution(executionID string) (executionCh chan testkube.TestExecution, err error) {
	executionCh = make(chan testkube.TestExecution)

	go func() {
		execution, err := c.GetTestExecution(executionID)
		if err != nil {
			close(executionCh)
			return
		}
		executionCh <- execution
		for range time.NewTicker(time.Second).C {
			execution, err = c.GetTestExecution(executionID)
			if err != nil {
				close(executionCh)
				return
			}

			if execution.IsCompleted() {
				close(executionCh)
				return
			}

			executionCh <- execution
		}
	}()
	return
}

// ListExecutions list all executions for given test name
func (c ProxyScriptsAPI) ListTestExecutions(testID string, limit int, tags []string) (executions testkube.TestExecutionsResult, err error) {
	uri := c.getURI("/test-executions")
	req := c.GetProxy("GET").
		Suffix(uri).
		Param("pageSize", fmt.Sprintf("%d", limit))

	if len(tags) > 0 {
		req.Param("tags", strings.Join(tags, ","))
	}

	if testID != "" {
		req.Param("id", testID)
	}

	resp := req.Do(context.Background())

	if err := c.responseError(resp); err != nil {
		return executions, fmt.Errorf("api/get-executions returned error: %w", err)
	}

	return c.getTestExecutionsFromResponse(resp)
}

func (c ProxyScriptsAPI) getTestsFromResponse(resp rest.Result) (tests testkube.Tests, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return tests, err
	}

	err = json.Unmarshal(bytes, &tests)

	return tests, err
}

func (c ProxyScriptsAPI) getTestExecutionFromResponse(resp rest.Result) (execution testkube.TestExecution, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return execution, err
	}

	err = json.Unmarshal(bytes, &execution)

	return execution, err
}

func (c ProxyScriptsAPI) getTestExecutionsFromResponse(resp rest.Result) (executions testkube.TestExecutionsResult, err error) {
	bytes, err := resp.Raw()
	if err != nil {
		return executions, err
	}

	err = json.Unmarshal(bytes, &executions)

	return executions, err
}
