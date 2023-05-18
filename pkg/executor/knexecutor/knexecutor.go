package knexecutor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/executor/output"
	"github.com/kubeshop/testkube/pkg/ui"
)

const knativeEndpoint = "http://host.docker.internal:8080"

func Execute(execution *testkube.Execution) (*testkube.ExecutionResult, error) {
	test, err := json.Marshal(*execution)
	if err != nil {
		output.PrintLogf("%s could not marshal execution", ui.IconCross, err)
		return &testkube.ExecutionResult{Status: testkube.ExecutionStatusFailed}, fmt.Errorf("couldn't create POST to knative: %v", err)
	}

	output.PrintLogf("Using endpoint %s", knativeEndpoint)
	r, err := http.NewRequest("POST", knativeEndpoint, bytes.NewBuffer(test))
	if err != nil {
		output.PrintLogf("%s could not create POST to knative", ui.IconCross, err)
		return &testkube.ExecutionResult{Status: testkube.ExecutionStatusFailed}, fmt.Errorf("couldn't create POST to knative: %v", err)
	}

	output.PrintLogf("Sending request")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		output.PrintLogf("%s could not POST to knative", ui.IconCross, err)
		return &testkube.ExecutionResult{Status: testkube.ExecutionStatusFailed}, fmt.Errorf("couldn't POST to knative: %v", err)
	}

	defer res.Body.Close()

	output.PrintLogf("processing result")
	post := &testkube.ExecutionResult{}
	derr := json.NewDecoder(res.Body).Decode(post)
	if derr != nil {
		output.PrintLogf("%s could not decode response", ui.IconCross, err)
		return &testkube.ExecutionResult{Status: testkube.ExecutionStatusFailed}, fmt.Errorf("couldn't decode answer: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		output.PrintLog("wrong status code")
		return &testkube.ExecutionResult{Status: testkube.ExecutionStatusFailed}, fmt.Errorf("wrong status code: %v", res.StatusCode)
	}

	output.PrintLogf("got result %v", res)

	output.PrintResult(*post)
	return post, nil
}
