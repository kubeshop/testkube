package testworkflowexecutor

import (
	"context"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/testworkflows"
	"github.com/kubeshop/testkube/pkg/version"
)

type stepStats struct {
	numSteps      int
	numExecute    int
	hasArtifacts  bool
	hasMatrix     bool
	hasParallel   bool
	hasTemplate   bool
	hasServices   bool
	imagesUsed    map[string]struct{}
	templatesUsed map[string]struct{}
}

func (ss *stepStats) Merge(stats *stepStats) {
	ss.numSteps += stats.numSteps
	ss.numExecute += stats.numExecute

	if stats.hasArtifacts {
		ss.hasArtifacts = true
	}
	if stats.hasMatrix {
		ss.hasMatrix = true
	}
	if stats.hasParallel {
		ss.hasParallel = true
	}
	if stats.hasServices {
		ss.hasServices = true
	}
	if stats.hasTemplate {
		ss.hasTemplate = true
	}
	for image := range stats.imagesUsed {
		ss.imagesUsed[image] = struct{}{}
	}
	for tmpl := range stats.templatesUsed {
		ss.templatesUsed[tmpl] = struct{}{}
	}
}

func getStepInfo(step testworkflowsv1.Step) *stepStats {
	res := &stepStats{
		imagesUsed:    make(map[string]struct{}),
		templatesUsed: make(map[string]struct{}),
	}
	if step.Execute != nil {
		res.numExecute++
	}
	if step.Artifacts != nil {
		res.hasArtifacts = true
	}
	if len(step.Use) > 0 {
		res.hasTemplate = true
		for _, tmpl := range step.Use {
			res.templatesUsed[tmpl.Name] = struct{}{}
		}
	}
	if step.Template != nil {
		res.hasTemplate = true
		res.templatesUsed[step.Template.Name] = struct{}{}
	}
	if len(step.Services) > 0 {
		res.hasServices = true
	}

	if step.Run != nil && step.Run.Image != "" {
		res.imagesUsed[step.Run.Image] = struct{}{}
	}
	if step.Container != nil && step.Container.Image != "" {
		res.imagesUsed[step.Container.Image] = struct{}{}
	}

	for _, step := range step.Steps {
		res.Merge(getStepInfo(step))
	}

	if step.Parallel != nil {
		res.hasParallel = true

		if len(step.Parallel.Matrix) != 0 {
			res.hasMatrix = true
		}
		if step.Parallel.Artifacts != nil {
			res.hasArtifacts = true
		}
		if step.Parallel.Execute != nil {
			res.numExecute++
		}
		if len(step.Parallel.Use) > 0 {
			res.hasTemplate = true
			for _, tmpl := range step.Parallel.Use {
				res.templatesUsed[tmpl.Name] = struct{}{}
			}
		}
		if step.Parallel.Template != nil {
			res.hasTemplate = true
			res.templatesUsed[step.Parallel.Template.Name] = struct{}{}
		}

		if len(step.Parallel.Services) > 0 {
			res.hasServices = true
		}

		if step.Parallel.Run != nil && step.Parallel.Run.Image != "" {
			res.imagesUsed[step.Parallel.Run.Image] = struct{}{}
		}
		if step.Parallel.Container != nil && step.Parallel.Container.Image != "" {
			res.imagesUsed[step.Parallel.Container.Image] = struct{}{}
		}

		for _, step := range step.Parallel.Steps {
			res.Merge(getStepInfo(step))
		}
	}

	return res
}

func (e *executor) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow, execution *testkube.TestWorkflowExecution) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := e.configMap.GetTelemetryEnabled(ctx)
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
			ClusterID:  testworkflows.GetClusterID(ctx, e.configMap),
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
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}
