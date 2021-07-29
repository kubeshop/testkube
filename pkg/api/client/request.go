package client

import "github.com/kubeshop/kubetest/pkg/api/kubetest"

type ExecuteRequest struct {
	Name   string                   `json:"name,omitempty"`
	Params kubetest.ExecutionParams `json:"params,omitempty"`
}
