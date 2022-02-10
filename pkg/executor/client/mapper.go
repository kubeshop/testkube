package client

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	testsmapper "github.com/kubeshop/testkube/pkg/mapper/tests"
)

func MapExecutionOptionsToStartRequest(options ExecuteOptions) testkube.ExecutorStartRequest {
	// pass options to executor client get params from test execution request
	request := testkube.ExecutorStartRequest{
		Id:      options.ID,
		Type_:   options.ScriptSpec.Type_,
		Content: testsmapper.MapScriptContentFromSpec(options.ScriptSpec.Content),
		Params:  options.Request.Params,
	}

	return request
}
