package client

import (
	"time"

	executorv1 "github.com/kubeshop/kubtest-operator/apis/executor/v1"
	scriptv1 "github.com/kubeshop/kubtest-operator/apis/script/v1"
	"github.com/kubeshop/kubtest/pkg/api/kubtest"
)

const (
	WatchInterval = time.Second

	ExecutorTypeRest = "rest"
	ExecutorTypeJob  = "job"
)

type ExecuteOptions struct {
	ID           string
	ScriptSpec   scriptv1.ScriptSpec
	ExecutorSpec executorv1.ExecutorSpec
	Request      kubtest.ScriptExecutionRequest
}

func NewExecuteOptions() ExecuteOptions {
	options := ExecuteOptions{}
	return options
}
