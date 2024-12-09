package v1

import (
	"context"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/event/kind/testworkflowexecutiontelemetry"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/version"
)

func (s *TestkubeAPI) sendCreateWorkflowTelemetry(ctx context.Context, workflow *testworkflowsv1.TestWorkflow) {
	if workflow == nil {
		log.DefaultLogger.Debug("empty workflow passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.ConfigMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: testworkflowexecutiontelemetry.GetDataSource(workflow.Spec.Content),
			Host:       testworkflowexecutiontelemetry.GetHostname(),
			ClusterID:  testworkflowexecutiontelemetry.GetClusterID(ctx, s.ConfigMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(workflow.Spec.Setup) + len(workflow.Spec.Steps) + len(workflow.Spec.After)),
			TestWorkflowTemplateUsed: len(workflow.Spec.Use) != 0,
			TestWorkflowImage:        testworkflowexecutiontelemetry.GetImage(workflow.Spec.Container),
			TestWorkflowArtifactUsed: testworkflowexecutiontelemetry.HasWorkflowStepLike(workflow.Spec, testworkflowexecutiontelemetry.HasArtifacts),
			TestWorkflowKubeshopGitURI: testworkflowexecutiontelemetry.IsKubeshopGitURI(workflow.Spec.Content) ||
				testworkflowexecutiontelemetry.HasWorkflowStepLike(workflow.Spec, testworkflowexecutiontelemetry.HasKubeshopGitURI),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow telemetry event", "output", out)
	}
}

func (s *TestkubeAPI) sendCreateWorkflowTemplateTelemetry(ctx context.Context, template *testworkflowsv1.TestWorkflowTemplate) {
	if template == nil {
		log.DefaultLogger.Debug("empty template passed to telemetry event")
		return
	}
	telemetryEnabled, err := s.ConfigMap.GetTelemetryEnabled(ctx)
	if err != nil {
		log.DefaultLogger.Debugf("getting telemetry enabled error", "error", err)
	}
	if !telemetryEnabled {
		return
	}

	out, err := telemetry.SendCreateWorkflowEvent("testkube_api_create_test_workflow_template", telemetry.CreateWorkflowParams{
		CreateParams: telemetry.CreateParams{
			AppVersion: version.Version,
			DataSource: testworkflowexecutiontelemetry.GetDataSource(template.Spec.Content),
			Host:       testworkflowexecutiontelemetry.GetHostname(),
			ClusterID:  testworkflowexecutiontelemetry.GetClusterID(ctx, s.ConfigMap),
		},
		WorkflowParams: telemetry.WorkflowParams{
			TestWorkflowSteps:        int32(len(template.Spec.Setup) + len(template.Spec.Steps) + len(template.Spec.After)),
			TestWorkflowImage:        testworkflowexecutiontelemetry.GetImage(template.Spec.Container),
			TestWorkflowArtifactUsed: testworkflowexecutiontelemetry.HasTemplateStepLike(template.Spec, testworkflowexecutiontelemetry.HasTemplateArtifacts),
			TestWorkflowKubeshopGitURI: testworkflowexecutiontelemetry.IsKubeshopGitURI(template.Spec.Content) ||
				testworkflowexecutiontelemetry.HasTemplateStepLike(template.Spec, testworkflowexecutiontelemetry.HasTemplateKubeshopGitURI),
		},
	})
	if err != nil {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event error", "error", err)
	} else {
		log.DefaultLogger.Debugf("sending create test workflow template telemetry event", "output", out)
	}
}
