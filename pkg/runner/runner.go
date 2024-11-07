package runner

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/app/api/metrics"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event"
	"github.com/kubeshop/testkube/pkg/log"
	testworkflows2 "github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/testworkflows"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/executionworkertypes"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/registry"
	"github.com/kubeshop/testkube/pkg/version"
)

//go:generate mockgen -destination=./mock_runner.go -package=runner "github.com/kubeshop/testkube/pkg/runner" Runner
type Runner interface {
	Monitor(ctx context.Context, id string) error
	Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher
	Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error)
	Pause(id string) error
	Resume(id string) error
	Abort(id string) error
}

type runner struct {
	worker               executionworkertypes.Worker
	outputRepository     testworkflow.OutputRepository
	executionsRepository testworkflow.Repository
	configRepository     configRepo.Repository
	emitter              *event.Emitter
	metrics              metrics.Metrics
	dashboardURI         string
	storageSkipVerify    bool

	watching sync.Map
}

func New(
	worker executionworkertypes.Worker,
	outputRepository testworkflow.OutputRepository,
	executionsRepository testworkflow.Repository,
	configRepository configRepo.Repository,
	emitter *event.Emitter,
	metrics metrics.Metrics,
	dashboardURI string,
	storageSkipVerify bool,
) Runner {
	return &runner{
		worker:               worker,
		outputRepository:     outputRepository,
		executionsRepository: executionsRepository,
		configRepository:     configRepository,
		emitter:              emitter,
		metrics:              metrics,
		dashboardURI:         dashboardURI,
		storageSkipVerify:    storageSkipVerify,
	}
}

// TODO: Update TestWorkflowExecution object in Kubernetes
func (r *runner) monitor(ctx context.Context, execution testkube.TestWorkflowExecution) error {
	defer r.watching.Delete(execution.Id)

	var notifications executionworkertypes.NotificationsWatcher
	for i := 0; i < 10; i++ {
		notifications = r.worker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{})
		if notifications.Err() == nil {
			break
		}
		if errors.Is(notifications.Err(), registry.ErrResourceNotFound) {
			// TODO: should it mark as job was aborted then?
			return registry.ErrResourceNotFound
		}
		time.Sleep(500 * time.Millisecond)
	}
	if notifications.Err() != nil {
		return errors.Wrapf(notifications.Err(), "failed to listen for '%s' execution notifications", execution.Id)
	}

	logs, err := NewExecutionLogsWriter(r.outputRepository, execution.Id, execution.Workflow.Name, r.storageSkipVerify)
	if err != nil {
		return err
	}
	saver, err := NewExecutionSaver(ctx, r.executionsRepository, execution.Id, logs)
	if err != nil {
		return err
	}
	defer logs.Cleanup()

	currentRef := ""
	var lastResult *testkube.TestWorkflowResult
	for n := range notifications.Channel() {
		if n.Temporary {
			continue
		}

		if n.Output != nil {
			saver.AppendOutput(*n.Output)
		} else if n.Result != nil {
			lastResult = n.Result
			// Don't send final result until everything is finished
			if n.Result.IsFinished() {
				continue
			}
			saver.UpdateResult(*n.Result)
		} else {
			if n.Ref != currentRef && n.Ref != "" {
				currentRef = n.Ref
				err = logs.WriteStart(n.Ref)
				if err != nil {
					// FIXME: what to do then?
					panic("logs write ref error")
				}
			}
			_, err = logs.Write([]byte(n.Log))
			if err != nil {
				// FIXME: what to do then?
				panic("logs write error")
			}
		}
	}

	// Ignore further monitoring if it has been canceled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if notifications.Err() != nil {
		log.DefaultLogger.Errorw("error from TestWorkflow watcher", "id", execution.Id, "error", notifications.Err())
	}

	if lastResult == nil || !lastResult.IsFinished() {
		log.DefaultLogger.Errorw("not finished TestWorkflow result received, trying to recover...", "id", execution.Id)
		watcher := r.worker.Notifications(ctx, execution.Id, executionworkertypes.NotificationsOptions{
			NoFollow: true,
		})
		if watcher.Err() == nil {
			for n := range watcher.Channel() {
				if n.Result != nil {
					lastResult = n.Result
				}
			}
		}

		if lastResult == nil {
			lastResult = execution.Result
		}
		if !lastResult.IsFinished() {
			log.DefaultLogger.Errorw("failed to recover TestWorkflow result, marking as fatal error...", "id", execution.Id)
			lastResult.Fatal(err, true, time.Now())
		}
	}

	for i := 0; i < 100; i++ {
		err = saver.End(ctx, *lastResult)
		if err == nil {
			break
		}
		log.DefaultLogger.Warnw("failed to save execution data", "id", execution.Id, "error", err)
		time.Sleep(time.Duration(i/10) * 500 * time.Millisecond)
	}

	// Handle fatal error
	if err != nil {
		log.DefaultLogger.Errorw("failed to save execution data", "id", execution.Id, "error", err)
		return errors.Wrapf(err, "failed to save execution '%s' data", execution.Id)
	}

	// Try to substitute execution data
	execution.Output = nil
	execution.Result = lastResult
	execution.StatusAt = lastResult.FinishedAt

	// Emit data
	r.metrics.IncAndObserveExecuteTestWorkflow(execution, r.dashboardURI)
	r.sendRunWorkflowTelemetry(ctx, testworkflows2.MapAPIToKube(execution.ResolvedWorkflow), &execution)
	if lastResult.IsPassed() {
		r.emitter.Notify(testkube.NewEventEndTestWorkflowSuccess(&execution))
	} else if lastResult.IsAborted() {
		r.emitter.Notify(testkube.NewEventEndTestWorkflowAborted(&execution))
	} else {
		r.emitter.Notify(testkube.NewEventEndTestWorkflowFailed(&execution))
	}

	err = r.worker.Destroy(context.Background(), execution.Id, executionworkertypes.DestroyOptions{})
	if err != nil {
		// TODO: what to do on error?
		log.DefaultLogger.Errorw("failed to cleanup TestWorkflow resources", "id", execution.Id, "error", err)
	}

	return nil
}

func (r *runner) Monitor(ctx context.Context, id string) error {
	ctx, ctxCancel := context.WithCancel(ctx)
	defer ctxCancel()

	// Check if there is any monitor attached already
	r.watching.LoadOrStore(id, false)
	if !r.watching.CompareAndSwap(id, false, true) {
		return nil
	}

	// Load the execution TODO: retry?
	execution, err := r.executionsRepository.Get(ctx, id)
	if err != nil {
		return err
	}

	return r.monitor(ctx, execution)
}

func (r *runner) Notifications(ctx context.Context, id string) executionworkertypes.NotificationsWatcher {
	return r.worker.Notifications(ctx, id, executionworkertypes.NotificationsOptions{})
}

func (r *runner) Execute(request executionworkertypes.ExecuteRequest) (*executionworkertypes.ExecuteResult, error) {
	return r.worker.Execute(context.Background(), request)
}

func (r *runner) Pause(id string) error {
	return r.worker.Pause(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Resume(id string) error {
	return r.worker.Resume(context.Background(), id, executionworkertypes.ControlOptions{})
}

func (r *runner) Abort(id string) error {
	return r.worker.Abort(context.Background(), id, executionworkertypes.DestroyOptions{})
}

func (r *runner) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := r.configRepository.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	properties := make(map[string]any)
	properties["name"] = workflow.Name
	stats := stepStats{
		imagesUsed:    make(map[string]struct{}),
		templatesUsed: make(map[string]struct{}),
	}

	var isSample bool
	if workflow.Labels != nil && workflow.Labels["docs"] == "example" && strings.HasSuffix(workflow.Name, "-sample") {
		isSample = true
	} else {
		isSample = false
	}

	spec := workflow.Spec
	for _, step := range spec.Steps {
		stats.Merge(getStepInfo(step))
	}
	if spec.Container != nil {
		stats.imagesUsed[spec.Container.Image] = struct{}{}
	}
	if len(spec.Services) != 0 {
		stats.hasServices = true
	}
	if len(spec.Use) > 0 {
		stats.hasTemplate = true
		for _, tmpl := range spec.Use {
			stats.templatesUsed[tmpl.Name] = struct{}{}
		}
	}

	var images []string
	for image := range stats.imagesUsed {
		if image == "" {
			continue
		}
		images = append(images, image)
	}

	var templates []string
	for t := range stats.templatesUsed {
		if t == "" {
			continue
		}
		templates = append(templates, t)
	}
	var (
		status     string
		durationMs int32
	)
	if execution.Result != nil {
		if execution.Result.Status != nil {
			status = string(*execution.Result.Status)
		}
		durationMs = execution.Result.DurationMs
	}

	out, err := telemetry.SendRunWorkflowEvent("testkube_api_run_test_workflow", telemetry.RunWorkflowParams{
		RunParams: telemetry.RunParams{
			AppVersion: version.Version,
			DataSource: testworkflows.GetDataSource(workflow.Spec.Content),
			Host:       testworkflows.GetHostname(),
			ClusterID:  testworkflows.GetClusterID(ctx, r.configRepository),
			DurationMs: durationMs,
			Status:     status,
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(stats.numSteps),
			TestWorkflowExecuteCount: int32(stats.numExecute),
			TestWorkflowImage:        testworkflows.GetImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed: stats.hasArtifacts,
			TestWorkflowParallelUsed: stats.hasParallel,
			TestWorkflowMatrixUsed:   stats.hasMatrix,
			TestWorkflowServicesUsed: stats.hasServices,
			TestWorkflowTemplateUsed: stats.hasTemplate,
			TestWorkflowIsSample:     isSample,
			TestWorkflowTemplates:    templates,
			TestWorkflowImages:       images,
			TestWorkflowKubeshopGitURI: testworkflows.IsKubeshopGitURI(workflow.Spec.Content) ||
				testworkflows.HasWorkflowStepLike(workflow.Spec, testworkflows.HasKubeshopGitURI),
		},
	})

	if err != nil {
		log.DefaultLogger.Debugw("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugw("sending run test workflow telemetry event", "output", out)
	}
}
