package v1

import "github.com/kubeshop/kubetest/pkg/api/kubetest"

// ExecuteRequest model for api server execution incoming data
type ExecuteRequest kubetest.ScriptExecutionRequest

// CreateRequest model for api server script incoming data
type CreateRequest kubetest.ScriptCreateRequest
