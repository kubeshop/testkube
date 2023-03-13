package client

import (
	"time"

	executorv1 "github.com/kubeshop/testkube-operator/apis/executor/v1"
	testsv3 "github.com/kubeshop/testkube-operator/apis/tests/v3"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

const (
	WatchInterval = time.Second
)

type ExecuteOptions struct {
	ID                   string
	TestName             string
	Namespace            string
	TestSpec             testsv3.TestSpec
	ExecutorName         string
	ExecutorSpec         executorv1.ExecutorSpec
	Request              testkube.ExecutionRequest
	Sync                 bool
	Labels               map[string]string
	UsernameSecret       *testkube.SecretRef
	TokenSecret          *testkube.SecretRef
	CertificateSecret    string
	ImageOverride        string
	ImagePullSecretNames []string
}
