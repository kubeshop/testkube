package executionworker

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
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
		Namespace: bundle.Job.Namespace,
	}, nil
}

// TODO: Better cache?
func (w *worker) hasJobAt(ctx context.Context, id, namespace string) (bool, error) {
	// TODO: consider retry
	job, err := w.clientSet.BatchV1().Jobs(namespace).Get(ctx, id, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return job != nil, nil
}

func (w *worker) hasJobTracesAt(ctx context.Context, id, namespace string) (bool, error) {
	events, err := w.clientSet.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + id,
		TypeMeta:      metav1.TypeMeta{Kind: "Job"},
		Limit:         1,
	})
	if err != nil {
		return false, err
	}
	return len(events.Items) > 0, nil
}

// TODO: Use in-memory cache (LRU?)
func (w *worker) findNamespace(ctx context.Context, id string) (string, error) {
	// Search firstly for the actual job
	has, err := w.hasJobAt(ctx, id, w.config.Cluster.DefaultNamespace)
	if err != nil || has {
		return w.config.Cluster.DefaultNamespace, err
	}
	for ns := range w.config.Cluster.Namespaces {
		has, err = w.hasJobAt(ctx, id, ns)
		if err != nil || has {
			return ns, err
		}
	}

	// Search for the traces
	has, err = w.hasJobTracesAt(ctx, id, w.config.Cluster.DefaultNamespace)
	if err != nil || has {
		return w.config.Cluster.DefaultNamespace, err
	}
	for ns := range w.config.Cluster.Namespaces {
		has, err = w.hasJobTracesAt(ctx, id, ns)
		if err != nil || has {
			return ns, err
		}
	}

	// Not found anything
	return "", ErrResourceNotFound
}

// TODO: Avoid multiple controller copies?
func (w *worker) Notifications(ctx context.Context, namespace, id string, opts NotificationsOptions) NotificationsWatcher {
	// When there is no namespace specified, find the designated namespace
	if namespace == "" {
		ns, err := w.findNamespace(ctx, id)
		if err != nil {
			watcher := newNotificationsWatcher()
			watcher.close(err)
			return watcher
		}
		return w.Notifications(ctx, ns, id, opts)
	}

	// Load the hints
	scheduledAt := time.Time{}
	if opts.ScheduledAt != nil {
		scheduledAt = *opts.ScheduledAt
	}
	var signature []stage.Signature
	if len(opts.Signature) > 0 {
		signature = stage.MapSignatureList(opts.Signature)
	}

	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err := testworkflowcontroller.New(ctx, w.clientSet, namespace, id, scheduledAt, testworkflowcontroller.ControllerOptions{
		Signature: signature,
	})
	watcher := newNotificationsWatcher()
	if errors.Is(err, testworkflowcontroller.ErrJobTimeout) {
		err = ErrResourceNotFound
	}
	if err != nil {
		watcher.close(err)
		return watcher
	}

	// Watch the resource
	watchCtx, watchCtxCancel := context.WithCancel(ctx)
	ch := ctrl.Watch(watchCtx, opts.NoFollow)
	go func() {
		for n := range ch {
			if n.Error != nil {
				watcher.close(n.Error)
				watchCtxCancel()
				return
			}
			watcher.send(n.Value.ToInternal())
		}
		watcher.close(nil)
		watchCtxCancel()
	}()
	return watcher
}

// TODO: Optimize?
// TODO: Allow fetching temporary logs too?
func (w *worker) Logs(ctx context.Context, namespace, id string, follow bool) LogsReader {
	reader := newLogsReader()
	notifications := w.Notifications(ctx, namespace, id, NotificationsOptions{
		NoFollow: !follow,
	})
	if notifications.Err() != nil {
		reader.close(notifications.Err())
		return reader
	}

	go func() {
		defer reader.Close()
		ref := ""
		for v := range notifications.Channel() {
			if v.Log != "" && !v.Temporary {
				if ref != v.Ref && v.Ref != "" {
					ref = v.Ref
					_, _ = reader.Write([]byte(instructions.SprintHint(ref, initconstants.InstructionStart)))
				}
				_, _ = reader.Write([]byte(v.Log))
			}
		}
	}()
	return reader
}

func (w *worker) Get(ctx context.Context, namespace, id string) (*GetResult, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Summary(ctx context.Context, namespace, id string) (*SummaryResult, error) {
	panic("not implemented")
	return nil, nil
}

func (w *worker) Finished(ctx context.Context, namespace, id string) (bool, error) {
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

func (w *worker) Destroy(ctx context.Context, namespace, id string) (err error) {
	if namespace == "" {
		namespace, err = w.findNamespace(ctx, id)
		if err != nil {
			return err
		}
	}
	// TODO: Move implementation there
	return testworkflowcontroller.Cleanup(ctx, w.clientSet, namespace, id)
}

func (w *worker) findPodIpAt(ctx context.Context, id, namespace string) (string, error) {
	// TODO: consider retry
	pods, err := w.clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: constants.ResourceIdLabelName + "=" + id,
		Limit:         1,
	})
	if err != nil {
		return "", err
	} else if len(pods.Items) == 0 {
		return "", ErrResourceNotFound
	}
	return pods.Items[0].Status.PodIP, nil
}

// TODO: Use in-memory cache (LRU?)
func (w *worker) findPodIp(ctx context.Context, id string) (string, error) {
	ip, err := w.findPodIpAt(ctx, id, w.config.Cluster.DefaultNamespace)
	if err != nil {
		return ip, err
	}
	for ns := range w.config.Cluster.Namespaces {
		ip, err = w.findPodIpAt(ctx, id, ns)
		if err == nil || !errors.Is(err, ErrResourceNotFound) {
			return ip, err
		}
	}
	// TODO: Handle a case when Pod (or its IP) is not available, but the job is/was there?
	return "", ErrResourceNotFound
}

func (w *worker) Pause(ctx context.Context, namespace, id string) (err error) {
	podIp := ""
	if namespace == "" {
		podIp, err = w.findPodIp(ctx, id)
	} else {
		podIp, err = w.findPodIpAt(ctx, id, namespace)
	}
	if err != nil {
		return err
	}

	// TODO: Move implementation there
	return testworkflowcontroller.Pause(ctx, podIp)
}

func (w *worker) Resume(ctx context.Context, namespace, id string) (err error) {
	podIp := ""
	if namespace == "" {
		podIp, err = w.findPodIp(ctx, id)
	} else {
		podIp, err = w.findPodIpAt(ctx, id, namespace)
	}
	if err != nil {
		return err
	}

	// TODO: Move implementation there
	return testworkflowcontroller.Resume(ctx, podIp)
}
