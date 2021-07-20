package client

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/kubeshop/kubetest/pkg/api/executor"
)

const DefaultURI = "http://localhost:8082"

type HTTPClient struct {
	URI    string
	client http.Client
}

func (c HTTPClient) Watch(id string) {

}

func (c HTTPClient) Get(id string) {

}

// Execute starts new external script execution, reads data and returns ID
// Execution is started asynchronously client can check later for results
func (c HTTPClient) Execute(content string) (id string, err error) {

	// create request
	request := executor.ExecuteRequest{
		Metadata: json.RawMessage([]byte(content)),
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// call executor API
	resp, err := c.client.Post(c.URI+"/v1/executions/", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// parse response
	var execution executor.Execution
	err = json.NewDecoder(resp.Body).Decode(&execution)
	if err != nil {
		return "", err
	}
	return execution.Id, nil
}
