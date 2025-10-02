package kubernetesworker

import (
	"context"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings/slices"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/control"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/watchers"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/utils"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
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
	baseWorkerConfig testworkflowconfig.WorkerConfig
	config           Config
	registry         registry.ControllersRegistry
}

func NewWorker(clientSet kubernetes.Interface, processor testworkflowprocessor.Processor, config Config) *worker {
	namespaces := registry.NewNamespacesRegistry(clientSet, config.Cluster.DefaultNamespace, maps.Keys(config.Cluster.Namespaces), 50)
	return &worker{
		clientSet: clientSet,
		processor: processor,
		config:    config,
		registry:  registry.NewControllersRegistry(clientSet, namespaces, config.RunnerId, 50),
		baseWorkerConfig: testworkflowconfig.WorkerConfig{
			Namespace:                         config.Cluster.DefaultNamespace,
			DefaultRegistry:                   config.Cluster.DefaultRegistry,
			DefaultServiceAccount:             config.Cluster.Namespaces[config.Cluster.DefaultNamespace].DefaultServiceAccountName,
			ClusterID:                         config.Cluster.Id,
			RunnerID:                          config.RunnerId,
			InitImage:                         constants.DefaultInitImage,
			ToolkitImage:                      constants.DefaultToolkitImage,
			ImageInspectorPersistenceEnabled:  config.ImageInspector.CacheEnabled,
			ImageInspectorPersistenceCacheKey: config.ImageInspector.CacheKey,
			ImageInspectorPersistenceCacheTTL: config.ImageInspector.CacheTTL,
			Connection:                        config.Connection,
			FeatureFlags:                      config.FeatureFlags,
			CommonEnvVariables:                config.CommonEnvVariables,
			AllowLowSecurityFields:            config.AllowLowSecurityFields,
		},
	}
}

func (w *worker) buildInternalConfig(resourceId, fsPrefix string, execution testworkflowconfig.ExecutionConfig, controlPlane testworkflowconfig.ControlPlaneConfig, workflow testworkflowsv1.TestWorkflow, executionToken string) testworkflowconfig.InternalConfig {
	cfg := testworkflowconfig.InternalConfig{
		Execution:    execution,
		Workflow:     testworkflowconfig.WorkflowConfig{Name: workflow.Name, Labels: workflow.Labels},
		Resource:     testworkflowconfig.ResourceConfig{Id: resourceId, RootId: execution.Id, FsPrefix: fsPrefix},
		ControlPlane: controlPlane,
		Worker:       w.baseWorkerConfig,
	}
	if executionToken != "" {
		cfg.Worker.Connection.ApiKey = executionToken
	}
	if workflow.Spec.Job != nil && workflow.Spec.Job.Namespace != "" {
		cfg.Worker.Namespace = workflow.Spec.Job.Namespace
	}
	if ns, ok := w.config.Cluster.Namespaces[cfg.Worker.Namespace]; ok && ns.DefaultServiceAccountName != "" {
		cfg.Worker.DefaultServiceAccount = ns.DefaultServiceAccountName
	}
	return cfg
}

func (w *worker) buildSecrets(maps map[string]map[string]string) []corev1.Secret {
	secrets := make([]corev1.Secret, 0, len(maps))
	for name, stringData := range maps {
		secrets = append(secrets, corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			StringData: stringData,
		})
	}
	return secrets
}

func (w *worker) Execute(ctx context.Context, request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	// Process the data
	resourceId := request.ResourceId
	if resourceId == "" {
		resourceId = request.Execution.Id
	}
	scheduledAt := time.Now()
	if request.ScheduledAt != nil {
		scheduledAt = *request.ScheduledAt
	} else if resourceId == request.Execution.Id && !request.Execution.ScheduledAt.IsZero() {
		scheduledAt = request.Execution.ScheduledAt
	}
	cfg := w.buildInternalConfig(resourceId, request.ArtifactsPathPrefix, request.Execution, request.ControlPlane, request.Workflow, request.Token)
	secrets := w.buildSecrets(request.Secrets)

	// Ensure the execution namespace is allowed
	if _, ok := w.config.Cluster.Namespaces[cfg.Worker.Namespace]; !ok {
		return nil, errors.New(fmt.Sprintf("namespace %s not supported", cfg.Worker.Namespace))
	}

	// Configure default service account
	if request.Workflow.Spec.Pod == nil {
		request.Workflow.Spec.Pod = &testworkflowsv1.PodConfig{
			ServiceAccountName: cfg.Worker.DefaultServiceAccount,
		}
	} else if request.Workflow.Spec.Pod.ServiceAccountName == "" {
		request.Workflow.Spec.Pod = request.Workflow.Spec.Pod.DeepCopy()
		request.Workflow.Spec.Pod.ServiceAccountName = cfg.Worker.DefaultServiceAccount
	}

	var runtimeOptions *testworkflowprocessor.RuntimeOptions
	if request.Runtime != nil && len(request.Runtime.Variables) > 0 {
		runtimeOptions = &testworkflowprocessor.RuntimeOptions{
			Variables: request.Runtime.Variables,
		}
	}

	// Process the Test Workflow
	bundle, err := w.processor.Bundle(ctx, &request.Workflow, testworkflowprocessor.BundleOptions{
		Config:                 cfg,
		Secrets:                secrets,
		ScheduledAt:            scheduledAt,
		CommonEnvVariables:     w.baseWorkerConfig.CommonEnvVariables,
		AllowLowSecurityFields: w.baseWorkerConfig.AllowLowSecurityFields,
		Runtime:                runtimeOptions,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to process test workflow")
	}

	// Annotate the group ID
	if request.GroupId != "" {
		bundle.SetGroupId(request.GroupId)
	}

	// Annotate the runner ID
	if w.config.RunnerId != "" {
		bundle.SetRunnerId(w.config.RunnerId)
	}

	// Register namespace information in the cache
	w.registry.RegisterNamespace(cfg.Resource.Id, cfg.Worker.Namespace)

	// Deploy required resources
	err = bundle.Deploy(context.Background(), w.clientSet, cfg.Worker.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy test workflow")
	}

	return &executionworkertypes.ExecuteResult{
		Signature:   stage.MapSignatureListToInternal(bundle.Signature),
		ScheduledAt: scheduledAt,
		Namespace:   bundle.Job.Namespace,
	}, nil
}

func (w *worker) Service(ctx context.Context, request executionworkertypes.ServiceRequest) (*executionworkertypes.ServiceResult, error) {
	// Process the data
	resourceId := request.ResourceId
	if resourceId == "" {
		resourceId = request.Execution.Id
	}
	scheduledAt := time.Now()
	if request.ScheduledAt != nil {
		scheduledAt = *request.ScheduledAt
	} else if resourceId == request.Execution.Id && !request.Execution.ScheduledAt.IsZero() {
		scheduledAt = request.Execution.ScheduledAt
	}
	cfg := w.buildInternalConfig(resourceId, "", request.Execution, request.ControlPlane, request.Workflow, request.Token)
	secrets := w.buildSecrets(request.Secrets)

	// Ensure the execution namespace is allowed
	if _, ok := w.config.Cluster.Namespaces[cfg.Worker.Namespace]; !ok {
		return nil, errors.New(fmt.Sprintf("namespace %s not supported", cfg.Worker.Namespace))
	}

	var runtimeOptions *testworkflowprocessor.RuntimeOptions
	if request.Runtime != nil && len(request.Runtime.Variables) > 0 {
		runtimeOptions = &testworkflowprocessor.RuntimeOptions{
			Variables: request.Runtime.Variables,
		}
	}

	// Process the Test Workflow
	bundle, err := w.processor.Bundle(ctx, &request.Workflow, testworkflowprocessor.BundleOptions{
		Config:                 cfg,
		Secrets:                secrets,
		ScheduledAt:            scheduledAt,
		CommonEnvVariables:     w.baseWorkerConfig.CommonEnvVariables,
		AllowLowSecurityFields: w.baseWorkerConfig.AllowLowSecurityFields,
		Runtime:                runtimeOptions,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to process test workflow")
	}

	// Apply the service setup
	// TODO: Handle RestartPolicy: Always?
	if request.RestartPolicy == "Never" {
		bundle.Job.Spec.BackoffLimit = common.Ptr(int32(0))
		bundle.Job.Spec.Template.Spec.RestartPolicy = "Never"
	} else {
		// TODO: Throw errors from the pod containers? Atm it will just end with "Success"...
		bundle.Job.Spec.BackoffLimit = nil
		bundle.Job.Spec.Template.Spec.RestartPolicy = "OnFailure"
	}
	if request.ReadinessProbe != nil {
		bundle.Job.Spec.Template.Spec.Containers[0].ReadinessProbe = common.MapPtr(request.ReadinessProbe, testworkflows.MapProbeAPIToKube)
	}

	// Annotate the group ID
	if request.GroupId != "" {
		bundle.SetGroupId(request.GroupId)
	}

	// Register namespace information in the cache
	w.registry.RegisterNamespace(cfg.Resource.Id, cfg.Worker.Namespace)

	// Deploy required resources
	err = bundle.Deploy(context.Background(), w.clientSet, cfg.Worker.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deploy test workflow")
	}

	return &executionworkertypes.ServiceResult{
		Signature:   stage.MapSignatureListToInternal(bundle.Signature),
		ScheduledAt: scheduledAt,
		Namespace:   bundle.Job.Namespace,
	}, nil
}

func (w *worker) Notifications(ctx context.Context, id string, opts executionworkertypes.NotificationsOptions) executionworkertypes.NotificationsWatcher {
	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err, recycle := w.registry.Connect(ctx, id, opts.Hints)
	watcher := executionworkertypes.NewNotificationsWatcher()
	if errors.Is(err, controller.ErrJobTimeout) {
		err = registry.ErrResourceNotFound
	}
	if err != nil {
		watcher.Close(err)
		return watcher
	}

	// Watch the resource
	watchCtx, watchCtxCancel := context.WithCancel(ctx)
	ch := ctrl.Watch(watchCtx, opts.NoFollow, w.config.LogAbortedDetails)
	go func() {
		defer func() {
			watchCtxCancel()
			recycle()
		}()
		for n := range ch {
			if n.Error != nil {
				watcher.Close(n.Error)
				return
			}
			watcher.Send(common.Ptr(n.Value.ToInternal()))
		}
		watcher.Close(nil)
	}()
	return watcher
}

// TODO: Avoid multiple controller copies?
// TODO: Optimize
func (w *worker) StatusNotifications(ctx context.Context, id string, opts executionworkertypes.StatusNotificationsOptions) executionworkertypes.StatusNotificationsWatcher {
	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err, recycle := w.registry.Connect(ctx, id, opts.Hints)
	watcher := executionworkertypes.NewStatusNotificationsWatcher()
	if errors.Is(err, controller.ErrJobTimeout) {
		err = registry.ErrResourceNotFound
	}
	if err != nil {
		watcher.Close(err)
		return watcher
	}

	// Watch the resource
	watchCtx, watchCtxCancel := context.WithCancel(ctx)
	sig := stage.MapSignatureListToInternal(ctrl.Signature())
	ch := ctrl.Watch(watchCtx, opts.NoFollow, w.config.LogAbortedDetails)
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
				watcher.Close(n.Error)
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
				watcher.Send(executionworkertypes.StatusNotification{
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
				watcher.Send(executionworkertypes.StatusNotification{
					Ref:      current,
					NodeName: nodeName,
					PodIp:    podIp,
					Ready:    ready,
				})
			}
		}
		watcher.Close(nil)
	}()
	return watcher
}

// TODO: Optimize?
// TODO: Allow fetching temporary logs too?
func (w *worker) Logs(ctx context.Context, id string, options executionworkertypes.LogsOptions) utils.LogsReader {
	reader := utils.NewLogsReader()
	notifications := w.Notifications(ctx, id, executionworkertypes.NotificationsOptions{
		Hints:    options.Hints,
		NoFollow: options.NoFollow,
	})
	if notifications.Err() != nil {
		reader.End(notifications.Err())
		return reader
	}

	go func() {
		defer reader.Close()
		ref := ""
		for v := range notifications.Channel() {
			if v.Log != "" {
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

func (w *worker) Get(ctx context.Context, id string, options executionworkertypes.GetOptions) (*executionworkertypes.GetResult, error) {
	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err, recycle := w.registry.Connect(ctx, id, options.Hints)
	if err != nil {
		return nil, err
	}
	defer recycle()

	cfg, err := ctrl.InternalConfig()
	if err != nil {
		return nil, err
	}

	result, err := ctrl.EstimatedResult(ctx)
	if err != nil {
		log.DefaultLogger.Warnw("failed to estimate result", "id", id, "error", err)
		result = &testkube.TestWorkflowResult{}
	}

	for notification := range ctrl.Watch(ctx, true, false) {
		if notification.Error != nil {
			continue
		}
		if notification.Value.Result != nil {
			result = notification.Value.Result
		}
	}

	return &executionworkertypes.GetResult{
		Execution: cfg.Execution,
		Workflow:  cfg.Workflow,
		Resource:  cfg.Resource,
		Signature: stage.MapSignatureListToInternal(ctrl.Signature()),
		Result:    *result,
		Namespace: ctrl.Namespace(),
	}, nil
}

func (w *worker) Summary(ctx context.Context, id string, options executionworkertypes.GetOptions) (*executionworkertypes.SummaryResult, error) {
	// Connect to the resource
	// TODO: Move the implementation directly there
	ctrl, err, recycle := w.registry.Connect(ctx, id, options.Hints)
	if err != nil {
		return nil, err
	}
	defer recycle()

	cfg, err := ctrl.InternalConfig()
	if err != nil {
		return nil, err
	}

	estimatedResult, err := ctrl.EstimatedResult(ctx)
	if err != nil {
		log.DefaultLogger.Warnw("failed to estimate result", "id", id, "error", err)
		estimatedResult = &testkube.TestWorkflowResult{}
	}

	return &executionworkertypes.SummaryResult{
		Execution:       cfg.Execution,
		Workflow:        cfg.Workflow,
		Resource:        cfg.Resource,
		Signature:       stage.MapSignatureListToInternal(ctrl.Signature()),
		EstimatedResult: *estimatedResult,
		Namespace:       ctrl.Namespace(),
	}, nil
}

func (w *worker) Finished(ctx context.Context, id string, options executionworkertypes.GetOptions) (bool, error) {
	panic("not implemented")
}

func (w *worker) List(ctx context.Context, options executionworkertypes.ListOptions) ([]executionworkertypes.ListResultItem, error) {
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
	listOptions.LabelSelector = strings.Join(labelSelectors, ",")

	// TODO: make concurrent calls
	list := make([]executionworkertypes.ListResultItem, 0)
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
			list = append(list, executionworkertypes.ListResultItem{
				Execution: cfg.Execution,
				Workflow:  cfg.Workflow,
				Resource:  cfg.Resource,
				Namespace: job.Namespace,
			})
		}
	}
	return list, nil
}

func (w *worker) Abort(ctx context.Context, id string, options executionworkertypes.DestroyOptions) (err error) {
	if options.Namespace == "" {
		options.Namespace, err = w.registry.GetNamespace(ctx, id)
		if err != nil {
			return err
		}
	}
	if err := w.patchTerminationAnnotations(ctx, id, options.Namespace, testkube.ABORTED_TestWorkflowStatus, "Job has been aborted by the system"); err != nil {
		return errors.Wrapf(err, "failed to patch job %s/%s with termination code & reason", options.Namespace, id)
	}
	// It may safely destroy all the resources - the trace should be still readable.
	return w.Destroy(ctx, id, options)
}

func (w *worker) Cancel(ctx context.Context, id string, options executionworkertypes.DestroyOptions) (err error) {
	if options.Namespace == "" {
		options.Namespace, err = w.registry.GetNamespace(ctx, id)
		if err != nil {
			return err
		}
	}
	if err := w.patchTerminationAnnotations(ctx, id, options.Namespace, testkube.CANCELED_TestWorkflowStatus, "Job has been canceled by a user"); err != nil {
		return errors.Wrapf(err, "failed to patch job %s/%s with termination code & reason", options.Namespace, id)
	}
	return w.Destroy(ctx, id, options)
}

func (w *worker) patchTerminationAnnotations(ctx context.Context, id string, namespace string, status testkube.TestWorkflowStatus, reason string) error {
	patch := map[string]interface{}{
		"metadata": map[string]any{
			"annotations": map[string]string{
				constants.AnnotationTerminationCode:   string(status),
				constants.AnnotationTerminationReason: reason,
			},
		},
	}
	patchBytes, _ := json.Marshal(patch)
	_, err := w.clientSet.BatchV1().Jobs(namespace).Patch(ctx, id, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func (w *worker) Destroy(ctx context.Context, id string, options executionworkertypes.DestroyOptions) (err error) {
	if options.Namespace == "" {
		options.Namespace, err = w.registry.GetNamespace(ctx, id)
		if err != nil {
			return err
		}
	}
	// TODO: Move implementation there
	return controller.Cleanup(ctx, w.clientSet, options.Namespace, id)
}

func (w *worker) DestroyGroup(ctx context.Context, groupId string, options executionworkertypes.DestroyOptions) error {
	if options.Namespace != "" {
		return controller.CleanupGroup(ctx, w.clientSet, options.Namespace, groupId)
	}

	// Delete group resources in all known namespaces
	errs := make([]error, 0)
	for ns := range w.config.Cluster.Namespaces {
		err := w.DestroyGroup(ctx, groupId, executionworkertypes.DestroyOptions{Namespace: ns})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors2.Join(errs...)
}

func (w *worker) Pause(ctx context.Context, id string, options executionworkertypes.ControlOptions) (err error) {
	podIp, err := w.registry.GetPodIP(ctx, id)
	if err != nil {
		return err
	} else if podIp == "" {
		return registry.ErrPodIpNotAssigned
	}

	// TODO: Move implementation there
	return controller.Pause(ctx, podIp)
}

func (w *worker) Resume(ctx context.Context, id string, options executionworkertypes.ControlOptions) (err error) {
	podIp, err := w.registry.GetPodIP(ctx, id)
	if err != nil {
		return err
	} else if podIp == "" {
		return registry.ErrPodIpNotAssigned
	}

	// TODO: Move implementation there
	return controller.Resume(ctx, podIp)
}

// TODO: consider status channel (?)
func (w *worker) ResumeMany(ctx context.Context, ids []string, options executionworkertypes.ControlOptions) (errs []executionworkertypes.IdentifiableError) {
	ips := make(map[string]string, len(ids))

	// Try to obtain IPs
	// TODO: concurrent operations (or single list operation)
	for _, id := range ids {
		podIp, err := w.registry.GetPodIP(ctx, id)
		if err != nil {
			errs = append(errs, executionworkertypes.IdentifiableError{Id: id, Error: err})
		} else if podIp == "" {
			errs = append(errs, executionworkertypes.IdentifiableError{Id: id, Error: registry.ErrPodIpNotAssigned})
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
			errs = append(errs, executionworkertypes.IdentifiableError{Id: id, Error: err})
			errsMu.Unlock()
		}(id, podIp)
	}
	wg.Wait()

	return errs
}
