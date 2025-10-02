package testworkflowexecutiontelemetry

import (
	"context"
	"fmt"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/mapper/testworkflows"
	"github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/version"
)

var _ common.Listener = (*testWorkflowExecutionTelemetryListener)(nil)

// Send telemetry events based on the Test Workflow Execution status changes
func NewListener(ctx context.Context, configRepository config.Repository) *testWorkflowExecutionTelemetryListener {
	return &testWorkflowExecutionTelemetryListener{
		ctx:              ctx,
		configRepository: configRepository,
	}
}

type testWorkflowExecutionTelemetryListener struct {
	ctx              context.Context
	configRepository config.Repository
}

func (l *testWorkflowExecutionTelemetryListener) Name() string {
	return "TestWorkflowExecutionTelemetry"
}

func (l *testWorkflowExecutionTelemetryListener) Selector() string {
	return ""
}

func (l *testWorkflowExecutionTelemetryListener) Kind() string {
	return "TestWorkflowExecutionTelemetry"
}

func (l *testWorkflowExecutionTelemetryListener) Events() []testkube.EventType {
	return []testkube.EventType{
		testkube.QUEUE_TESTWORKFLOW_EventType,
		testkube.START_TESTWORKFLOW_EventType,
		testkube.END_TESTWORKFLOW_SUCCESS_EventType,
		testkube.END_TESTWORKFLOW_FAILED_EventType,
		testkube.END_TESTWORKFLOW_ABORTED_EventType,
	}
}

func (l *testWorkflowExecutionTelemetryListener) Metadata() map[string]string {
	return map[string]string{
		"name":     l.Name(),
		"events":   fmt.Sprintf("%v", l.Events()),
		"selector": l.Selector(),
	}
}

func (l *testWorkflowExecutionTelemetryListener) Notify(event testkube.Event) testkube.EventResult {
	if event.TestWorkflowExecution == nil {
		return testkube.NewSuccessEventResult(event.Id, "ignored")
	}

	if *event.Type_ == testkube.END_TESTWORKFLOW_ABORTED_EventType ||
		*event.Type_ == testkube.END_TESTWORKFLOW_FAILED_EventType ||
		*event.Type_ == testkube.END_TESTWORKFLOW_SUCCESS_EventType {
		l.sendRunWorkflowTelemetry(context.Background(), testworkflows.MapAPIToKube(event.TestWorkflowExecution.ResolvedWorkflow), event.TestWorkflowExecution)
	}

	return testkube.NewSuccessEventResult(event.Id, "monitored")
}

func (l *testWorkflowExecutionTelemetryListener) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := l.configRepository.GetTelemetryEnabled(ctx)
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
			DataSource: GetDataSource(workflow.Spec.Content),
			Host:       GetHostname(),
			ClusterID:  GetClusterID(ctx, l.configRepository),
			DurationMs: durationMs,
			Status:     status,
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(stats.numSteps),
			TestWorkflowExecuteCount: int32(stats.numExecute),
			TestWorkflowImage:        GetImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed: stats.hasArtifacts,
			TestWorkflowParallelUsed: stats.hasParallel,
			TestWorkflowMatrixUsed:   stats.hasMatrix,
			TestWorkflowServicesUsed: stats.hasServices,
			TestWorkflowTemplateUsed: stats.hasTemplate,
			TestWorkflowIsSample:     isSample,
			TestWorkflowTemplates:    templates,
			TestWorkflowImages:       images,
			TestWorkflowKubeshopGitURI: IsKubeshopGitURI(workflow.Spec.Content) ||
				HasWorkflowStepLike(workflow.Spec, HasKubeshopGitURI),
		},
	})

	if err != nil {
		log.DefaultLogger.Debugw("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugw("sending run test workflow telemetry event", "output", out)
	}
}
