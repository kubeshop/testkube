package executionworker

import (
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	kubernetes2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/kubernetesworker"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
)

func NewKubernetes(clientSet kubernetes.Interface, processor testworkflowprocessor.Processor, config kubernetes2.Config) executionworkertypes.Worker {
	return kubernetes2.NewWorker(clientSet, processor, config)
}
