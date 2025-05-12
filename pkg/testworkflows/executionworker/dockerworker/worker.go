package dockerworker

import (
	"context"
	errors2 "errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	volume2 "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	initconstants "github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/instructions"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	controller2 "github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/utils"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

type worker struct {
	processor        testworkflowprocessor.Processor
	baseWorkerConfig testworkflowconfig.WorkerConfig
	config           Config
	client           *client.Client
}

func NewWorker(client *client.Client, processor testworkflowprocessor.Processor, config Config) *worker {
	return &worker{
		client:    client,
		processor: processor,
		config:    config,
		baseWorkerConfig: testworkflowconfig.WorkerConfig{
			InitImage:                         constants.DefaultInitImage,
			ToolkitImage:                      constants.DefaultToolkitImage,
			ImageInspectorPersistenceEnabled:  config.ImageInspector.CacheEnabled,
			ImageInspectorPersistenceCacheKey: config.ImageInspector.CacheKey,
			ImageInspectorPersistenceCacheTTL: config.ImageInspector.CacheTTL,
			Connection:                        config.Connection,
		},
	}
}

func (w *worker) buildInternalConfig(resourceId, fsPrefix string, execution testworkflowconfig.ExecutionConfig, controlPlane testworkflowconfig.ControlPlaneConfig, workflow testworkflowsv1.TestWorkflow) testworkflowconfig.InternalConfig {
	cfg := testworkflowconfig.InternalConfig{
		Execution:    execution,
		Workflow:     testworkflowconfig.WorkflowConfig{Name: workflow.Name, Labels: workflow.Labels},
		Resource:     testworkflowconfig.ResourceConfig{Id: resourceId, RootId: execution.Id, FsPrefix: fsPrefix},
		ControlPlane: controlPlane,
		Worker:       w.baseWorkerConfig,
	}
	if workflow.Spec.Job != nil && workflow.Spec.Job.Namespace != "" {
		cfg.Worker.Namespace = workflow.Spec.Job.Namespace
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
	cfg := w.buildInternalConfig(resourceId, request.ArtifactsPathPrefix, request.Execution, request.ControlPlane, request.Workflow)
	secrets := w.buildSecrets(request.Secrets)

	// Process the Test Workflow
	bundle, err := w.processor.Bundle(ctx, &request.Workflow, testworkflowprocessor.BundleOptions{Config: cfg, Secrets: secrets, ScheduledAt: scheduledAt})
	if err != nil {
		return nil, errors.Wrap(err, "failed to process test workflow")
	}

	// Annotate the group ID
	if request.GroupId != "" {
		bundle.SetGroupId(request.GroupId)
	}

	//// Determine FS Group
	//opts := make([]string, 0)
	//if bundle.Job.Spec.Template.Spec.SecurityContext != nil {
	//	if bundle.Job.Spec.Template.Spec.SecurityContext.FSGroup != nil {
	//		opts = append(opts, fmt.Sprintf("gid=%d", *bundle.Job.Spec.Template.Spec.SecurityContext.FSGroup))
	//	}
	//}

	// List all the expected containers
	containers := append(bundle.Job.Spec.Template.Spec.InitContainers, bundle.Job.Spec.Template.Spec.Containers...)

	// Validate volumes
	for _, volume := range bundle.Job.Spec.Template.Spec.Volumes {
		if volume.EmptyDir == nil {
			// TODO: cleanup
			return nil, errors.New("only emptyDir volumes are allowed")
		}
	}

	// Deploy the volumes
	wg := sync.WaitGroup{}
	wg.Add(len(bundle.Job.Spec.Template.Spec.Volumes))
	errs, errsMu := make([]error, 0), sync.Mutex{}
	for _, volume := range bundle.Job.Spec.Template.Spec.Volumes {
		go func() {
			defer wg.Done()
			_, err = w.client.VolumeCreate(context.Background(), volume2.CreateOptions{
				Labels: map[string]string{
					constants.RootResourceIdLabelName: resourceId,
					constants.ResourceIdLabelName:     resourceId,
				},
				Driver: "local",
				Name:   fmt.Sprintf("%s-%s", resourceId, volume.Name),
			})
			if err != nil {
				errsMu.Lock()
				defer errsMu.Unlock()
				errs = append(errs, err)
			}
		}()
	}
	wg.Wait()
	if len(errs) > 0 {
		// TODO: cleanup
		return nil, errors.Wrap(errors2.Join(errs...), "failed to create docker volumes")
	}

	// Deploy the containers
	wg = sync.WaitGroup{}
	wg.Add(len(containers))
	errs, errsMu = make([]error, 0), sync.Mutex{}
	containerIds := make([]string, len(containers))
	// TODO: Consider lazy creating the containers
	for index, cn := range containers {
		go func() {
			defer wg.Done()
			//user := ""
			//if cn.SecurityContext.RunAsUser != nil {
			//	if cn.SecurityContext.RunAsGroup != nil {
			//		user = fmt.Sprintf("%d:%d", *cn.SecurityContext.RunAsUser, *cn.SecurityContext.RunAsGroup)
			//	} else {
			//		user = fmt.Sprintf("%d", *cn.SecurityContext.RunAsUser)
			//	}
			//}
			user := "root" // TODO: add another start 'root' container that will set proper permissions for the volumes

			// Pull the image if necessary
			shouldPullImage := cn.ImagePullPolicy != "Never"
			if cn.ImagePullPolicy == "IfNotPresent" {
				_, _, err := w.client.ImageInspectWithRaw(context.Background(), cn.Image)
				shouldPullImage = err != nil
			}
			if shouldPullImage {
				pullReader, err := w.client.ImagePull(context.Background(), cn.Image, image.PullOptions{})
				if err != nil {
					errsMu.Lock()
					errs = append(errs, errors.Wrapf(err, "failed to pull image: %s", cn.Image))
					errsMu.Unlock()
					return
				}
				_, _ = io.Copy(os.Stdout, pullReader)
			}

			mounts := make([]mount.Mount, len(cn.VolumeMounts))
			for i := range cn.VolumeMounts {
				mounts[i] = mount.Mount{
					Type:     mount.TypeVolume,
					Source:   fmt.Sprintf("%s-%s", resourceId, cn.VolumeMounts[i].Name),
					Target:   cn.VolumeMounts[i].MountPath,
					ReadOnly: cn.VolumeMounts[i].ReadOnly,
				}
				if cn.VolumeMounts[i].SubPath != "" {
					mounts[i].VolumeOptions = &mount.VolumeOptions{
						Subpath: cn.VolumeMounts[i].SubPath,
					}
				}
			}
			envs := make([]string, len(cn.Env))
			for i := range cn.Env {
				if cn.Env[i].ValueFrom != nil {
					if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == constants.InternalAnnotationFieldPath {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, bundle.Job.Spec.Template.Annotations[constants.InternalAnnotationName])
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == constants.SpecAnnotationFieldPath {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, bundle.Job.Spec.Template.Annotations[constants.SpecAnnotationName])
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == constants.SignatureAnnotationFieldPath {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, bundle.Job.Spec.Template.Annotations[constants.SignatureAnnotationName])
					} else if cn.Env[i].ValueFrom.ResourceFieldRef != nil && cn.Env[i].ValueFrom.ResourceFieldRef.Resource == "requests.cpu" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "") // TODO
					} else if cn.Env[i].ValueFrom.ResourceFieldRef != nil && cn.Env[i].ValueFrom.ResourceFieldRef.Resource == "limits.cpu" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "") // TODO
					} else if cn.Env[i].ValueFrom.ResourceFieldRef != nil && cn.Env[i].ValueFrom.ResourceFieldRef.Resource == "requests.memory" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "") // TODO
					} else if cn.Env[i].ValueFrom.ResourceFieldRef != nil && cn.Env[i].ValueFrom.ResourceFieldRef.Resource == "limits.memory" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "") // TODO
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == "spec.nodeName" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "docker-node") // TODO?
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == "metadata.name" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, resourceId)
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == "metadata.namespace" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "")
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == "spec.serviceAccountName" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "")
					} else if cn.Env[i].ValueFrom.FieldRef != nil && cn.Env[i].ValueFrom.FieldRef.FieldPath == "status.podIP" {
						envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, "") // TODO
					} else {
						errsMu.Lock()
						errs = append(errs, fmt.Errorf("cannot use environment variables from external place: %s", cn.Env[i]))
						errsMu.Unlock()
						return
					}
				} else {
					envs[i] = fmt.Sprintf("%s=%s", cn.Env[i].Name, cn.Env[i].Value)
				}
			}
			containerName := fmt.Sprintf("%s-%d", resourceId, index+1)
			res, err := w.client.ContainerCreate(context.Background(), &container.Config{
				User:            user,
				AttachStdout:    false, // ?
				AttachStderr:    false, // ?
				ExposedPorts:    nil,   // ?
				Env:             envs,
				Entrypoint:      cn.Command,
				Cmd:             cn.Args,
				Healthcheck:     nil,   // ?
				ArgsEscaped:     false, // ?
				Image:           cn.Image,
				WorkingDir:      cn.WorkingDir,
				NetworkDisabled: false,
				OnBuild:         nil, // ?
				Labels: map[string]string{ // ?
					constants.RootResourceIdLabelName:   resourceId,
					constants.ResourceIdLabelName:       resourceId,
					constants.ScheduledAtAnnotationName: bundle.Job.Spec.Template.Annotations[constants.ScheduledAtAnnotationName],
					constants.InternalAnnotationName:    bundle.Job.Spec.Template.Annotations[constants.InternalAnnotationName],
					constants.SignatureAnnotationName:   bundle.Job.Spec.Template.Annotations[constants.SignatureAnnotationName],
					constants.SpecAnnotationName:        bundle.Job.Spec.Template.Annotations[constants.SpecAnnotationName],
				},
				StopTimeout: nil, // ?
			}, &container.HostConfig{
				Mounts: mounts,
			}, &network.NetworkingConfig{}, &v1.Platform{}, containerName)
			if err != nil {
				errsMu.Lock()
				errs = append(errs, errors.Wrap(err, "failed to create docker container"))
				errsMu.Unlock()
				return
			}
			if len(res.Warnings) > 0 {
				log.DefaultLogger.Warnw("created docker container with warnings", "containerId", res.ID, "warnings", res.Warnings)
			}
			containerIds[index] = res.ID
		}()
	}
	wg.Wait()
	if len(errs) > 0 {
		// TODO: cleanup
		return nil, errors.Wrap(errors2.Join(errs...), "failed to create containers")
	}

	go w.orchestrate(context.Background(), resourceId, scheduledAt, stage.MapSignatureListToInternal(bundle.Signature))

	return &executionworkertypes.ExecuteResult{
		Signature:   stage.MapSignatureListToInternal(bundle.Signature),
		ScheduledAt: scheduledAt,
		Namespace:   bundle.Job.Namespace,
	}, nil
}

// TODO: handle correctly reattaching when some containers have not been created before
func (w *worker) orchestrate(ctx context.Context, resourceId string, scheduledAt time.Time, signature []testkube.TestWorkflowSignature) {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()
	ctrl, err := NewController(ctx, w.client, resourceId, scheduledAt, ControllerOptions{
		Signature: stage.MapSignatureList(signature),
		RunnerId:  w.config.Connection.AgentID,
	})
	if err != nil {
		log.DefaultLogger.Errorw("failed to create controller", "error", err)
		return
	}

	updatesCh := ctrl.watcher.Updated(ctx)
	for ok := true; ok; _, ok = <-updatesCh {
		state := ctrl.watcher.State().(*executionState)

		// Abandon if failed
		if state.ExecutionError() != "" {
			return
		}

		// Check if another one needs to be started
		for i := 0; i < len(state.containers); i++ {
			prevName := fmt.Sprintf("%d", i)
			name := fmt.Sprintf("%d", i+1)
			prevFinished := prevName == "0" || state.ContainerFinished(prevName)
			currentStarted := state.Completed() || state.ContainerStarted(name) || state.ContainerFinished(name)
			if prevFinished && !currentStarted {
				fullContainerName := fmt.Sprintf("%s-%s", resourceId, name)
				err := w.client.ContainerStart(context.Background(), fullContainerName, container.StartOptions{})
				if err != nil {
					log.DefaultLogger.Errorw("failed to start docker container", "error", err)
					return
				}
			}
		}
	}
}

func (w *worker) Service(ctx context.Context, request executionworkertypes.ServiceRequest) (*executionworkertypes.ServiceResult, error) {
	panic("not implemented")
	//// Process the data
	//resourceId := request.ResourceId
	//if resourceId == "" {
	//	resourceId = request.Execution.Id
	//}
	//scheduledAt := time.Now()
	//if request.ScheduledAt != nil {
	//	scheduledAt = *request.ScheduledAt
	//} else if resourceId == request.Execution.Id && !request.Execution.ScheduledAt.IsZero() {
	//	scheduledAt = request.Execution.ScheduledAt
	//}
	//cfg := w.buildInternalConfig(resourceId, "", request.Execution, request.ControlPlane, request.Workflow)
	//secrets := w.buildSecrets(request.Secrets)
	//
	//// Process the Test Workflow
	//bundle, err := w.processor.Bundle(ctx, &request.Workflow, testworkflowprocessor.BundleOptions{Config: cfg, Secrets: secrets, ScheduledAt: scheduledAt})
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to process test workflow")
	//}
	//
	//// Apply the service setup
	//// TODO: Handle RestartPolicy: Always?
	//if request.RestartPolicy == "Never" {
	//	bundle.Job.Spec.BackoffLimit = common.Ptr(int32(0))
	//	bundle.Job.Spec.Template.Spec.RestartPolicy = "Never"
	//} else {
	//	// TODO: Throw errors from the pod containers? Atm it will just end with "Success"...
	//	bundle.Job.Spec.BackoffLimit = nil
	//	bundle.Job.Spec.Template.Spec.RestartPolicy = "OnFailure"
	//}
	//if request.ReadinessProbe != nil {
	//	bundle.Job.Spec.Template.Spec.Containers[0].ReadinessProbe = common.MapPtr(request.ReadinessProbe, testworkflows.MapProbeAPIToKube)
	//}
	//
	//// Annotate the group ID
	//if request.GroupId != "" {
	//	bundle.SetGroupId(request.GroupId)
	//}
	//
	//// Register namespace information in the cache
	//w.registry.RegisterNamespace(cfg.Resource.Id, cfg.Worker.Namespace)
	//
	////// Deploy required resources
	////err = bundle.Deploy(context.Background(), w.clientSet, cfg.Worker.Namespace)
	////if err != nil {
	////	return nil, errors.Wrap(err, "failed to deploy test workflow")
	////}
	//
	//return &executionworkertypes.ServiceResult{
	//	Signature:   stage.MapSignatureListToInternal(bundle.Signature),
	//	ScheduledAt: scheduledAt,
	//	Namespace:   bundle.Job.Namespace,
	//}, nil
}

func (w *worker) Notifications(ctx context.Context, id string, opts executionworkertypes.NotificationsOptions) executionworkertypes.NotificationsWatcher {
	// Connect to the resource
	// TODO: Move the implementation directly there
	scheduledAt := time.Time{}
	if opts.Hints.ScheduledAt != nil {
		scheduledAt = *opts.Hints.ScheduledAt
	}
	ctx, ctxCancel := context.WithCancel(ctx)
	ctrl, err := NewController(ctx, w.client, id, scheduledAt, ControllerOptions{
		Signature: stage.MapSignatureList(opts.Hints.Signature),
		RunnerId:  w.config.Connection.AgentID,
	})
	watcher := executionworkertypes.NewNotificationsWatcher()
	if errors.Is(err, controller2.ErrJobTimeout) {
		err = registry.ErrResourceNotFound
	}
	if err != nil {
		watcher.Close(err)
		ctxCancel()
		return watcher
	}

	// Watch the resource
	ch := ctrl.Watch(ctx, opts.NoFollow, w.config.LogAbortedDetails)
	go func() {
		defer func() {
			ctxCancel()
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
	panic("not implemented")
	//// Connect to the resource
	//// TODO: Move the implementation directly there
	//ctrl, err, recycle := w.registry.Connect(ctx, id, opts.Hints)
	//watcher := executionworkertypes.NewStatusNotificationsWatcher()
	//if errors.Is(err, controller.ErrJobTimeout) {
	//	err = registry.ErrResourceNotFound
	//}
	//if err != nil {
	//	watcher.Close(err)
	//	return watcher
	//}
	//
	//// Watch the resource
	//watchCtx, watchCtxCancel := context.WithCancel(ctx)
	//sig := stage.MapSignatureListToInternal(ctrl.Signature())
	//ch := ctrl.Watch(watchCtx, opts.NoFollow, w.config.LogAbortedDetails)
	//go func() {
	//	defer func() {
	//		watchCtxCancel()
	//		recycle()
	//	}()
	//	prevNodeName := ""
	//	prevStep := ""
	//	prevIp := ""
	//	prevStatus := testkube.QUEUED_TestWorkflowStatus
	//	prevStepStatus := testkube.QUEUED_TestWorkflowStepStatus
	//	prevReady := false
	//	for n := range ch {
	//		if n.Error != nil {
	//			watcher.Close(n.Error)
	//			return
	//		}
	//
	//		// Check the readiness
	//
	//		nodeName, _ := ctrl.NodeName()
	//		podIp, _ := ctrl.PodIP()
	//		ready, _ := ctrl.ContainersReady()
	//		current := prevStep
	//		status := prevStatus
	//		stepStatus := prevStepStatus
	//		if n.Value.Result != nil {
	//			if n.Value.Result.Status != nil {
	//				status = *n.Value.Result.Status
	//			} else {
	//				status = testkube.QUEUED_TestWorkflowStatus
	//			}
	//			current = n.Value.Result.Current(sig)
	//			if current == "" {
	//				stepStatus = common.ResolvePtr(n.Value.Result.Initialization.Status, testkube.QUEUED_TestWorkflowStepStatus)
	//			} else {
	//				stepStatus = common.ResolvePtr(n.Value.Result.Steps[current].Status, testkube.QUEUED_TestWorkflowStepStatus)
	//			}
	//		}
	//		if current != prevStep || status != prevStatus || stepStatus != prevStepStatus {
	//			prevNodeName = nodeName
	//			prevIp = podIp
	//			prevReady = ready
	//			prevStatus = status
	//			prevStepStatus = stepStatus
	//			prevStep = current
	//			watcher.Send(executionworkertypes.StatusNotification{
	//				Ref:      current,
	//				NodeName: nodeName,
	//				PodIp:    podIp,
	//				Ready:    ready,
	//				Result:   n.Value.Result,
	//			})
	//		} else if nodeName != prevNodeName || podIp != prevIp || ready != prevReady {
	//			prevNodeName = nodeName
	//			prevIp = podIp
	//			prevReady = ready
	//			prevStatus = status
	//			prevStepStatus = stepStatus
	//			prevStep = current
	//			watcher.Send(executionworkertypes.StatusNotification{
	//				Ref:      current,
	//				NodeName: nodeName,
	//				PodIp:    podIp,
	//				Ready:    ready,
	//			})
	//		}
	//	}
	//	watcher.Close(nil)
	//}()
	//return watcher
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

func (w *worker) Get(ctx context.Context, id string, opts executionworkertypes.GetOptions) (*executionworkertypes.GetResult, error) {
	// Connect to the resource
	// TODO: Move the implementation directly there
	scheduledAt := time.Time{}
	if opts.Hints.ScheduledAt != nil {
		scheduledAt = *opts.Hints.ScheduledAt
	}
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()
	ctrl, err := NewController(ctx, w.client, id, scheduledAt, ControllerOptions{
		Signature: stage.MapSignatureList(opts.Hints.Signature),
		RunnerId:  w.config.Connection.AgentID,
	})
	if err != nil {
		return nil, err
	}

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

func (w *worker) Summary(ctx context.Context, id string, opts executionworkertypes.GetOptions) (*executionworkertypes.SummaryResult, error) {
	// Connect to the resource
	// TODO: Move the implementation directly there
	scheduledAt := time.Time{}
	if opts.Hints.ScheduledAt != nil {
		scheduledAt = *opts.Hints.ScheduledAt
	}
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()
	ctrl, err := NewController(ctx, w.client, id, scheduledAt, ControllerOptions{
		Signature: stage.MapSignatureList(opts.Hints.Signature),
		RunnerId:  w.config.Connection.AgentID,
	})
	if err != nil {
		return nil, err
	}

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
	// TODO: retry?
	//jobs, err := w.clientSet.BatchV1().Jobs(ns).List(ctx, listOptions)
	//if err != nil {
	//	return nil, err
	//}
	//for _, job := range jobs.Items {
	//	if options.Finished != nil && *options.Finished != watchers.IsJobFinished(&job) {
	//		continue
	//	}
	//	if options.Root != nil && *options.Root != (job.Labels[constants.RootResourceIdLabelName] == job.Labels[constants.ResourceIdLabelName]) {
	//		continue
	//	}
	//	var cfg testworkflowconfig.InternalConfig
	//	err = json.Unmarshal([]byte(job.Spec.Template.Annotations[constants.InternalAnnotationName]), &cfg)
	//	if err != nil {
	//		log.DefaultLogger.Warnw("detected execution job that have invalid internal configuration", "name", job.Name, "namespace", job.Namespace, "error", err)
	//		continue
	//	}
	//	if options.OrganizationId != "" && options.OrganizationId != cfg.Execution.OrganizationId {
	//		continue
	//	}
	//	if options.EnvironmentId != "" && options.EnvironmentId != cfg.Execution.EnvironmentId {
	//		continue
	//	}
	//	list = append(list, executionworkertypes.ListResultItem{
	//		Execution: cfg.Execution,
	//		Workflow:  cfg.Workflow,
	//		Resource:  cfg.Resource,
	//		Namespace: job.Namespace,
	//	})
	//}
	return list, nil
}

func (w *worker) Abort(ctx context.Context, id string, options executionworkertypes.DestroyOptions) error {
	return w.Destroy(ctx, id, options)
}

func (w *worker) Destroy(ctx context.Context, id string, options executionworkertypes.DestroyOptions) (err error) {
	// List all the containers
	containers, err := w.client.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "label",
			Value: fmt.Sprintf("testkube.io/root=%s", id),
		}),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	// List all the volumes
	volumes, err := w.client.VolumeList(ctx, volume2.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "label",
			Value: fmt.Sprintf("testkube.io/root=%s", id),
		}),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list volumes")
	}

	errs := make([]error, 0)
	var errMu sync.Mutex

	// Delete all the containers
	wg := sync.WaitGroup{}
	wg.Add(len(containers))
	for _, cn := range containers {
		go func(cn types.Container) {
			err := w.client.ContainerRemove(ctx, cn.ID, container.RemoveOptions{
				Force: true,
			})
			if err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
			wg.Done()
		}(cn)
	}
	wg.Wait()

	// Delete all the volumes
	wg = sync.WaitGroup{}
	wg.Add(len(volumes.Volumes))
	for _, v := range volumes.Volumes {
		go func(v *volume2.Volume) {
			err := w.client.VolumeRemove(ctx, v.Name, true)
			if err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
			wg.Done()
		}(v)
	}
	wg.Wait()

	return errors2.Join(errs...)
}

func (w *worker) DestroyGroup(ctx context.Context, groupId string, options executionworkertypes.DestroyOptions) error {
	panic("not implemented")
	//if options.Namespace != "" {
	//	return controller.CleanupGroup(ctx, w.clientSet, options.Namespace, groupId)
	//}
	//
	//// Delete group resources in all known namespaces
	//errs := make([]error, 0)
	//for ns := range w.config.Cluster.Namespaces {
	//	err := w.DestroyGroup(ctx, groupId, executionworkertypes.DestroyOptions{Namespace: ns})
	//	if err != nil {
	//		errs = append(errs, err)
	//	}
	//}
	//return errors2.Join(errs...)
}

func (w *worker) Pause(ctx context.Context, id string, options executionworkertypes.ControlOptions) (err error) {
	panic("not implemented")
	//podIp, err := w.registry.GetPodIP(ctx, id)
	//if err != nil {
	//	return err
	//} else if podIp == "" {
	//	return registry.ErrPodIpNotAssigned
	//}
	//
	//// TODO: Move implementation there
	//return controller.Pause(ctx, podIp)
}

func (w *worker) Resume(ctx context.Context, id string, options executionworkertypes.ControlOptions) (err error) {
	panic("not implemented")
	//podIp, err := w.registry.GetPodIP(ctx, id)
	//if err != nil {
	//	return err
	//} else if podIp == "" {
	//	return registry.ErrPodIpNotAssigned
	//}
	//
	//// TODO: Move implementation there
	//return controller.Resume(ctx, podIp)
}

// TODO: consider status channel (?)
func (w *worker) ResumeMany(ctx context.Context, ids []string, options executionworkertypes.ControlOptions) (errs []executionworkertypes.IdentifiableError) {
	panic("not implemented")
	//ips := make(map[string]string, len(ids))
	//
	//// Try to obtain IPs
	//// TODO: concurrent operations (or single list operation)
	//for _, id := range ids {
	//	podIp, err := w.registry.GetPodIP(ctx, id)
	//	if err != nil {
	//		errs = append(errs, executionworkertypes.IdentifiableError{Id: id, Error: err})
	//	} else if podIp == "" {
	//		errs = append(errs, executionworkertypes.IdentifiableError{Id: id, Error: registry.ErrPodIpNotAssigned})
	//	} else {
	//		ips[id] = podIp
	//	}
	//}
	//
	//// Finish early when there are no IPs
	//if len(ips) == 0 {
	//	return errs
	//}
	//
	//// Initialize counters and synchronisation for waiting
	//var wg sync.WaitGroup
	//var mu sync.Mutex
	//cond := sync.NewCond(&mu)
	//counter := atomic.Int32{}
	//ready := func() {
	//	v := counter.Add(1)
	//	if v < int32(len(ips)) {
	//		cond.Wait()
	//	} else {
	//		cond.Broadcast()
	//	}
	//}
	//
	//// Create client connection and send to all of them
	//wg.Add(len(ips))
	//var errsMu sync.Mutex
	//for id, podIp := range ips {
	//	go func(id, address string) {
	//		cond.L.Lock()
	//		defer cond.L.Unlock()
	//
	//		client, err := control.NewClient(context.Background(), address, initconstants.ControlServerPort)
	//		ready()
	//		defer func() {
	//			if client != nil {
	//				client.Close()
	//			}
	//			wg.Done()
	//		}()
	//
	//		// Fast-track: immediate success
	//		if err == nil {
	//			err = client.Resume()
	//			if err == nil {
	//				return
	//			}
	//			log.DefaultLogger.Warnw("failed to resume, retrying...", "id", id, "address", address, "error", err)
	//		}
	//
	//		// Retrying mechanism
	//		for i := 0; i < 6; i++ {
	//			if client != nil {
	//				client.Close()
	//			}
	//			client, err = control.NewClient(context.Background(), address, initconstants.ControlServerPort)
	//			if err == nil {
	//				err = client.Resume()
	//				if err == nil {
	//					return
	//				}
	//			}
	//			log.DefaultLogger.Warnw("failed to resume, retrying...", "id", id, "address", address, "error", err)
	//			time.Sleep(ResumeRetryOnFailureDelay)
	//		}
	//
	//		// Total failure while retrying
	//		log.DefaultLogger.Errorw("failed to resume, maximum retries reached.", "id", id, "address", address, "error", err)
	//		errsMu.Lock()
	//		errs = append(errs, executionworkertypes.IdentifiableError{Id: id, Error: err})
	//		errsMu.Unlock()
	//	}(id, podIp)
	//}
	//wg.Wait()
	//
	//return errs
}
