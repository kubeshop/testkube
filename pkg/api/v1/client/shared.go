package client

import (
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// NewSharedClient creates new client for some common methods
func NewSharedClient(
	labelsTransport Transport[map[string][]string],
	serverInfoTransport Transport[testkube.ServerInfo],
	debugInfoTransport Transport[testkube.DebugInfo],
) SharedClient {
	return SharedClient{
		labelsTransport:     labelsTransport,
		serverInfoTransport: serverInfoTransport,
		debugInfoTransport:  debugInfoTransport,
	}
}

// SharedClient is a client for test workflows
type SharedClient struct {
	labelsTransport     Transport[map[string][]string]
	serverInfoTransport Transport[testkube.ServerInfo]
	debugInfoTransport  Transport[testkube.DebugInfo]
}

// ListLabels returns map of labels
func (c SharedClient) ListLabels() (map[string][]string, error) {
	uri := c.labelsTransport.GetURI("/labels")
	return c.labelsTransport.Execute(http.MethodGet, uri, nil, nil)
}

// GetServerInfo returns server info
func (c SharedClient) GetServerInfo() (info testkube.ServerInfo, err error) {
	uri := c.serverInfoTransport.GetURI("/info")
	return c.serverInfoTransport.Execute(http.MethodGet, uri, nil, nil)
}

func (c SharedClient) GetDebugInfo() (debugInfo testkube.DebugInfo, err error) {
	uri := c.debugInfoTransport.GetURI("/debug")
	return c.debugInfoTransport.Execute(http.MethodGet, uri, nil, nil)
}

// singleValueParams lifts the single-valued parameter map the transports have
// always taken into the repeating shape their implementations now work in, so
// that only the calls which need a repeated parameter have to think about it.
func singleValueParams(params map[string]string) map[string][]string {
	if params == nil {
		return nil
	}
	values := make(map[string][]string, len(params))
	for key, value := range params {
		values[key] = []string{value}
	}
	return values
}
