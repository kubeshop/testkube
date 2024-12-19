package client

import (
	"net/http"
)

// NewSharedClient creates new client for some common methods
func NewSharedClient(
	labelsTransport Transport[map[string][]string],
) SharedClient {
	return SharedClient{
		labelsTransport: labelsTransport,
	}
}

// SharedClient is a client for test workflows
type SharedClient struct {
	labelsTransport Transport[map[string][]string]
}

// ListLabels returns map of labels
func (c SharedClient) ListLabels() (map[string][]string, error) {
	uri := c.labelsTransport.GetURI("/labels")
	return c.labelsTransport.Execute(http.MethodGet, uri, nil, nil)
}
