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

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: getDataSource(workflow.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(workflow.Spec.Steps)),
			TestWorkflowTemplateUsed:   len(workflow.Spec.Use) != 0,
			TestWorkflowImage:          getImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed:   hasArtifacts(workflow.Spec.Steps),
			TestWorkflowKubeshopGitURI: isKubeshopGitURI(workflow.Spec.Content),
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

	hasArtifacts := false
	for _, step := range template.Spec.Steps {
		if step.Artifacts != nil {
			hasArtifacts = true
			break
		}
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow_template", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: getDataSource(template.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(template.Spec.Steps)),
			TestWorkflowImage:          getImage(template.Spec.Container),
			TestWorkflowArtifactUsed:   hasArtifacts,
			TestWorkflowKubeshopGitURI: isKubeshopGitURI(template.Spec.Content),
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

	out, err := telemetry.SendRunWorkflowEvent("testkube_api_run_test_workflow", telemetry.RunWorkflowParams{
		RunParams: telemetry.RunParams{
			AppVersion: version.Version,
			DataSource: getDataSource(workflow.Spec.Content),
			Host:       getHostname(),
			ClusterID:  s.getClusterID(ctx),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:          int32(len(workflow.Spec.Steps)),
			TestWorkflowImage:          getImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed:   hasArtifacts(workflow.Spec.Steps),
			TestWorkflowKubeshopGitURI: isKubeshopGitURI(workflow.Spec.Content),
		},
	})

	if err != nil {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending run test workflow telemetry event", "output", out)
	}
}

func (s *apiTCL) getClusterID(ctx context.Context) string {
	clusterID, err := s.configMap.GetUniqueClusterId(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting cluster id error", "error", err)
		return ""
	}
	return clusterID
}

func getImage(container *testworkflowsv1.ContainerConfig) string {
	if container != nil {
		return container.Image
	}
	return ""
}

func hasArtifacts(steps []testworkflowsv1.Step) bool {
	for _, step := range steps {
		if step.Artifacts != nil {
			return true
		}
	}
	return false
}

func isKubeshopGitURI(content *testworkflowsv1.Content) bool {
	if content != nil && content.Git != nil && strings.Contains(content.Git.Uri, "kubeshop") {
		return true
	}
	return false
}

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

func getHostname() string {
	host, err := os.Hostname()
	if err != nil {
		log.DefaultLogger.Debugf("getting hostname error", "hostname", host, "error", err)
		return ""
	}
	return host
}
