package client

import (
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

func MapExecutionOptionsToStartRequest(options ExecuteOptions) kubtest.ExecutorStartRequest {
	// check if repository exists in cr repository
	var respository *kubtest.Repository
	if options.ScriptSpec.Repository != nil {
		respository = &kubtest.Repository{
			Type_:  "git",
			Uri:    options.ScriptSpec.Repository.Uri,
			Branch: options.ScriptSpec.Repository.Branch,
			Path:   options.ScriptSpec.Repository.Path,
		}
	}

	// pass options to executor client get params from script execution request
	request := kubtest.ExecutorStartRequest{
		Type_:      options.ScriptSpec.Type_,
		InputType:  options.ScriptSpec.InputType,
		Content:    options.ScriptSpec.Content,
		Repository: respository,
		Params:     options.Request.Params,
	}

	return request
}
