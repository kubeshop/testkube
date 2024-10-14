package executionworker

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"

	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	registry2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowcontroller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

const (
	ResumeRetryOnFailureDelay = 300 * time.Millisecond
)

type worker struct {
	clientSet        kubernetes.Interface
	processor        testworkflowprocessor.Processor
	inspector        imageinspector.Inspector
	baseWorkerConfig testworkflowconfig.WorkerConfig
	config           Config
	registry         *controllersRegistry
}

func New(clientSet kubernetes.Interface, processor testworkflowprocessor.Processor, config Config) Worker {
	namespaces := registry2.NewNamespacesRegistry(clientSet, config.Cluster.DefaultNamespace, maps.Keys(config.Cluster.Namespaces), 50)
	return &worker{
		clientSet: clientSet,
		processor: processor,
		config:    config,
		registry:  newControllersRegistry(clientSet, namespaces, 50),
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
	resourceId := request.ResourceId
	if resourceId == "" {
		resourceId = request.Execution.Id
	}

	// Build internal configuration
	cfg := testworkflowconfig.InternalConfig{
		Execution:    request.Execution,
		Workflow:     testworkflowconfig.WorkflowConfig{Name: request.Workflow.Name, Labels: request.Workflow.Labels},
		Resource:     testworkflowconfig.ResourceConfig{Id: resourceId, RootId: request.Execution.Id, FsPrefix: request.FsPrefix},
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

	// Apply the service setup
	if request.Service != nil {
		// TODO: Handle RestartPolicy: Always?
		if request.Service.RestartPolicy == "Never" {
			bundle.Job.Spec.BackoffLimit = common.Ptr(int32(0))
			bundle.Job.Spec.Template.Spec.RestartPolicy = "Never"
		} else {
			// TODO: Throw errors from the pod containers? Atm it will just end with "Success"...
			bundle.Job.Spec.BackoffLimit = nil
			bundle.Job.Spec.Template.Spec.RestartPolicy = "OnFailure"
		}
		if request.Service.ReadinessProbe != nil {
			bundle.Job.Spec.Template.Spec.Containers[0].ReadinessProbe = common.MapPtr(request.Service.ReadinessProbe, testworkflows.MapProbeAPIToKube)
		}
	}

	// Annotate the group ID
	if request.GroupId != "" {
		testworkflowprocessor.AnnotateGroupId(&bundle.Job, request.GroupId)
		for i := range bundle.ConfigMaps {
			testworkflowprocessor.AnnotateGroupId(&bundle.ConfigMaps[i], request.GroupId)
		}
		for i := range bundle.Secrets {
			testworkflowprocessor.AnnotateGroupId(&bundle.Secrets[i], request.GroupId)
		}
	}

	// Deploy required resources
	err = bundle.Deploy(context.Background(), w.clientSet, cfg.Worker.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy test workflow")
	}

	// Register namespace information in the cache
	w.registry.RegisterNamespace(cfg.Resource.Id, cfg.Worker.Namespace)

	return &ExecuteResult{
		Signature: stage.MapSignatureListToInternal(bundle.Signature),
		Namespace: bundle.Job.Namespace,
	}, nil
}

func (w *worker) Notifications(ctx context.Context, namespace, id string, opts NotificationsOptions) NotificationsWatcher {
	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err, recycle := w.registry.Connect(ctx, id, ResourceHints{
		Namespace:   namespace,
		ScheduledAt: opts.ScheduledAt,
		Signature:   opts.Signature,
	})
	watcher := newNotificationsWatcher()
	if errors.Is(err, testworkflowcontroller.ErrJobTimeout) {
		err = registry2.ErrResourceNotFound
	}
	if err != nil {
		watcher.close(err)
		return watcher
	}

	// Watch the resource
	watchCtx, watchCtxCancel := context.WithCancel(ctx)
	ch := ctrl.Watch(watchCtx, opts.NoFollow)
	go func() {
		defer func() {
			watchCtxCancel()
			recycle()
		}()
		for n := range ch {
			if n.Error != nil {
				watcher.close(n.Error)
				return
			}
			watcher.send(n.Value.ToInternal())
		}
		watcher.close(nil)
	}()
	return watcher
}

// TODO: Avoid multiple controller copies?
// TODO: Optimize
func (w *worker) StatusNotifications(ctx context.Context, namespace, id string, opts StatusNotificationsOptions) StatusNotificationsWatcher {
	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err, recycle := w.registry.Connect(ctx, id, ResourceHints{
		Namespace:   namespace,
		ScheduledAt: opts.ScheduledAt,
		Signature:   opts.Signature,
	})
	watcher := newStatusNotificationsWatcher()
	if errors.Is(err, testworkflowcontroller.ErrJobTimeout) {
		err = registry2.ErrResourceNotFound
	}
	if err != nil {
		watcher.close(err)
		return watcher
	}

	// Watch the resource
	watchCtx, watchCtxCancel := context.WithCancel(ctx)
	sig := stage.MapSignatureListToInternal(ctrl.Signature())
	ch := ctrl.Watch(watchCtx, opts.NoFollow)
	go func() {
		defer func() {
			watchCtxCancel()
			recycle()
		}()
		prevNodeName := ""
		prevStep := ""
		prevIp := ""
		prevStatus := testkube.QUEUED_TestWorkflowStatus
		prevStepStatus := testkube.QUEUED_TestWorkflowStepStatus
		prevReady := false
		for n := range ch {
			if n.Error != nil {
				watcher.close(n.Error)
				return
			}

			// Check the readiness

			nodeName, _ := ctrl.NodeName()
			podIp, _ := ctrl.PodIP()
			ready, _ := ctrl.ContainersReady()
			current := prevStep
			status := prevStatus
			stepStatus := prevStepStatus
			if n.Value.Result != nil {
				if n.Value.Result.Status != nil {
					status = *n.Value.Result.Status
				} else {
					status = testkube.QUEUED_TestWorkflowStatus
				}
				current = n.Value.Result.Current(sig)
				if current == "" {
					stepStatus = common.ResolvePtr(n.Value.Result.Initialization.Status, testkube.QUEUED_TestWorkflowStepStatus)
				} else {
					stepStatus = common.ResolvePtr(n.Value.Result.Steps[current].Status, testkube.QUEUED_TestWorkflowStepStatus)
				}
			}
			if current != prevStep || status != prevStatus || stepStatus != prevStepStatus {
				prevNodeName = nodeName
				prevIp = podIp
				prevReady = ready
				prevStatus = status
				prevStepStatus = stepStatus
				prevStep = current
				watcher.send(StatusNotification{
					Ref:      current,
					NodeName: nodeName,
					PodIp:    podIp,
					Ready:    ready,
					Result:   n.Value.Result,
				})
			} else if nodeName != prevNodeName || podIp != prevIp || ready != prevReady {
				prevNodeName = nodeName
				prevIp = podIp
				prevReady = ready
				prevStatus = status
				prevStepStatus = stepStatus
				prevStep = current
				watcher.send(StatusNotification{
					Ref:      current,
					NodeName: nodeName,
					PodIp:    podIp,
					Ready:    ready,
				})
			}
		}
		watcher.close(nil)
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
	namespaces := maps.Keys(w.config.Cluster.Namespaces)
	if len(options.Namespaces) > 0 {
		namespaces = slices.Filter(nil, namespaces, func(ns string) bool {
			return slices.Contains(options.Namespaces, ns)
		})
	}

	listOptions := metav1.ListOptions{
		Limit: 100000,
	}
	labelSelectors := make([]string, 0)
	if options.GroupId != "" {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", constants.GroupIdLabelName, options.GroupId))
	}
	if options.RootId != "" {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", constants.RootResourceIdLabelName, options.RootId))
	}

	// TODO: make concurrent calls
	list := make([]ListResultItem, 0)
	for _, ns := range namespaces {
		// TODO: retry?
		jobs, err := w.clientSet.BatchV1().Jobs(ns).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, job := range jobs.Items {
			if options.Finished != nil && *options.Finished != watchers.IsJobFinished(&job) {
				continue
			}
			if options.Root != nil && *options.Root != (job.Labels[constants.RootResourceIdLabelName] == job.Labels[constants.ResourceIdLabelName]) {
				continue
			}
			var cfg testworkflowconfig.InternalConfig
			err = json.Unmarshal([]byte(job.Spec.Template.Annotations[constants.InternalAnnotationName]), &cfg)
			if err != nil {
				log.DefaultLogger.Warnw("detected execution job that have invalid internal configuration", "name", job.Name, "namespace", job.Namespace, "error", err)
				continue
			}
			if options.OrganizationId != "" && options.OrganizationId != cfg.Execution.OrganizationId {
				continue
			}
			if options.EnvironmentId != "" && options.EnvironmentId != cfg.Execution.EnvironmentId {
				continue
			}
			list = append(list, ListResultItem{
				Execution: cfg.Execution,
				Workflow:  cfg.Workflow,
				Resource:  cfg.Resource,
				Namespace: job.Namespace,
			})
		}
	}
	return list, nil
}

func (w *worker) Destroy(ctx context.Context, namespace, id string) (err error) {
	if namespace == "" {
		namespace, err = w.registry.GetNamespace(ctx, id)
		if err != nil {
			return err
		}
	}
	// TODO: Move implementation there
	return testworkflowcontroller.Cleanup(ctx, w.clientSet, namespace, id)
}

func (w *worker) DestroyGroup(ctx context.Context, namespace, groupId string) error {
	if namespace != "" {
		return testworkflowcontroller.CleanupGroup(ctx, w.clientSet, namespace, groupId)
	}

	// Delete group resources in all known namespaces
	errs := make([]error, 0)
	for ns := range w.config.Cluster.Namespaces {
		err := w.Destroy(ctx, ns, groupId)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors2.Join(errs...)
}

func (w *worker) Pause(ctx context.Context, namespace, id string) (err error) {
	podIp, err := w.registry.GetPodIP(ctx, id)
	if err != nil {
		return err
	} else if podIp == "" {
		return registry2.ErrPodIpNotAssigned
	}

	// TODO: Move implementation there
	return testworkflowcontroller.Pause(ctx, podIp)
}

func (w *worker) Resume(ctx context.Context, namespace, id string) (err error) {
	podIp, err := w.registry.GetPodIP(ctx, id)
	if err != nil {
		return err
	} else if podIp == "" {
		return registry2.ErrPodIpNotAssigned
	}

	// TODO: Move implementation there
	return testworkflowcontroller.Resume(ctx, podIp)
}

// TODO: consider status channel (?)
func (w *worker) ResumeMany(ctx context.Context, ids []string) (errs []IdentifiableError) {
	ips := make(map[string]string, len(ids))

	// Try to obtain IPs
	// TODO: concurrent operations (or single list operation)
	for _, id := range ids {
		podIp, err := w.registry.GetPodIP(ctx, id)
		if err != nil {
			errs = append(errs, IdentifiableError{Id: id, Error: err})
		} else if podIp == "" {
			errs = append(errs, IdentifiableError{Id: id, Error: registry2.ErrPodIpNotAssigned})
		} else {
			ips[id] = podIp
		}
	}

	// Finish early when there are no IPs
	if len(ips) == 0 {
		return errs
	}

	// Initialize counters and synchronisation for waiting
	var wg sync.WaitGroup
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	counter := atomic.Int32{}
	ready := func() {
		v := counter.Add(1)
		if v < int32(len(ips)) {
			cond.Wait()
		} else {
			cond.Broadcast()
		}
	}

	// Create client connection and send to all of them
	wg.Add(len(ips))
	var errsMu sync.Mutex
	for id, podIp := range ips {
		go func(id, address string) {
			cond.L.Lock()
			defer cond.L.Unlock()

			client, err := control.NewClient(context.Background(), address, initconstants.ControlServerPort)
			ready()
			defer func() {
				if client != nil {
					client.Close()
				}
				wg.Done()
			}()

			// Fast-track: immediate success
			if err == nil {
				err = client.Resume()
				if err == nil {
					return
				}
				log.DefaultLogger.Warnw("failed to resume, retrying...", "id", id, "address", address, "error", err)
			}

			// Retrying mechanism
			for i := 0; i < 6; i++ {
				if client != nil {
					client.Close()
				}
				client, err = control.NewClient(context.Background(), address, initconstants.ControlServerPort)
				if err == nil {
					err = client.Resume()
					if err == nil {
						return
					}
				}
				log.DefaultLogger.Warnw("failed to resume, retrying...", "id", id, "address", address, "error", err)
				time.Sleep(ResumeRetryOnFailureDelay)
			}

			// Total failure while retrying
			log.DefaultLogger.Errorw("failed to resume, maximum retries reached.", "id", id, "address", address, "error", err)
			errsMu.Lock()
			errs = append(errs, IdentifiableError{Id: id, Error: err})
			errsMu.Unlock()
		}(id, podIp)
	}
	wg.Wait()

	return errs
}
