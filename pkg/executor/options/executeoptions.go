package options

import (
	executorv1 "github.com/kubeshop/testkube-operator/api/executor/v1"
	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	"github.com/kubeshop/testkube/internal/featureflags"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
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
	ImagePullSecretNames []string
	Features             featureflags.FeatureFlags
}
