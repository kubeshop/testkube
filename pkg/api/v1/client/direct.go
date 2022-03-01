package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/problem"
)

const (
	ClientHTTPTimeout = time.Minute
)

// check in compile time if interface is implemented
var _ Client = (*DirectAPIClient)(nil)

type Config struct {
	URI string `default:"http://localhost:8088"`
}

var config Config

func init() {
	envconfig.Process("TESTKUBE_API", &config)
}
func NewDirectAPIClient(uri string) DirectAPIClient {
	return DirectAPIClient{
		URI: uri,
		client: &http.Client{
			Timeout: ClientHTTPTimeout,
		},
	}
}

func NewDefaultDirectAPIClient() DirectAPIClient {
	return NewDirectAPIClient(config.URI)
}

type DirectAPIClient struct {
	URI    string
	client HTTPClient
}

// tests and executions -----------------------------------------------------------------------------

func (c DirectAPIClient) GetTest(id, namespace string) (test testkube.Test, err error) {
	uri := c.getURI("/tests/%s?namespace=%s", id, namespace)
	resp, err := c.client.Get(uri)
	if err != nil {
		return test, err
	}

	if err := c.responseError(resp); err != nil {
		return test, fmt.Errorf("api/get-test returned error: %w", err)
	}

	return c.getTestFromResponse(resp)
}

func (c DirectAPIClient) GetExecution(executionID string) (execution testkube.Execution, err error) {

	uri := c.getURI("/executions/%s", executionID)

	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/get-execution returned error: %w", err)
	}

	return c.getExecutionFromResponse(resp)
}

// ListExecutions list all executions for given test name
func (c DirectAPIClient) ListExecutions(id string, limit int, tags []string) (executions testkube.ExecutionsResult, err error) {

	uri := "/executions"

	if id != "" {
		uri = fmt.Sprintf("/tests/%s/executions", id)
	}

	if len(tags) > 0 {
		uri = c.getURI("%s?pageSize=%d&tags=%s", uri, limit, strings.Join(tags, ","))
	} else {
		uri = c.getURI("%s?pageSize=%d", uri, limit)
	}

	resp, err := c.client.Get(uri)
	if err != nil {
		return executions, err
	}

	if err := c.responseError(resp); err != nil {
		return executions, fmt.Errorf("api/get-executions returned error: %w", err)
	}

	return c.getExecutionsFromResponse(resp)
}

func (c DirectAPIClient) DeleteTests(namespace string) error {
	uri := c.getURI("/tests?namespace=%s", namespace)
	return c.makeDeleteRequest(uri, true)
}

func (c DirectAPIClient) DeleteTest(name string, namespace string) error {
	if name == "" {
		return fmt.Errorf("test name '%s' is not valid", name)
	}
	uri := c.getURI("/tests/%s?namespace=%s", name, namespace)
	return c.makeDeleteRequest(uri, true)
}

// CreateTest creates new Test Custom Resource
func (c DirectAPIClient) CreateTest(options UpsertTestOptions) (test testkube.Test, err error) {
	uri := c.getURI("/tests")

	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return test, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return test, err
	}

	if err := c.responseError(resp); err != nil {
		return test, fmt.Errorf("api/create-test returned error: %w", err)
	}

	return c.getTestFromResponse(resp)
}

// UpdateTest updates Test Custom Resource
func (c DirectAPIClient) UpdateTest(options UpsertTestOptions) (test testkube.Test, err error) {
	uri := c.getURI("/tests/%s", options.Name)
	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return test, err
	}

	req, err := http.NewRequest("PATCH", uri, bytes.NewReader(body))
	req.Header.Add("Content-type", "application/json")
	if err != nil {
		return test, fmt.Errorf("prepare request error: %w", err)
	}
	resp, err := c.client.Do(req)

	if err != nil {
		return test, err
	}

	if err := c.responseError(resp); err != nil {
		return test, fmt.Errorf("api/update-test returned error: %w", err)
	}

	return c.getTestFromResponse(resp)
}

// ExecuteTest starts new external test execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c DirectAPIClient) ExecuteTest(id, namespace, executionName string, executionParams map[string]string, executionParamsFileContent string, args []string) (execution testkube.Execution, err error) {
	uri := c.getURI("/tests/%s/executions", id)

	// get test to get test tags
	test, err := c.GetTest(id, namespace)
	if err != nil {
		return execution, nil
	}

	request := testkube.ExecutionRequest{
		Name:       executionName,
		Namespace:  namespace,
		ParamsFile: executionParamsFileContent,
		Params:     executionParams,
		Tags:       test.Tags,
		Args:       args,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return execution, err
	}

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/execute-test returned error: %w", err)
	}

	return c.getExecutionFromResponse(resp)
}

// Logs reads logs from API SSE endpoint asynchronously
func (c DirectAPIClient) Logs(id string) (logs chan output.Output, err error) {
	logs = make(chan output.Output)
	uri := c.getURI("/executions/%s/logs", id)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return logs, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return logs, err
	}

	go func() {
		defer close(logs)
		defer resp.Body.Close()

		StreamToLogsChannel(resp.Body, logs)
	}()

	return
}

// ListTests list all tests in given namespace
func (c DirectAPIClient) ListTests(namespace string, tags []string) (tests testkube.Tests, err error) {
	var uri string
	if len(tags) > 0 {
		uri = c.getURI("/tests?namespace=%s&tags=%s", namespace, strings.Join(tags, ","))
	} else {
		uri = c.getURI("/tests?namespace=%s", namespace)
	}

	resp, err := c.client.Get(uri)
	if err != nil {
		return tests, fmt.Errorf("client.Get error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return tests, fmt.Errorf("api/list-tests returned error: %w", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&tests)
	return
}

func (c DirectAPIClient) AbortExecution(testID, id string) error {
	uri := c.getURI("/tests/%s/executions/%s", testID, id)
	err := c.makeDeleteRequest(uri, false)

	if err != nil {
		return fmt.Errorf("api/abort-test returned error: %w", err)
	}

	return nil
}

// executor --------------------------------------------------------------------------------

func (c DirectAPIClient) CreateExecutor(options CreateExecutorOptions) (executor testkube.ExecutorDetails, err error) {
	uri := c.getURI("/executors")

	request := testkube.ExecutorCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return executor, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return executor, err
	}

	if err := c.responseError(resp); err != nil {
		return executor, fmt.Errorf("api/create-executor returned error: %w", err)
	}

	return c.getExecutorDetailsFromResponse(resp)
}

func (c DirectAPIClient) GetExecutor(name string) (executor testkube.ExecutorDetails, err error) {
	uri := c.getURI("/executors/%s", name)
	resp, err := c.client.Get(uri)
	if err != nil {
		return executor, err
	}

	if err := c.responseError(resp); err != nil {
		return executor, fmt.Errorf("api/get-test returned error: %w", err)
	}

	return c.getExecutorDetailsFromResponse(resp)

}

func (c DirectAPIClient) ListExecutors() (executors testkube.ExecutorsDetails, err error) {
	uri := c.getURI("/executors?namespace=%s", "testkube")
	resp, err := c.client.Get(uri)
	if err != nil {
		return executors, fmt.Errorf("client.Get error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return executors, fmt.Errorf("api/list-exeutors returned error: %w", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&executors)
	return

}

func (c DirectAPIClient) DeleteExecutor(name string) (err error) {
	uri := c.getURI("/executors/%s?namespace=%s", name, "testkube")
	req, err := http.NewRequest("DELETE", uri, bytes.NewReader([]byte("")))
	if err != nil {
		return fmt.Errorf("prepare request error: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return fmt.Errorf("api/list-exeutors returned error: %w", err)
	}

	return
}

// webhook --------------------------------------------------------------------------------

func (c DirectAPIClient) CreateWebhook(options CreateWebhookOptions) (webhook testkube.Webhook, err error) {
	uri := c.getURI("/webhooks")

	request := testkube.WebhookCreateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return webhook, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return webhook, err
	}

	if err := c.responseError(resp); err != nil {
		return webhook, fmt.Errorf("api/create-webhook returned error: %w", err)
	}

	return c.getWebhookFromResponse(resp)
}

func (c DirectAPIClient) GetWebhook(name string) (webhook testkube.Webhook, err error) {
	uri := c.getURI("/webhooks/%s", name)
	resp, err := c.client.Get(uri)
	if err != nil {
		return webhook, err
	}

	if err := c.responseError(resp); err != nil {
		return webhook, fmt.Errorf("api/get-webhook returned error: %w", err)
	}

	return c.getWebhookFromResponse(resp)

}

func (c DirectAPIClient) ListWebhooks() (webhooks []testkube.Webhook, err error) {
	uri := c.getURI("/webhooks?namespace=%s", "testkube")
	resp, err := c.client.Get(uri)
	if err != nil {
		return webhooks, fmt.Errorf("client.Get error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return webhooks, fmt.Errorf("api/list-exeutors returned error: %w", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&webhooks)
	return

}

func (c DirectAPIClient) DeleteWebhook(name string) (err error) {
	uri := c.getURI("/webhooks/%s?namespace=%s", name, "testkube")
	req, err := http.NewRequest("DELETE", uri, bytes.NewReader([]byte("")))
	if err != nil {
		return fmt.Errorf("prepare request error: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return fmt.Errorf("api/delete-executor returned error: %w", err)
	}

	return
}

// maintenance --------------------------------------------------------------------------------------------

func (c DirectAPIClient) GetServerInfo() (info testkube.ServerInfo, err error) {
	uri := c.getURI("/info")
	resp, err := c.client.Get(uri)
	if err != nil {
		return info, err
	}

	err = json.NewDecoder(resp.Body).Decode(&info)

	return
}

// helper funcs --------------------------------------------------------------------------------

func (c DirectAPIClient) getExecutionFromResponse(resp *http.Response) (execution testkube.Execution, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}

func (c DirectAPIClient) getExecutionsFromResponse(resp *http.Response) (executions testkube.ExecutionsResult, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&executions)

	return
}

func (c DirectAPIClient) getTestFromResponse(resp *http.Response) (test testkube.Test, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&test)
	return
}

func (c DirectAPIClient) getExecutorDetailsFromResponse(resp *http.Response) (executor testkube.ExecutorDetails, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&executor)
	return
}

func (c DirectAPIClient) getWebhookFromResponse(resp *http.Response) (webhook testkube.Webhook, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&webhook)
	return
}

func (c DirectAPIClient) getArtifactsFromResponse(resp *http.Response) (artifacts []testkube.Artifact, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&artifacts)

	return
}

func (c DirectAPIClient) responseError(resp *http.Response) error {
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

func (c DirectAPIClient) getURI(pathTemplate string, params ...interface{}) string {
	path := fmt.Sprintf(pathTemplate, params...)
	return fmt.Sprintf("%s/%s%s", c.URI, Version, path)
}

func (c DirectAPIClient) makeDeleteRequest(uri string, isContentExpected bool) error {
	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return err
	}

	if isContentExpected && resp.StatusCode != http.StatusNoContent {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("request returned error: %s", respBody)
	}

	return nil
}

// GetExecutionArtifacts list all artifacts of the execution
func (c DirectAPIClient) GetExecutionArtifacts(executionID string) (artifacts testkube.Artifacts, err error) {
	uri := c.getURI("/executions/%s/artifacts", executionID)
	resp, err := c.client.Get(uri)
	if err != nil {
		return artifacts, err
	}

	if err := c.responseError(resp); err != nil {
		return artifacts, fmt.Errorf("api/list-artifacts returned error: %w", err)
	}

	return c.getArtifactsFromResponse(resp)
}

func (c DirectAPIClient) DownloadFile(executionID, fileName, destination string) (artifact string, err error) {
	uri := c.getURI("/executions/%s/artifacts/%s", executionID, url.QueryEscape(fileName))
	resp, err := c.client.Get(uri)
	if err != nil {
		return artifact, err
	}

	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return "", fmt.Errorf("error: %d", resp.StatusCode)
	}
	split := strings.Split(fileName, "/")
	f, err := os.Create(filepath.Join(destination, split[len(split)-1]))
	if err != nil {
		return artifact, err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return artifact, err
	}

	if err := c.responseError(resp); err != nil {
		return artifact, fmt.Errorf("api/download-file returned error: %w", err)
	}

	return f.Name(), nil
}

func (c DirectAPIClient) GetTestSuite(id, namespace string) (testSuite testkube.TestSuite, err error) {
	uri := c.getURI("/test-suites/%s", id)
	resp, err := c.client.Get(uri)
	if err != nil {
		return testSuite, err
	}

	if err := c.responseError(resp); err != nil {
		return testSuite, fmt.Errorf("api/get-test returned error: %w", err)
	}

	return c.getTestSuiteFromResponse(resp)
}

// CreateTestSuite creates new TestSuite Custom Resource
func (c DirectAPIClient) CreateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error) {
	uri := c.getURI("/test-suites")

	request := testkube.TestSuiteUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return testSuite, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return testSuite, err
	}

	if err := c.responseError(resp); err != nil {
		return testSuite, fmt.Errorf("api/create-test returned error: %w", err)
	}

	return c.getTestSuiteFromResponse(resp)
}

func (c DirectAPIClient) DeleteTestSuite(name, namespace string) (err error) {
	uri := c.getURI("/test-suites/%s?namespace=%s", name, namespace)
	req, err := http.NewRequest("DELETE", uri, bytes.NewReader([]byte("")))
	if err != nil {
		return fmt.Errorf("prepare request error: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return fmt.Errorf("api/delete-test returned error: %w", err)
	}

	return
}

func (c DirectAPIClient) DeleteTestSuites(namespace string) (err error) {
	uri := c.getURI("/test-suites?namespace=%s", namespace)
	req, err := http.NewRequest("DELETE", uri, bytes.NewReader([]byte("")))
	if err != nil {
		return fmt.Errorf("prepare request error: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return fmt.Errorf("api/delete-tests returned error: %w", err)
	}

	return
}

// UpdateTestSuite updates TestSuite Custom Resource
func (c DirectAPIClient) UpdateTestSuite(options UpsertTestSuiteOptions) (testSuite testkube.TestSuite, err error) {
	uri := c.getURI("/test-suites/%s", options.Name)

	request := testkube.TestSuiteUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return testSuite, err
	}

	req, err := http.NewRequest("PATCH", uri, bytes.NewReader(body))
	req.Header.Add("Content-type", "application/json")
	if err != nil {
		return testSuite, fmt.Errorf("prepare request error: %w", err)
	}
	resp, err := c.client.Do(req)

	if err != nil {
		return testSuite, err
	}

	if err := c.responseError(resp); err != nil {
		return testSuite, fmt.Errorf("api/update-test returned error: %w", err)
	}

	return c.getTestSuiteFromResponse(resp)
}

// ListTestSuites list all tests suites in given namespace
func (c DirectAPIClient) ListTestSuites(namespace string, tags []string) (testSuites testkube.TestSuites, err error) {
	var uri string
	if len(tags) > 0 {
		uri = c.getURI("/test-suites?namespace=%s&tags=%s", namespace, strings.Join(tags, ","))
	} else {
		uri = c.getURI("/test-suites?namespace=%s", namespace)
	}

	resp, err := c.client.Get(uri)
	if err != nil {
		return testSuites, fmt.Errorf("client.Get error: %w", err)
	}
	defer resp.Body.Close()

	if err := c.responseError(resp); err != nil {
		return testSuites, fmt.Errorf("api/list-tests returned error: %w", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&testSuites)

	return
}

// ExecuteTestSuite starts test suite execution, reads data and returns ID
func (c DirectAPIClient) ExecuteTestSuite(id, namespace, executionName string, executionParams map[string]string) (execution testkube.TestSuiteExecution, err error) {
	uri := c.getURI("/test-suites/%s/executions", id)

	request := testkube.TestSuiteExecutionRequest{
		Name:      executionName,
		Namespace: namespace,
		Params:    executionParams,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	resp, err := c.client.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		return execution, err
	}

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/execute-test returned error: %w", err)
	}

	return c.getTestExecutionFromResponse(resp)
}

// WatchTestSuiteExecution watches for changes in test suite executions
func (c DirectAPIClient) WatchTestSuiteExecution(executionID string) (executionCh chan testkube.TestSuiteExecution, err error) {
	executionCh = make(chan testkube.TestSuiteExecution)

	go func() {
		execution, err := c.GetTestSuiteExecution(executionID)
		if err != nil {
			close(executionCh)
			return
		}
		executionCh <- execution
		for range time.NewTicker(time.Second).C {
			execution, err = c.GetTestSuiteExecution(executionID)
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

func (c DirectAPIClient) GetTestSuiteExecution(executionID string) (execution testkube.TestSuiteExecution, err error) {
	uri := c.getURI("/test-suite-executions/%s", executionID)

	resp, err := c.client.Get(uri)
	if err != nil {
		return execution, err
	}

	if err := c.responseError(resp); err != nil {
		return execution, fmt.Errorf("api/get-test-execution returned error: %w", err)
	}

	return c.getTestExecutionFromResponse(resp)
}

// ListExecutions list all executions for given test suite
func (c DirectAPIClient) ListTestSuiteExecutions(testSuiteName string, limit int, tags []string) (executions testkube.TestSuiteExecutionsResult, err error) {
	var uri string
	if len(tags) > 0 {
		uri = c.getURI("/test-suite-executions?id=%s&pageSize=%d&tags=%s", testSuiteName, limit, strings.Join(tags, ","))
	} else {
		uri = c.getURI("/test-suite-executions?id=%s&pageSize=%d", testSuiteName, limit)
	}

	resp, err := c.client.Get(uri)

	if err != nil {
		return executions, err
	}

	if err := c.responseError(resp); err != nil {
		return executions, fmt.Errorf("api/list-test-executions returned error: %w", err)
	}

	return c.getTestExecutionsFromResponse(resp)
}

func (c DirectAPIClient) getTestSuiteFromResponse(resp *http.Response) (testSuite testkube.TestSuite, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&testSuite)
	return
}

func (c DirectAPIClient) getTestExecutionFromResponse(resp *http.Response) (execution testkube.TestSuiteExecution, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&execution)
	return
}

func (c DirectAPIClient) getTestExecutionsFromResponse(resp *http.Response) (executions testkube.TestSuiteExecutionsResult, err error) {
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&executions)

	return
}
