package client

import (
	"time"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	testsv2 "github.com/kubeshop/testkube-operator/apis/tests/v2"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	WatchInterval = time.Second
)

type ExecuteOptions struct {
	ID           string
	TestName     string
	Namespace    string
	TestSpec     testsv2.TestSpec
	ExecutorName string
	ExecutorSpec executorv1.ExecutorSpec
	Request      testkube.ExecutionRequest
	Sync         bool
	HasSecrets   bool
}
