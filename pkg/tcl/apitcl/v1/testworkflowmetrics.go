package v1

import (
	"context"
	"os"
	"strings"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/version"
)

func (s *apiTCL) sendCreateWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	allSteps := append(workflow.Spec.Steps, workflow.Spec.Setup...)
	allSteps = append(allSteps, workflow.Spec.After...)

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: getDataSource(workflow.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(allSteps)),
			TestWorkflowTemplateUsed:   len(workflow.Spec.Use) != 0,
			TestWorkflowImage:          getImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed:   hasArtifacts(allSteps),
			TestWorkflowKubeshopGitURI: hasKubeshopGitURI(workflow.Spec),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event", "output", out)
	}
}

func (s *apiTCL) sendCreateWorkflowTemplateTelemetry(ctx context.Context, template *testworkflowsv1.TestWorkflowTemplate) {
	if template == nil {
		log.DefaultLogger.Debug("empty template passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	allSteps := append(template.Spec.Steps, template.Spec.Setup...)
	allSteps = append(allSteps, template.Spec.After...)

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow_template", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: getDataSource(template.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(allSteps)),
			TestWorkflowImage:          getImage(template.Spec.Container),
			TestWorkflowArtifactUsed:   hasTemplateArtifacts(template.Spec.Steps),
			TestWorkflowKubeshopGitURI: hasTemplateKubeshopGitURI(template.Spec),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event", "output", out)
	}
}

func (s *apiTCL) sendRunWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.configMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}
	allSteps := append(workflow.Spec.Steps, workflow.Spec.Setup...)
	allSteps = append(allSteps, workflow.Spec.After...)

	out, err := telemetry.SendRunWorkflowEvent("testkube_api_run_test_workflow", telemetry.RunWorkflowParams{
		RunParams: telemetry.RunParams{
			AppVersion: version.Version,
			DataSource: getDataSource(workflow.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(allSteps)),
			TestWorkflowImage:          getImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed:   hasArtifacts(allSteps),
			TestWorkflowKubeshopGitURI: hasKubeshopGitURI(workflow.Spec),
		},
	})

	if err != nil {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}

// getClusterID returns the cluster id
func (s *apiTCL) getClusterID(ctx context.Context) string {
	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
		return ""
	}
	return clusterID
}

// getImage returns the image of the container
func getImage(container *testworkflowsv1.ContainerConfig) string {
	if container != nil {
		return container.Image
	}
	return ""
}

// hasArtifacts checks if the test workflow steps have artifacts
func hasArtifacts(steps []testworkflowsv1.Step) bool {
	for _, step := range steps {
		if step.Artifacts != nil {
			return true
		}
		if hasArtifacts(step.Setup) {
			return true
		}
		if hasArtifacts(step.Steps) {
			return true
		}
	}
	return false
}

// hasTemplateArtifacts checks if the test workflow steps have artifacts
func hasTemplateArtifacts(steps []testworkflowsv1.IndependentStep) bool {
	for _, step := range steps {
		if step.Artifacts != nil {
			return true
		}
		if hasTemplateArtifacts(step.Setup) {
			return true
		}
		if hasTemplateArtifacts(step.Steps) {
			return true
		}
	}
	return false
}

// hasKubeshopGitURI checks if the test workflow spec has a git URI that contains "kubeshop"
func hasKubeshopGitURI(spec testworkflowsv1.TestWorkflowSpec) bool {
	if isKubeshopGitURI(spec.Content) {
		return true
	}

	for _, step := range spec.Steps {
		if hasStepKubeshopGitURI(step) {
			return true
		}
	}
	for _, step := range spec.Setup {
		if hasStepKubeshopGitURI(step) {
			return true
		}
	}
	for _, step := range spec.After {
		if hasStepKubeshopGitURI(step) {
			return true
		}
	}

	return false
}

// hasTemplateKubeshopGitURI checks if the test workflow spec has a git URI that contains "kubeshop"
func hasTemplateKubeshopGitURI(spec testworkflowsv1.TestWorkflowTemplateSpec) bool {
	if isKubeshopGitURI(spec.Content) {
		return true
	}

	for _, step := range spec.Steps {
		if hasTemplateStepKubeshopGitURI(step) {
			return true
		}
	}
	for _, step := range spec.Setup {
		if hasTemplateStepKubeshopGitURI(step) {
			return true
		}
	}
	for _, step := range spec.After {
		if hasTemplateStepKubeshopGitURI(step) {
			return true
		}
	}

	return false
}

// hasTemplateStepKubeshopGitURI checks if the step has a git URI that contains "kubeshop"
func hasTemplateStepKubeshopGitURI(step testworkflowsv1.IndependentStep) bool {
	for _, step := range step.Setup {
		if isKubeshopGitURI(step.Content) {
			return true
		}
		if hasTemplateStepKubeshopGitURI(step) {
			return true
		}
	}
	for _, step := range step.Steps {
		if isKubeshopGitURI(step.Content) {
			return true
		}
		if hasTemplateStepKubeshopGitURI(step) {
			return true
		}
	}
	return false
}

// hasStepKubeshopGitURI checks if the step has a git URI that contains "kubeshop"
func hasStepKubeshopGitURI(step testworkflowsv1.Step) bool {
	for _, step := range step.Setup {
		if isKubeshopGitURI(step.Content) {
			return true
		}
		if hasStepKubeshopGitURI(step) {
			return true
		}
	}
	for _, step := range step.Steps {
		if isKubeshopGitURI(step.Content) {
			return true
		}
		if hasStepKubeshopGitURI(step) {
			return true
		}
	}
	return false
}

// isKubeshopGitURI checks if the content has a git URI that contains "kubeshop"
func isKubeshopGitURI(content *testworkflowsv1.Content) bool {
	switch {
	case content == nil:
		return false
	case content.Git == nil:
		return false
	case strings.Contains(content.Git.Uri, "kubeshop"):
		return true
	default:
		return false
	}
}

// getDataSource returns the data source of the content
func getDataSource(content *testworkflowsv1.Content) string {
	var dataSource string
	if content != nil {
		if len(content.Files) != 0 {
			dataSource = "files"
		} else if content.Git != nil {
			dataSource = "git"
		}
	}
	return dataSource
}

// getHostname returns the hostname
func getHostname() string {
	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
		return ""
	}
	return host
}
