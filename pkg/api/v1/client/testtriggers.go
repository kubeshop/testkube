package client

import (
	"encoding/json"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewTestTriggerClient creates new TestTrigger client
func NewTestTriggerClient(triggerTransport Transport[testkube.TestTrigger]) TestTriggerClient {
	return TestTriggerClient{
		triggerTransport: triggerTransport,
	}
}

// TestTriggerClient is a client for triggers
type TestTriggerClient struct {
	triggerTransport Transport[testkube.TestTrigger]
}

// GetTestTrigger gets trigger by name
func (c TestTriggerClient) GetTestTrigger(name string) (trigger testkube.TestTrigger, err error) {
	uri := c.triggerTransport.GetURI("/triggers/%s", name)
	return c.triggerTransport.Execute(http.MethodGet, uri, nil, nil)
}

// ListTestTriggers list all triggers
func (c TestTriggerClient) ListTestTriggers(selector string) (triggers []testkube.TestTrigger, err error) {
	uri := c.triggerTransport.GetURI("/triggers")
	params := map[string]string{
		"selector": selector,
	}

	return c.triggerTransport.ExecuteMultiple(http.MethodGet, uri, nil, params)
}

// CreateTestTrigger creates new TestTrigger Custom Resource
func (c TestTriggerClient) CreateTestTrigger(options CreateTestTriggerOptions) (trigger testkube.TestTrigger, err error) {
	uri := c.triggerTransport.GetURI("/triggers")
	request := testkube.TestTriggerUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return trigger, err
	}

	return c.triggerTransport.Execute(http.MethodPost, uri, body, nil)
}

// UpdateTestTrigger updates TestTrigger Custom Resource
func (c TestTriggerClient) UpdateTestTrigger(options UpdateTestTriggerOptions) (trigger testkube.TestTrigger, err error) {
	name := ""
	if options.Name != "" {
		name = options.Name
	}

	uri := c.triggerTransport.GetURI("/triggers/%s", name)
	request := testkube.TestTriggerUpsertRequest(options)

	body, err := json.Marshal(request)
	if err != nil {
		return trigger, err
	}

	return c.triggerTransport.Execute(http.MethodPatch, uri, body, nil)
}

// DeleteTestTriggers deletes all triggers
func (c TestTriggerClient) DeleteTestTriggers(selector string) (err error) {
	uri := c.triggerTransport.GetURI("/triggers")
	return c.triggerTransport.Delete(uri, selector, true)
}

// DeleteTestTrigger deletes single trigger by name
func (c TestTriggerClient) DeleteTestTrigger(name string) (err error) {
	uri := c.triggerTransport.GetURI("/triggers/%s", name)
	return c.triggerTransport.Delete(uri, "", true)
}
