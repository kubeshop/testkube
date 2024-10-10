package executionworker

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type worker struct {
	clientSet        kubernetes.Interface
	processor        testworkflowprocessor.Processor
	inspector        imageinspector.Inspector
	baseWorkerConfig testworkflowconfig.WorkerConfig
	config           Config
}

func New(clientSet kubernetes.Interface, processor testworkflowprocessor.Processor, config Config) Worker {
	return &worker{
		clientSet: clientSet,
		processor: processor,
		config:    config,
		baseWorkerConfig: testworkflowconfig.WorkerConfig{
			Namespace:                         config.Cluster.DefaultNamespace,
			DefaultRegistry:                   config.Cluster.DefaultRegistry,
			DefaultServiceAccount:             config.Cluster.Namespaces[config.Cluster.DefaultNamespace].DefaultServiceAccountName,
			ClusterID:                         config.Cluster.Id,
			InitImage:                         constants.DefaultInitImage,
			ToolkitImage:                      constants.DefaultToolkitImage,
			ImageInspectorPersistenceEnabled:  config.ImageInspector.CacheEnabled,
			ImageInspectorPersistenceCacheKey: config.ImageInspector.CacheKey,
			ImageInspectorPersistenceCacheTTL: config.ImageInspector.CacheTTL,
			Connection:                        config.Connection,
		},
	}
}

func (w *worker) Execute(ctx context.Context, request ExecuteRequest) (*ExecuteResult, error) {
	// Build internal configuration
	cfg := testworkflowconfig.InternalConfig{
		Execution:    request.Execution,
		Workflow:     testworkflowconfig.WorkflowConfig{Name: request.Workflow.Name, Labels: request.Workflow.Labels},
		Resource:     testworkflowconfig.ResourceConfig{Id: request.Execution.Id, RootId: request.Execution.Id, FsPrefix: ""}, // TODO: Consider allowing sub-resources
		ControlPlane: request.ControlPlane,
		Worker:       w.baseWorkerConfig,
	}

	// Build list of secrets to create
	secrets := make([]corev1.Secret, 0, len(request.Secrets))
	for name, stringData := range request.Secrets {
		secrets = append(secrets, corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			StringData: stringData,
		})
	}

	// Determine execution namespace
	if request.Workflow.Spec.Job != nil && request.Workflow.Spec.Job.Namespace != "" {
		cfg.Worker.Namespace = request.Workflow.Spec.Job.Namespace
	}
	if _, ok := w.config.Cluster.Namespaces[cfg.Worker.Namespace]; !ok {
		return nil, errors.New(fmt.Sprintf("namespace %s not supported", cfg.Worker.Namespace))
	}

	// Process the Test Workflow
	bundle, err := w.processor.Bundle(ctx, &request.Workflow, testworkflowprocessor.BundleOptions{Config: cfg, Secrets: secrets})
	if err != nil {
		return nil, errors.Wrap(err, "failed to process test workflow")
	}

	// Deploy required resources
	err = bundle.Deploy(context.Background(), w.clientSet, cfg.Worker.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy test workflow")
	}

	return &ExecuteResult{
		Signature: stage.MapSignatureListToInternal(bundle.Signature),
	}, nil
}

func (w *worker) Notifications(ctx context.Context, id string) (<-chan testkube.TestWorkflowExecutionNotification, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Logs(ctx context.Context, id string) (<-chan []byte, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Get(ctx context.Context, id string) (*GetResult, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Summary(ctx context.Context, id string) (*SummaryResult, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Finished(ctx context.Context, id string) (bool, error) {
	panic("not implemented")
	return false, nil
}

func (w *worker) ListIds(ctx context.Context, options ListOptions) ([]string, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) List(ctx context.Context, options ListOptions) ([]ListResultItem, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Destroy(ctx context.Context, id string) error {
	panic("not implemented")
	return nil
}

func (w *worker) Pause(ctx context.Context, id string) error {
	panic("not implemented")
	return nil
}

func (w *worker) Resume(ctx context.Context, id string) error {
	panic("not implemented")
	return nil
}
