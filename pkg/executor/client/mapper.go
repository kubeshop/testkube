package client

import (
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

func MapExecutionOptionsToStartRequest(options ExecuteOptions) testkube.ExecutorStartRequest {
	// check if repository exists in cr repository
	var respository *testkube.Repository
	if options.ScriptSpec.Repository != nil {
		respository = &testkube.Repository{
			Type_:  "git",
			Uri:    options.ScriptSpec.Repository.Uri,
			Branch: options.ScriptSpec.Repository.Branch,
			Path:   options.ScriptSpec.Repository.Path,
		}
	}

	// pass options to executor client get params from script execution request
	request := testkube.ExecutorStartRequest{
		Id:         options.ID,
		Type_:      options.ScriptSpec.Type_,
		InputType:  options.ScriptSpec.InputType,
		Content:    options.ScriptSpec.Content,
		Repository: respository,
		Params:     options.Request.Params,
	}

	return request
}
