package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestClient creates new Test client
func NewTestClient(testTransport Transport[testkube.Test], executionTransport Transport[testkube.Execution]) TestClient {
	return TestClient{
		testTransport:      testTransport,
		executionTransport: executionTransport,
	}
}

// TestClient is a client for tests
type TestClient struct {
	testTransport      Transport[testkube.Test]
	executionTransport Transport[testkube.Execution]
}

// GetTest returns single test by id
func (c TestClient) GetTest(id string) (test testkube.Test, err error) {
	uri := getURI("/tests/%s", id)
	return c.testTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTests list all tests
func (c TestClient) ListTests(selector string) (tests testkube.Tests, err error) {
	uri := getURI("/tests")

	params := map[string]string{
		"selector": selector,
	}

	return c.testTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateTest creates new Test Custom Resource
func (c TestClient) CreateTest(options UpsertTestOptions) (test testkube.Test, err error) {
	uri := getURI("/tests")
	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return test, err
	}

	return c.testTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTest updates Test Custom Resource
func (c TestClient) UpdateTest(options UpsertTestOptions) (test testkube.Test, err error) {
	uri := getURI("/tests/%s", options.Name)
	request := testkube.TestUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return test, err
	}

	return c.testTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteTests deletes all tests
func (c TestClient) DeleteTests(selector string) error {
	uri := getURI("/tests")
	return c.testTransport.Delete(uri, selector, true)
}

// DeleteTest deletes single test by name
func (c TestClient) DeleteTest(name string) error {
	if name == "" {
		return fmt.Errorf("test name '%s' is not valid", name)
	}
	uri := getURI("/tests/%s", name)
	return c.testTransport.Delete(uri, "", true)
}

// GetExecution returns test execution by excution id
func (c TestClient) GetExecution(executionID string) (execution testkube.Execution, err error) {
	uri := getURI("/executions/%s", executionID)
	return c.executionTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ExecuteTest starts test execution, reads data and returns ID
// execution is started asynchronously client can check later for results
func (c TestClient) ExecuteTest(id, executionName string, options ExecuteTestOptions) (execution testkube.Execution, err error) {
	uri := getURI("/tests/%s/executions", id)

	request := testkube.ExecutionRequest{
		Name:       executionName,
		ParamsFile: options.ExecutionParamsFileContent,
		Params:     options.ExecutionParams,
		Args:       options.Args,
		SecretEnvs: options.SecretEnvs,
		HttpProxy:  options.HTTPProxy,
		HttpsProxy: options.HTTPSProxy,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return execution, err
	}

	return c.executionTransport.Execute(http.MethodPost, uri, body, nil)
}

// ExecuteTests starts test executions, reads data and returns IDs
// executions are started asynchronously client can check later for results
func (c TestClient) ExecuteTests(selector string, concurrencyLevel int, options ExecuteTestOptions) (executions []testkube.Execution, err error) {
	uri := getURI("/executions")
	request := testkube.ExecutionRequest{
		ParamsFile: options.ExecutionParamsFileContent,
		Params:     options.ExecutionParams,
		Args:       options.Args,
		SecretEnvs: options.SecretEnvs,
		HttpProxy:  options.HTTPProxy,
		HttpsProxy: options.HTTPSProxy,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return executions, err
	}

	params := map[string]string{
		"selector":    selector,
		"concurrency": strconv.Itoa(concurrencyLevel),
	}

	return c.executionTransport.ExecuteMultiple(http.MethodPost, uri, body, params)
}

// AbortExecution aborts execution by testId and id
func (c TestClient) AbortExecution(testID, id string) error {
	uri := getURI("/tests/%s/executions/%s", testID, id)
	return c.executionTransport.Delete(uri, "", false)
}
