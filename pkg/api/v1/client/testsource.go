package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestSourceClient creates new TestSource client
func NewTestSourceClient(testSourceTransport Transport[testkube.TestSource]) TestSourceClient {
	return TestSourceClient{
		testSourceTransport: testSourceTransport,
	}
}

// TestSourceClient is a client for test sources
type TestSourceClient struct {
	testSourceTransport Transport[testkube.TestSource]
}

// GetTestSource gets test source by name
func (c TestSourceClient) GetTestSource(name string) (testSource testkube.TestSource, err error) {
	uri := c.testSourceTransport.GetURI("/test-sources/%s", name)
	return c.testSourceTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestSources list all test sources
func (c TestSourceClient) ListTestSources(selector string) (testSources testkube.TestSources, err error) {
	uri := c.testSourceTransport.GetURI("/test-sources")
	params := map[string]string{
		"selector": selector,
	}

	return c.testSourceTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateTestSource creates new TestSource Custom Resource
func (c TestSourceClient) CreateTestSource(options UpsertTestSourceOptions) (testSource testkube.TestSource, err error) {
	uri := c.testSourceTransport.GetURI("/test-sources")
	request := testkube.TestSourceUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return testSource, err
	}

	return c.testSourceTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTestSource updates TestSource Custom Resource
func (c TestSourceClient) UpdateTestSource(options UpdateTestSourceOptions) (testSource testkube.TestSource, err error) {
	name := ""
	if options.Name != nil {
		name = *options.Name
	}

	uri := c.testSourceTransport.GetURI("/test-sources/%s", name)
	request := testkube.TestSourceUpdateRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return testSource, err
	}

	return c.testSourceTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteTestSources deletes all test sources
func (c TestSourceClient) DeleteTestSources(selector string) (err error) {
	uri := c.testSourceTransport.GetURI("/test-sources")
	return c.testSourceTransport.Delete(uri, selector, true)
}

// DeleteTestSource deletes single test source by name
func (c TestSourceClient) DeleteTestSource(name string) (err error) {
	uri := c.testSourceTransport.GetURI("/test-sources/%s", name)
	return c.testSourceTransport.Delete(uri, "", true)
}
